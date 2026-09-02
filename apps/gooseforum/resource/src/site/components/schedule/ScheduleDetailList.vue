<script setup lang="ts">
// 课程班级列表：展示 clickedCourseInfo 对应课程的全部教学班，点击班级加入课表。
// 容忍式冲突：无论是否冲突都入表；有冲突时 emit('conflict') 仅用于父级 flash 提示，
// 不再弹「强制替换/放弃」窗（多方案/周次模型下不阻断）。
// 桌面端（lg+）双栏：左栏课程头部 + 教学班列表（行内课评链接聚焦该班评价）；
// 右栏浮动课评面板（复用课程详情页 RatingSummaryCard，与课评页 aside complementary
// 一致），含课程级评分仪表、教学班课评列表与「查看完整课评」入口。
// 移动端保持单栏，按 DOM 顺序堆叠（班级列表 → 课评面板）。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { BookOpen, ExternalLink, Star } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
import RatingSummaryCard from '@/site/components/RatingSummaryCard.vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import { getPkCourseReviewBrief } from '@/runtime/pk-api'
import type { PkConflictItem } from '@/site/utils/pkConflict'
import type { PkCourseDetail, PkCourseReviewBrief, PkReviewBriefClass } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

const emit = defineEmits<{
  conflict: [detail: PkCourseDetail, conflicts: PkConflictItem[]]
  staged: []
}>()

const currentCourse = computed(() =>
  store.state.commonLists.stagedCourses.find(
    (course) => course.courseCode === store.state.clickedCourseInfo.courseCode,
  ),
)

const brief = ref<PkCourseReviewBrief | null>(null)
const briefError = ref('')
const briefLoading = ref(false)
/** 请求序号：快速切换课程时丢弃过期响应，避免旧请求覆盖新课评摘要。 */
let briefRequestSeq = 0

/** 课评跳转：能匹配到课程目录主键时直达详情页，否则回退课程搜索页。
 * clickedCourseInfo.courseCode 已是基础课号（无班号），无需再剥班号。 */
const reviewHref = computed(() => {
  const base = store.state.clickedCourseInfo.courseCode
  if (brief.value?.courseId) {
    return `/courses/${brief.value.courseId}`
  }
  return `/courses?keyword=${encodeURIComponent(base)}`
})

/** 归一化班级课号：去掉点号（PK "122004.01" ↔ offering.class_code "12200401"）。 */
function normalizeClassCode(code: string): string {
  return String(code ?? '').replaceAll('.', '')
}

/** 教学班对应的 offering 级课评摘要（P13 classes 匹配；无记录返回 undefined）。 */
function classBrief(detailCode: string): PkReviewBriefClass | undefined {
  const target = normalizeClassCode(detailCode)
  return brief.value?.classes?.find((item) => normalizeClassCode(item.classCode) === target)
}

/** 教学班课评跳转（左栏行内链接）：有 offeringId 时聚焦该班评价，否则回退课程搜索页。 */
function classReviewHref(detailCode: string): string {
  const base = store.state.clickedCourseInfo.courseCode
  const item = classBrief(detailCode)
  if (brief.value?.courseId && item?.offeringId) {
    return `/courses/${brief.value.courseId}?offeringId=${item.offeringId}`
  }
  return `/courses?keyword=${encodeURIComponent(base)}`
}

/** 右栏教学班课评项跳转：有 offeringId 时聚焦该班评价，否则回退课程搜索页。 */
function classPanelHref(item: PkReviewBriefClass): string {
  const base = store.state.clickedCourseInfo.courseCode
  if (brief.value?.courseId && item.offeringId) {
    return `/courses/${brief.value.courseId}?offeringId=${item.offeringId}`
  }
  return `/courses?keyword=${encodeURIComponent(base)}`
}

