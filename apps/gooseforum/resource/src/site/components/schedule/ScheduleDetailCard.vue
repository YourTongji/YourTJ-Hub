<script setup lang="ts">
// 课表课程详情卡（桌面点击 / 移动端长按课程块打开）。
// 展示教师与上课安排，并异步加载课评摘要（P13 course-review-brief，验收标准 4 课评入口）。
// P13 端点属于 #187，未实现时友好降级显示「课评暂不可用」。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDialogAccessibility } from '@/site/composables/useDialogAccessibility'
import { X } from '@lucide/vue'
import { getPkCourseReviewBrief } from '@/runtime/pk-api'
import { getCourseBaseCode } from '@/site/utils/pkConflict'
import type { PkCourseOnTable, PkCourseReviewBrief } from '@/site/types/pk'

const { t } = useI18n()

const props = defineProps<{
  course: PkCourseOnTable | null
}>()

const emit = defineEmits<{
  close: []
}>()

const { panelRef } = useDialogAccessibility(computed(() => props.course !== null), {
  onClose: () => emit('close'),
})

const brief = ref<PkCourseReviewBrief | null>(null)
const briefError = ref('')
const briefLoading = ref(false)
/** 请求序号：快速切换课程时丢弃过期响应，避免旧请求覆盖新课评摘要。 */
let briefRequestSeq = 0

const parsed = computed(() => {
  const raw = String(props.course?.showText || '').trim()
  const match = /^(\S+)\s+(.+?)\(([^)]+)\)\s+(.+)$/.exec(raw)
  if (match) {
    return {
      teacherAndCode: match[1],
      name: match[2].trim(),
      code: match[3].trim(),
      arrangement: match[4].trim(),
    }
  }
  return {
    teacherAndCode: '',
    name: props.course?.courseName || t('schedule.courseFallback'),
    code: props.course?.code || '',
    arrangement: raw,
  }
})

/** 课评入口：能匹配到课程目录主键时直达详情页，否则回退课程搜索页。 */
const reviewHref = computed(() => {
  const base = getCourseBaseCode(props.course?.code || '')
  if (brief.value?.courseId) {
    return `/courses/${brief.value.courseId}`
  }
  return `/courses?keyword=${encodeURIComponent(base)}`
})

async function loadBrief() {
  const course = props.course
  if (!course) return
  const seq = ++briefRequestSeq
  brief.value = null
  briefError.value = ''
  briefLoading.value = true
  try {
    const result = await getPkCourseReviewBrief({
      courseCode: getCourseBaseCode(course.code),
      teacherName: '',
    })
    if (seq !== briefRequestSeq) return // 过期响应丢弃
    brief.value = result
  } catch (err) {
    if (seq !== briefRequestSeq) return
    briefError.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  } finally {
    if (seq === briefRequestSeq) briefLoading.value = false
  }
}

watch(
  () => props.course,
  (course) => {
    if (course) void loadBrief()
  },
  { immediate: true },
)
</script>

<template>
  <Teleport to="body">
    <Transition name="gf-fade">
      <div
        v-if="course"
        ref="panelRef"
        role="dialog"
        aria-modal="true"
        aria-labelledby="schedule-detail-title"
        class="fixed inset-0 z-[2100]"
      >
        <div class="absolute inset-0 bg-black/40" @click="emit('close')"></div>
        <div class="absolute left-1/2 top-1/2 max-h-[88vh] w-[88vw] max-w-[420px] -translate-x-1/2 -translate-y-1/2 overflow-y-auto">
          <div class="overflow-hidden rounded-2xl border border-line/70 bg-base-100 shadow-2xl" @click.stop>
            <div class="flex items-start justify-between gap-2 px-4 py-3">
              <div class="min-w-0">
                <div id="schedule-detail-title" class="text-sm font-bold text-base-content">{{ parsed.name }}</div>
                <div class="text-[11px] text-base-content/55">{{ parsed.code }}</div>
              </div>
              <button
                type="button"
                class="gf-icon-button"
                :aria-label="t('common.close')"
                @click="emit('close')"
              >
                <X class="h-4 w-4" />
              </button>
            </div>
            <div class="space-y-2 p-4 pt-0">
              <p v-if="parsed.teacherAndCode" class="text-[12px] text-base-content/70">
                {{ parsed.teacherAndCode }}
              </p>
              <p class="whitespace-pre-wrap break-words text-[13px] leading-snug text-base-content">
                {{ parsed.arrangement }}
              </p>

              <a
                :href="reviewHref"
                class="gf-button gf-button-md gf-button-primary mt-2 w-full"
              >
                {{ t('schedule.reviews') }}
              </a>

              <div class="rounded-lg border border-line/60 bg-base-200/40 p-3">
                <template v-if="briefLoading">
                  <p class="text-[12px] text-base-content/55">{{ t('schedule.loading') }}</p>
                </template>
                <template v-else-if="briefError">
                  <p class="text-[12px] text-base-content/45">{{ t('schedule.loadFailed') }}</p>
                </template>
                <template v-else-if="brief">
                  <p class="text-[12px] text-base-content/70">
                    <template v-if="brief.ratingAvg != null">
                      {{ t('schedule.reviewAvg', { value: brief.ratingAvg.toFixed(1) }) }}
                    </template>
                    {{ t('schedule.reviewCount', { count: brief.reviewCount }) }}
                  </p>
                </template>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
