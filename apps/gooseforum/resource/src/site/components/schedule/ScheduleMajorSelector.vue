<script setup lang="ts">
// 学期→年级→专业 选择器。任何一级变更都会清空已选/备选课程（防跨学期污染），
// 对齐上游 MajorInfo 交互语义。
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { AlertCircle, Check, ChevronDown, ChevronUp, Compass, Copy, ExternalLink, HelpCircle, Info, Link2, RotateCcw, Sparkles, X } from '@lucide/vue'
import {
  ComboboxAnchor,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxItemIndicator,
  ComboboxPortal,
  ComboboxRoot,
  ComboboxTrigger,
  ComboboxViewport,
  PopoverArrow,
  PopoverClose,
  PopoverContent,
  PopoverPortal,
  PopoverRoot,
  PopoverTrigger,
} from 'reka-ui'
import { useI18n } from 'vue-i18n'
import SiteSelect from '@/site/components/SiteSelect.vue'
import { queueFlashMessage } from '@/runtime/flash-message'
import { getPkCalendars, getPkGrades, getPkMajors } from '@/runtime/pk-api'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import type { PkCalendar, PkMajor } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

const props = withDefaults(
  defineProps<{
    collapsible?: boolean
  }>(),
  {
    collapsible: false,
  },
)

const emit = defineEmits<{
  'toggle-collapse': []
}>()

const calendars = ref<PkCalendar[]>([])
const grades = ref<number[]>([])
const majors = ref<PkMajor[]>([])
const loading = ref(false)
const error = ref('')
/** 初始化恢复期：抑制 watch 触发清空（避免误清刷新恢复的已选课程）。 */
let isRestoring = true

// 选择组件的 modelValue 为 string；选择值本地持有字符串，变更时写回 store。
const calendarValue = ref('')
const gradeValue = ref('')
const majorValue = ref('')

const copied = ref(false)
let copyResetTimer: ReturnType<typeof setTimeout> | undefined

const majorInfoApiUrl = computed(() => {
  const calId = store.state.majorSelected.calendarId ?? (calendars.value.length > 0 ? calendars.value[0].calendarId : 1)
  return `https://1.tongji.edu.cn/api/electionservice/student/getElecStudentInfo?calendarId=${calId}`
})

async function copyQueryUrl() {
  try {
    await navigator.clipboard.writeText(majorInfoApiUrl.value)
    copied.value = true
    queueFlashMessage(t('schedule.copyQueryUrlSuccess'), 'success')
    if (copyResetTimer) clearTimeout(copyResetTimer)
    copyResetTimer = setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    queueFlashMessage(t('schedule.copyQueryUrlSuccess'), 'info')
  }
}

const calendarOptions = computed(() =>
  calendars.value.map((c) => ({ value: String(c.calendarId), label: c.calendarName })),
)
const gradeOptions = computed(() => grades.value.map((g) => ({ value: String(g), label: String(g) })))
const majorOptions = computed(() => majors.value.map((m) => ({ value: m.code, label: m.name })))

function displayMajor(value: string | undefined) {
  return majorOptions.value.find((option) => option.value === value)?.label ?? ''
}