async function loadBrief(courseCode: string) {
  const seq = ++briefRequestSeq
  brief.value = null
  briefError.value = ''
  briefLoading.value = true
  try {
    const result = await getPkCourseReviewBrief({
      courseCode,
      teacherName: '',
      calendarId: store.state.majorSelected.calendarId ?? 0,
    })
    if (seq !== briefRequestSeq) return
    brief.value = result
  } catch (err) {
    if (seq !== briefRequestSeq) return
    briefError.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  } finally {
    if (seq === briefRequestSeq) briefLoading.value = false
  }
}

watch(
  () => store.state.clickedCourseInfo.courseCode,
  (code) => {
    if (code) void loadBrief(code)
  },
  { immediate: true },
)

function statusLabel(status: number | undefined): string {
  if (status === 2) return t('schedule.statusSelected')
  if (status === 1) return t('schedule.statusStaged')
  return t('schedule.statusUnselected')
}

function statusClass(status: number | undefined): string {
  if (status === 2) return 'gf-badge gf-badge-success'
  if (status === 1) return 'gf-badge gf-badge-warning'
  return 'gf-badge gf-badge-muted'
}

function arrangementText(detail: PkCourseDetail): string {
  return detail.arrangementInfo.map((arr) => arr.arrangementText).join('；')
}

function teacherText(detail: PkCourseDetail): string {
  return detail.teachers.map((teacher) => teacher.teacherName).filter(Boolean).join('、')
}

function tryStage(detail: PkCourseDetail) {
  // 容忍式：总是入表；冲突仅作 flash 提示（deriveConflicts 负责课表/列表/统计标注）。
  const result = store.stageCourse(detail)
  store.solidify()
  if (result.conflicts && result.conflicts.length > 0) {
    emit('conflict', detail, result.conflicts)
    return
  }
  emit('staged')
}
</script>

<template>
  <div class="space-y-3 lg:grid lg:grid-cols-[minmax(0,1fr)_minmax(0,320px)] lg:items-start lg:gap-3 lg:space-y-0">
    <!-- 左栏：课程信息 + 教学班列表（点击班级加入课表） -->
    <div v-if="!currentCourse" class="gf-panel">
      <EmptyState
        :icon="BookOpen"
        :title="t('schedule.emptyDetailGuide')"
        :description="t('schedule.majorHint')"
      />
    </div>

    <div v-else class="gf-panel">
      <div class="border-b border-line/60 px-3 py-2">
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <h3 class="truncate text-[13px] font-bold text-base-content">
              {{ currentCourse.courseNameReserved }}
            </h3>
            <p class="text-[11px] text-base-content/50">{{ currentCourse.courseCode }} · {{ t('schedule.credit', { credit: currentCourse.credit }) }}</p>
          </div>
          <a
            :href="reviewHref"
            class="shrink-0 rounded-lg border border-line/60 px-2 py-1 text-[11px] font-medium text-primary hover:bg-base-200/60"
          >
            {{ t('schedule.reviews') }}
          </a>
        </div>
      </div>
      <ul v-if="currentCourse.courseDetail.length" class="gf-scrollbar-thin divide-y divide-line/60">
        <li
          v-for="detail in currentCourse.courseDetail"
          :key="detail.code"
          class="px-3 py-2"
        >
          <div class="flex items-center justify-between gap-2">
            <button
              type="button"
              class="min-w-0 flex-1 text-left"
              @click="tryStage(detail)"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="min-w-0 truncate text-[13px] font-medium text-base-content">{{ detail.code }}</span>
                <span :class="statusClass(detail.status)">{{ statusLabel(detail.status) }}</span>
              </div>
              <p v-if="teacherText(detail)" class="mt-0.5 text-[12px] text-base-content/60">
                {{ t('schedule.teacherWith', { value: teacherText(detail) }) }}
              </p>
              <p class="mt-0.5 line-clamp-2 text-[12px] text-base-content/60">
                {{ arrangementText(detail) }}
              </p>
              <p class="mt-0.5 text-[11px] text-base-content/45">
                {{ detail.campus }} · {{ detail.teachingLanguage }}
                <template v-if="detail.isExclusive"> · {{ t('schedule.tabRequired') }}</template>
              </p>
            </button>
          </div>
          <a
            v-if="classBrief(detail.code)"
            :href="classReviewHref(detail.code)"
            class="mt-1.5 inline-flex items-center gap-1 rounded-md border border-line/50 px-1.5 py-0.5 text-[11px] text-primary hover:bg-base-200/60"
          >
            <Star v-if="classBrief(detail.code)?.ratingAvg != null" class="h-3 w-3 fill-amber-400 text-amber-400" />
            <template v-if="classBrief(detail.code)?.ratingAvg != null">
              {{ classBrief(detail.code)?.ratingAvg?.toFixed(1) }}
            </template>
            {{ t('schedule.reviewCount', { count: classBrief(detail.code)?.reviewCount ?? 0 }) }}
            →
          </a>
        </li>
      </ul>
      <p v-else class="px-3 py-3 text-[12px] text-base-content/50">{{ t('schedule.emptyDetailNoClass') }}</p>
    </div>

    <!-- 右栏：课评面板（lg+ 浮动右侧，与课评详情页 aside complementary 同款评分仪表卡；
         移动端按 DOM 顺序堆叠在班级列表下方） -->
    <aside
      v-if="currentCourse"
      class="min-w-0 space-y-3 lg:sticky lg:top-0"
      role="complementary"
      :aria-label="t('schedule.reviews')"
    >
      <template v-if="briefLoading">
        <div class="gf-panel p-4">
          <p class="text-[12px] text-base-content/55">{{ t('schedule.loading') }}</p>
        </div>
      </template>
      <template v-else-if="briefError">
        <div class="gf-panel p-4">
          <p class="text-[12px] text-base-content/45">{{ t('schedule.loadFailed') }}</p>
        </div>
      </template>
      <template v-else-if="brief">
        <RatingSummaryCard
          :rating-avg="brief.ratingAvg ?? null"
          :review-count="brief.reviewCount"
          :distribution="brief.ratingDistribution ?? [0, 0, 0, 0, 0]"
        />

        <section v-if="brief.classes?.length" class="gf-panel p-4" :aria-label="t('schedule.classReviewsTitle')">
          <h4 class="mb-2 inline-flex items-center gap-1.5 text-[13px] font-semibold text-base-content">
            <Star class="h-3.5 w-3.5 text-base-content/45" />
            {{ t('schedule.classReviewsTitle') }}
          </h4>
          <ul class="divide-y divide-line/60">
            <li v-for="item in brief.classes" :key="item.offeringId" class="py-1.5 first:pt-0 last:pb-0">
              <a
                :href="classPanelHref(item)"
                class="flex items-center justify-between gap-2 rounded-md px-1 py-1 transition hover:bg-base-200/60"
              >
                <span class="min-w-0">
                  <span class="block truncate text-[12px] font-medium text-base-content">{{ item.classCode }}</span>
                  <span class="block truncate text-[11px] text-base-content/50">
                    {{ item.teachers.join('、') || '—' }}
                  </span>
                </span>
                <span class="shrink-0 text-right">
                  <span class="block text-[12px] font-semibold tabular-nums text-warning">
                    {{ item.ratingAvg != null ? item.ratingAvg.toFixed(1) : '—' }}
                  </span>
                  <span class="block text-[10px] tabular-nums text-base-content/45">
                    {{ t('schedule.reviewCount', { count: item.reviewCount }) }}
                  </span>
                </span>
              </a>
            </li>
          </ul>
        </section>

        <a :href="reviewHref" class="gf-button gf-button-md gf-button-primary w-full">
          {{ t('schedule.reviews') }}
          <ExternalLink class="h-3.5 w-3.5" />
        </a>
      </template>
    </aside>
  </div>
</template>