async function loadCalendars() {
  loading.value = true
  error.value = ''
  try {
    calendars.value = await getPkCalendars()
    // 回填 store 供课表周次定位/学期日期条消费（会话缓存，不持久化）。
    store.setCalendars(calendars.value)
    // 恢复持久化的学期选择（无则自动选第一个）。
    const restored = store.state.majorSelected.calendarId
    if (restored !== undefined && calendars.value.some((c) => c.calendarId === restored)) {
      calendarValue.value = String(restored)
    } else if (calendars.value.length > 0) {
      calendarValue.value = String(calendars.value[0].calendarId)
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  } finally {
    loading.value = false
  }
}

async function loadGrades(calendarId: number) {
  grades.value = []
  majors.value = []
  try {
    grades.value = await getPkGrades(calendarId)
    const restored = store.state.majorSelected.grade
    if (restored !== undefined && grades.value.includes(restored)) {
      gradeValue.value = String(restored)
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  }
}

async function loadMajors(grade: number, calendarId: number) {
  majors.value = []
  try {
    majors.value = await getPkMajors(grade, calendarId)
    const restored = store.state.majorSelected.major
    if (restored && majors.value.some((m) => m.code === restored)) {
      majorValue.value = restored
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  }
}

function resetSelection() {
  store.clearStagedAndSelectedCourses()
  // 换学期/年级/专业清空课程后立即持久化，避免刷新后旧课程从
  // localStorage 复活（与退课同一缺陷族）。
  store.solidify()
}

/**
 * 初始化恢复序列：从 localStorage 恢复学期→年级→专业选择。
 * 恢复期间抑制 watch（不触发清空已选课程），否则刷新后恢复的已选课程会被误清
 * （验收标准 5：刷新后已选课程从 localStorage 恢复）。
 */
async function restoreSelection() {
  isRestoring = true
  await loadCalendars()
  const restored = store.state.majorSelected
  const calendarId = calendarValue.value ? Number(calendarValue.value) : undefined
  // 首次访问（无已存选择）或已存学期失效回退时，isRestoring 抑制 watch，
  // 必须把最终选中的学期写回 store；否则后续选年级时 calendarId 为
  // undefined（首次）或旧学期（回退），专业与课程将按错误学期加载。
  const calendarChanged = calendarId !== undefined && calendarId !== restored.calendarId
  if (calendarChanged) {
    store.setMajorInfo({ calendarId, grade: undefined, major: undefined })
    // 旧学期已失效：清掉其课程缓存，防跨学期污染（对齐学期变更 watch 语义）。
    resetSelection()
  }
  if (calendarId !== undefined) {
    await loadGrades(calendarId)
    // 仅当学期未变（有效恢复）且年级/专业均恢复成功时加载专业；
    // 回退场景专业已清空，等用户重新选择（watch 使用已写回的 calendarId）。
    if (gradeValue.value && restored.major && !calendarChanged) {
      await loadMajors(Number(gradeValue.value), calendarId)
      if (!restored.majorName) {
        const found = displayMajor(restored.major)
        if (found) {
          store.setMajorInfo({ ...restored, majorName: found })
        }
      }
    }
  }
  isRestoring = false
}

// 学期变更：拉年级，清空年级/专业与课程缓存。
watch(calendarValue, (value) => {
  if (isRestoring) return
  store.setMajorInfo({ calendarId: value ? Number(value) : undefined, grade: undefined, major: undefined, majorName: undefined })
  resetSelection()
  if (value) {
    void loadGrades(Number(value))
  } else {
    grades.value = []
    gradeValue.value = ''
    majors.value = []
    majorValue.value = ''
  }
})

// 年级变更：拉专业，清空专业与课程缓存。
watch(gradeValue, (value) => {
  if (isRestoring) return
  const calendarId = store.state.majorSelected.calendarId
  store.setMajorInfo({ calendarId, grade: value ? Number(value) : undefined, major: undefined, majorName: undefined })
  resetSelection()
  if (value && calendarId !== undefined) {
    void loadMajors(Number(value), calendarId)
  } else {
    majors.value = []
    majorValue.value = ''
  }
})

// 专业变更：写回 store（课程缓存已由上游清空）。
watch(majorValue, (value) => {
  if (isRestoring) return
  const calendarId = store.state.majorSelected.calendarId
  const grade = store.state.majorSelected.grade
  const majorName = displayMajor(value)
  store.setMajorInfo({ calendarId, grade, major: value || undefined, majorName: majorName || undefined })
  resetSelection()
})

function clearCalendar() {
  calendarValue.value = ''
  gradeValue.value = ''
  majorValue.value = ''
  grades.value = []
  majors.value = []
  store.setMajorInfo({ calendarId: undefined, grade: undefined, major: undefined, majorName: undefined })
  resetSelection()
}

function clearGrade() {
  gradeValue.value = ''
  majorValue.value = ''
  majors.value = []
  const calendarId = store.state.majorSelected.calendarId
  store.setMajorInfo({ calendarId, grade: undefined, major: undefined, majorName: undefined })
  resetSelection()
}

function clearMajor() {
  majorValue.value = ''
  const calendarId = store.state.majorSelected.calendarId
  const grade = store.state.majorSelected.grade
  store.setMajorInfo({ calendarId, grade, major: undefined, majorName: undefined })
  resetSelection()
}

function clearAllSelection() {
  clearCalendar()
}

const hasSelection = computed(() => !!(calendarValue.value || gradeValue.value || majorValue.value))

/** 配置是否已全部完成（学期、年级、专业均已选定）。 */
const isConfigComplete = computed(() => !!(calendarValue.value && gradeValue.value && majorValue.value))

// ---- 专业代码指南 Popover Hover / Click 交互（高醒目度 + 触感克制） ----
const guideOpen = ref(false)
const guidePinned = ref(false)
let guideHoverTimer: ReturnType<typeof setTimeout> | null = null

function handleGuideMouseEnter() {
  if (guidePinned.value) return
  if (guideHoverTimer) clearTimeout(guideHoverTimer)
  guideHoverTimer = setTimeout(() => {
    guideOpen.value = true
  }, 120)
}

function handleGuideMouseLeave() {
  if (guidePinned.value) return
  if (guideHoverTimer) clearTimeout(guideHoverTimer)
  guideHoverTimer = setTimeout(() => {
    guideOpen.value = false
  }, 240)
}

function handleGuideContentMouseEnter() {
  if (guidePinned.value) return
  if (guideHoverTimer) {
    clearTimeout(guideHoverTimer)
    guideHoverTimer = null
  }
}

function handleGuideContentMouseLeave() {
  if (guidePinned.value) return
  if (guideHoverTimer) clearTimeout(guideHoverTimer)
  guideHoverTimer = setTimeout(() => {
    guideOpen.value = false
  }, 200)
}

function handleGuideTriggerClick() {
  if (guidePinned.value) {
    guidePinned.value = false
    guideOpen.value = false
  } else {
    guidePinned.value = true
    guideOpen.value = true
  }
}

watch(guideOpen, (isOpen) => {
  if (!isOpen) {
    guidePinned.value = false
    if (guideHoverTimer) {
      clearTimeout(guideHoverTimer)
      guideHoverTimer = null
    }
  }
})

onMounted(() => {
  void restoreSelection()
})

onBeforeUnmount(() => {
  if (guideHoverTimer) {
    clearTimeout(guideHoverTimer)
    guideHoverTimer = null
  }
})
</script>

<template>
  <div class="gf-panel rounded-2xl border border-line/70 p-4 shadow-[0_2px_10px_-4px_rgba(0,0,0,0.05)]">
    <div class="space-y-3">
      <!-- 第一行：学期（主要自适应宽度，长名称不被截断） + 年级（适度紧凑固定宽度） -->
      <div class="flex items-end gap-2.5">
        <label class="block min-w-0 flex-1">
          <div class="mb-1.5 flex items-center justify-between">
            <span class="block text-[12px] font-medium text-base-content/75">{{ t('schedule.calendar') }}</span>
            <button
              v-if="calendarValue"
              type="button"
              class="text-[11px] font-normal text-base-content/45 hover:text-error transition-colors cursor-pointer select-none"
              :title="t('schedule.clearCalendar')"
              :aria-label="t('schedule.clearCalendar')"
              @click.prevent="clearCalendar"
            >
              {{ t('schedule.clear') }}
            </button>
          </div>
          <SiteSelect
            v-model="calendarValue"
            :options="calendarOptions"
            :placeholder="t('schedule.selectPlaceholder')"
            :label="t('schedule.calendar')"
            :clearable="true"
          />
        </label>
        <label class="block w-24 shrink-0 sm:w-28">
          <div class="mb-1.5 flex items-center justify-between">
            <span class="block text-[12px] font-medium text-base-content/75">{{ t('schedule.grade') }}</span>
            <button
              v-if="gradeValue"
              type="button"
              class="text-[11px] font-normal text-base-content/45 hover:text-error transition-colors cursor-pointer select-none"
              :title="t('schedule.clearGrade')"
              :aria-label="t('schedule.clearGrade')"
              @click.prevent="clearGrade"
            >
              {{ t('schedule.clear') }}
            </button>
          </div>
          <SiteSelect
            v-model="gradeValue"
            :options="gradeOptions"
            :placeholder="t('schedule.selectPlaceholder')"
            :label="t('schedule.grade')"
            :clearable="true"
          />
        </label>
      </div>

      <!-- 第二行：专业（独占全宽，长专业名称完整展示） -->
      <div class="block">
        <div class="mb-1.5 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="text-[12px] font-medium text-base-content/80">{{ t('schedule.major') }}</span>
            <PopoverRoot v-model:open="guideOpen">
              <PopoverTrigger
                type="button"
                class="group inline-flex items-center gap-1 rounded-full border border-primary/30 bg-primary/10 px-2 py-0.5 text-[11px] font-medium text-primary hover:bg-primary/18 hover:border-primary/45 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 transition-all duration-150 cursor-help active:scale-[0.96] select-none"
                :class="{ 'bg-primary/20 border-primary/50 ring-1 ring-primary/25 shadow-xs': guideOpen }"
                :title="t('schedule.majorCodeHelp')"
                :aria-label="t('schedule.majorCodeHelp')"
                data-testid="schedule-major-code-trigger"
                @mouseenter="handleGuideMouseEnter"
                @mouseleave="handleGuideMouseLeave"
                @click="handleGuideTriggerClick"
              >
                <HelpCircle class="h-3 w-3 shrink-0 text-primary transition-transform duration-150 group-hover:scale-110" />
                <span class="leading-none">{{ t('schedule.majorCodeBadge') }}</span>
              </PopoverTrigger>
              <PopoverPortal>
                <PopoverContent
                  side="bottom"
                  align="start"
                  :side-offset="8"
                  :collision-padding="16"
                  class="z-[2200] w-[min(380px,calc(100vw-2rem))] rounded-2xl border border-line/80 bg-base-100/98 p-5 shadow-2xl backdrop-blur-xl outline-none text-xs text-base-content data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=top]:slide-in-from-bottom-2 duration-150 ease-out origin-[var(--reka-popper-transform-origin)]"
                  @mouseenter="handleGuideContentMouseEnter"
                  @mouseleave="handleGuideContentMouseLeave"
                >
                  <!-- 头部：标题与关闭按钮 -->
                  <div class="flex items-start justify-between gap-3 border-b border-line/50 pb-3">
                    <div class="flex items-center gap-2.5">
                      <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary shadow-xs">
                        <Compass class="h-4 w-4" />
                      </div>
                      <div>
                        <h3 class="text-sm font-bold text-base-content leading-none">
                          {{ t('schedule.majorCodeGuideTitle') }}
                        </h3>
                        <p class="mt-1 text-[11px] text-base-content/60">
                          {{ t('schedule.majorCodeGuideSubtitle') }}
                        </p>
                      </div>
                    </div>
                    <PopoverClose
                      type="button"
                      class="rounded-lg p-1 text-base-content/40 hover:text-base-content hover:bg-base-200 transition-colors cursor-pointer"
                      :aria-label="t('schedule.close')"
                    >
                      <X class="h-4 w-4" />
                    </PopoverClose>
                  </div>

                  <!-- 步骤流 -->
                  <div class="mt-3.5 space-y-3">
                    <!-- 步骤 1 -->
                    <div class="flex items-start gap-2.5">
                      <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-base-200 font-mono text-[10px] font-bold text-base-content/70">
                        1
                      </span>
                      <div class="min-w-0 flex-1">
                        <p class="text-[12px] font-medium text-base-content leading-snug">
                          {{ t('schedule.majorCodeStep1Title') }}
                        </p>
                        <div class="mt-1">
                          <a
                            href="https://1.tongji.edu.cn"
                            target="_blank"
                            rel="noopener noreferrer"
                            class="inline-flex items-center gap-1 rounded-md bg-base-200/80 px-2 py-0.5 text-[11px] font-medium text-primary hover:bg-primary/10 hover:underline transition-colors"
                          >
                            <span>{{ t('schedule.openTongji1System') }}</span>
                            <ExternalLink class="h-3 w-3" />
                          </a>
                        </div>
                      </div>
                    </div>

                    <!-- 步骤 2 -->
                    <div class="flex items-start gap-2.5">
                      <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-base-200 font-mono text-[10px] font-bold text-base-content/70">
                        2
                      </span>
                      <div class="min-w-0 flex-1 space-y-1.5">
                        <p class="text-[12px] font-medium text-base-content leading-snug">
                          {{ t('schedule.majorCodeStep2Title') }}
                        </p>
                        <!-- 整合式复制卡片：代码链接预览 + 一体化复制按钮 -->
                        <div class="flex items-center gap-2 rounded-xl border border-line/70 bg-base-200/50 p-1.5 pl-2.5 transition-colors focus-within:border-primary/50">
                          <Link2 class="h-3.5 w-3.5 shrink-0 text-base-content/40" />
                          <span
                            class="min-w-0 flex-1 truncate font-mono text-[11px] text-base-content/75 select-all"
                            :title="majorInfoApiUrl"
                          >
                            {{ majorInfoApiUrl }}
                          </span>
                          <button
                            type="button"
                            class="gf-button gf-button-sm gf-button-secondary shrink-0 gap-1 rounded-lg px-2.5 text-xs transition-transform active:scale-[0.96]"
                            @click="copyQueryUrl"
                          >
                            <Check v-if="copied" class="h-3.5 w-3.5 text-success" />
                            <Copy v-else class="h-3.5 w-3.5 text-base-content/60" />
                            <span :class="copied ? 'text-success font-medium' : ''">
                              {{ copied ? t('schedule.copied') : t('schedule.copy') }}
                            </span>
                          </button>
                        </div>
                      </div>
                    </div>

                    <!-- 步骤 3 -->
                    <div class="flex items-start gap-2.5">
                      <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-base-200 font-mono text-[10px] font-bold text-base-content/70">
                        3
                      </span>
                      <div class="min-w-0 flex-1 space-y-1.5">
                        <p class="text-[12px] font-medium text-base-content leading-snug">
                          {{ t('schedule.majorCodeStep3Title') }}
                        </p>
                        <div class="inline-flex items-center gap-1.5 rounded-lg border border-line/50 bg-base-200/60 px-2 py-1 font-mono text-[11px] text-base-content/80">
                          <span>"profession":</span>
                          <span class="rounded bg-primary/15 px-1.5 py-0.5 font-bold text-primary">"080901"</span>
                        </div>
                        <p class="text-[11px] text-base-content/60 leading-normal">
                          {{ t('schedule.majorCodeStep3Desc') }}
                        </p>
                      </div>
                    </div>

                    <!-- 步骤 4 -->
                    <div class="flex items-start gap-2.5">
                      <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-base-200 font-mono text-[10px] font-bold text-base-content/70">
                        4
                      </span>
                      <div class="min-w-0 flex-1">
                        <p class="text-[12px] font-medium text-base-content leading-snug">
                          {{ t('schedule.majorCodeStep4Title') }}
                        </p>
                      </div>
                    </div>
                  </div>

                  <!-- 底部提示：轻量警告与提示卡片 -->
                  <div class="mt-3.5 flex items-start gap-2 rounded-xl border border-warning/25 bg-warning/5 p-2.5 text-[11px] leading-relaxed text-base-content/70">
                    <AlertCircle class="h-3.5 w-3.5 shrink-0 text-warning mt-0.5" />
                    <div class="space-y-0.5">
                      <p>{{ t('schedule.majorCodeTip') }}</p>
                      <p class="text-base-content/50">{{ t('schedule.majorCodeWarn') }}</p>
                    </div>
                  </div>
                  <PopoverArrow class="fill-base-100 stroke-line/80" />
                </PopoverContent>
              </PopoverPortal>
            </PopoverRoot>
          </div>
          <div class="flex items-center gap-2">
            <span v-if="!gradeValue" class="text-[11px] font-normal text-base-content/45">
              {{ t('schedule.selectGradeFirst') }}
            </span>
            <button
              v-else-if="majorValue"
              type="button"
              class="text-[11px] font-normal text-base-content/45 hover:text-error transition-colors cursor-pointer select-none"
              :title="t('schedule.clearMajor')"
              :aria-label="t('schedule.clearMajor')"
              @click.prevent="clearMajor"
            >
              {{ t('schedule.clear') }}
            </button>
          </div>
        </div>
        <ComboboxRoot v-model="majorValue" :disabled="!gradeValue || majors.length === 0" open-on-focus open-on-click>
          <ComboboxAnchor class="relative block group">
            <ComboboxInput
              class="gf-input gf-input-md w-full pr-9 transition-colors duration-150 disabled:cursor-not-allowed disabled:bg-base-200/40 disabled:text-base-content/40"
              :display-value="displayMajor"
              :placeholder="!gradeValue ? t('schedule.selectGradeFirst') : t('schedule.selectSearchPlaceholder')"
              :aria-label="t('schedule.major')"
              :title="displayMajor(majorValue)"
              data-testid="schedule-major-combobox-input"
            />
            <div class="absolute inset-y-0 right-0 flex w-9 items-center justify-center">
              <button
                v-if="majorValue"
                type="button"
                class="hidden group-hover:flex h-6 w-6 items-center justify-center rounded-md text-base-content/45 hover:text-error hover:bg-base-200 transition-colors cursor-pointer"
                :title="t('schedule.clearMajor')"
                :aria-label="t('schedule.clearMajor')"
                @click.stop.prevent="clearMajor"
              >
                <X class="h-3.5 w-3.5" />
              </button>
              <ComboboxTrigger
                class="flex h-full w-full items-center justify-center text-base-content/45 hover:text-base-content transition-colors cursor-pointer"
                :class="{ 'group-hover:hidden': !!majorValue }"
                :aria-label="t('schedule.major')"
              >
                <ChevronDown class="h-4 w-4 transition-transform duration-200 group-data-[state=open]:rotate-180" />
              </ComboboxTrigger>
            </div>
          </ComboboxAnchor>

          <ComboboxPortal>
            <ComboboxContent
              class="gf-menu-surface z-[2100] min-w-[var(--reka-combobox-trigger-width)] max-w-[min(28rem,calc(100vw-2rem))] overflow-hidden rounded-xl border border-line/80 bg-base-100/98 p-1 shadow-2xl backdrop-blur-md outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-1.5 data-[side=top]:slide-in-from-bottom-1.5 duration-150 ease-out origin-[var(--reka-popper-transform-origin)]"
              position="popper"
              :side-offset="6"
              :collision-padding="8"
              align="start"
              :body-lock="false"
              :disable-outside-pointer-events="false"
            >
              <ComboboxEmpty class="px-2.5 py-3 text-center text-sm text-base-content/55" role="status">
                {{ t('schedule.selectSearchEmpty') }}
              </ComboboxEmpty>
              <ComboboxViewport class="gf-scrollbar-thin max-h-[min(18rem,calc(var(--reka-popper-available-height,18rem)-1.5rem))] overflow-y-auto overscroll-contain">
                <ComboboxItem
                  v-for="option in majorOptions"
                  :key="option.value"
                  :value="option.value"
                  :text-value="option.label"
                  :title="option.label"
                  class="flex h-9 w-full cursor-pointer items-center gap-2 rounded-lg px-2.5 text-left text-sm font-medium text-base-content outline-none select-none transition-colors duration-150 hover:bg-base-200/80 data-[highlighted]:bg-primary/10 data-[highlighted]:text-primary"
                  :class="option.value === majorValue ? 'bg-primary/10 text-primary font-semibold' : ''"
                >
                  <span class="min-w-0 flex-1 truncate">{{ option.label }}</span>
                  <ComboboxItemIndicator>
                    <Check class="h-4 w-4 shrink-0 text-primary" />
                  </ComboboxItemIndicator>
                </ComboboxItem>
              </ComboboxViewport>
            </ComboboxContent>
          </ComboboxPortal>
        </ComboboxRoot>
      </div>
    </div>

    <p v-if="error" class="mt-3 rounded-lg border border-error/25 bg-error/10 px-3 py-2 text-sm text-error">
      {{ error }}
    </p>
    <!-- 学期下拉为空：区分「无数据」与「加载失败」，引导管理员同步（issue #248）。 -->
    <p
      v-else-if="!loading && calendars.length === 0"
      class="mt-3 rounded-lg border border-warning/25 bg-warning/10 px-3 py-2 text-sm text-warning"
    >
      {{ t('schedule.noCalendar') }}
      <span class="mt-1 block text-[12px] text-warning/80">{{ t('schedule.noCalendarHint') }}</span>
    </p>
    <!-- 跨学科选课提示：微型信息卡片 -->
    <div class="mt-3 flex items-start gap-2 rounded-lg bg-base-200/60 p-2.5 text-[12px] leading-relaxed text-base-content/70">
      <Info class="h-4 w-4 shrink-0 text-primary/70 mt-0.5" />
      <span class="min-w-0 flex-1">{{ t('schedule.majorHint') }}</span>
    </div>

    <!-- 底部操作与收起引导栏 -->
    <Transition name="config-ready" mode="out-in">
      <!-- 情况 A: collapsible 且已全部完成配置 (isConfigComplete) -> 醒目的主动收起提示卡片 -->
      <div
        v-if="collapsible && isConfigComplete"
        key="config-complete-card"
        class="mt-3 rounded-xl border border-primary/30 bg-gradient-to-b from-primary/[0.08] via-primary/[0.04] to-transparent p-3 shadow-xs"
        role="status"
        aria-live="polite"
        data-testid="schedule-config-ready-card"
      >
        <div class="flex items-start gap-2.5">
          <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary/15 text-primary shadow-xs ring-1 ring-primary/25">
            <Sparkles class="h-4 w-4" />
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-1.5 flex-wrap">
              <span class="text-xs font-semibold text-base-content">{{ t('schedule.configReadyTitle') }}</span>
              <span class="inline-flex items-center rounded-full bg-emerald-500/15 border border-emerald-500/25 px-1.5 py-0.2 text-[10px] font-medium text-emerald-700 dark:text-emerald-400">
                {{ t('schedule.readyToPick') }}
              </span>
            </div>
            <p class="mt-0.5 text-[11px] text-base-content/65 leading-relaxed">
              {{ t('schedule.configReadyDesc') }}
            </p>
          </div>
        </div>

        <div class="mt-2.5 flex items-center justify-between gap-2 pt-2 border-t border-primary/15">
          <button
            v-if="hasSelection"
            type="button"
            class="inline-flex items-center gap-1 text-[11px] font-medium text-base-content/50 hover:text-error transition-colors py-1 px-1.5 -ml-1 rounded-md hover:bg-error/10 active:scale-[0.97] cursor-pointer"
            :title="t('schedule.resetSelectionHint')"
            @click="clearAllSelection"
          >
            <RotateCcw class="h-3 w-3" />
            <span>{{ t('schedule.resetSelection') }}</span>
          </button>
          <div v-else />

          <button
            type="button"
            class="group gf-button gf-button-primary gf-button-sm font-semibold shadow-xs hover:shadow-md transition-all active:scale-[0.97] cursor-pointer inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg ring-1 ring-primary/30"
            @click="emit('toggle-collapse')"
          >
            <span>{{ t('schedule.collapseSettingsDone') }}</span>
            <ChevronUp class="h-3.5 w-3.5 transition-transform duration-200 group-hover:-translate-y-0.5" />
          </button>
        </div>
      </div>

      <!-- 情况 B: 常规底部操作栏 (未完成配置或非 collapsible 模式) -->
      <div
        v-else-if="collapsible || hasSelection"
        key="config-default-bar"
        class="mt-3 pt-2.5 border-t border-line/60 flex items-center justify-between gap-2"
      >
        <button
          v-if="hasSelection"
          type="button"
          class="inline-flex items-center gap-1.5 text-xs font-medium text-base-content/50 hover:text-error transition-colors py-1 px-2 -ml-1 rounded-lg hover:bg-error/10 active:scale-[0.97] cursor-pointer"
          :title="t('schedule.resetSelectionHint')"
          @click="clearAllSelection"
        >
          <RotateCcw class="h-3.5 w-3.5" />
          <span>{{ t('schedule.resetSelection') }}</span>
        </button>
        <div v-else />

        <!-- 快捷收起配置按钮（仅在 collapsible 模式下显示） -->
        <button
          v-if="collapsible"
          type="button"
          class="inline-flex items-center gap-1 text-xs font-medium text-base-content/60 hover:text-primary transition-colors py-1 px-2.5 rounded-lg hover:bg-base-200 active:scale-[0.97] cursor-pointer"
          @click="emit('toggle-collapse')"
        >
          <span>{{ t('schedule.collapseSettings') }}</span>
          <ChevronUp class="h-3.5 w-3.5" />
        </button>
      </div>
    </Transition>

    <p v-if="loading" class="mt-2 text-[12px] text-base-content/45">{{ t('schedule.loading') }}</p>
  </div>
</template>

<style scoped>
.config-ready-enter-active,
.config-ready-leave-active {
  transition: opacity 0.22s cubic-bezier(0.16, 1, 0.3, 1), transform 0.22s cubic-bezier(0.16, 1, 0.3, 1);
}

.config-ready-enter-from,
.config-ready-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

@media (prefers-reduced-motion: reduce) {
  .config-ready-enter-active,
  .config-ready-leave-active {
    transition: none !important;
  }
}
</style>
