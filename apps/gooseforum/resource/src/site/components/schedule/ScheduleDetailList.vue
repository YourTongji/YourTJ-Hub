<script setup lang="ts">
// 课程班级列表：展示 clickedCourseInfo 对应课程的全部教学班，点击班级尝试加入课表。
// 无冲突直接加入（status → 备选）；有冲突 emit('conflict') 由父级弹窗决定「强制替换/放弃」。
// 课程头部异步加载课评摘要（P13 course-review-brief），并给出跳转课评详情/搜索的入口。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { BookOpen, Star } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
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

/** 教学班课评跳转：有 offeringId 时聚焦该班评价，否则回退课程搜索页。 */
function classReviewHref(detailCode: string): string {
  const base = store.state.clickedCourseInfo.courseCode
  const item = classBrief(detailCode)
  if (brief.value?.courseId && item?.offeringId) {
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
  const result = store.stageCourse(detail)
  if (result.added) {
    store.solidify()
    emit('staged')
    return
  }
  emit('conflict', detail, result.conflicts ?? [])
}
</script>

<template>
  <div class="space-y-3">
    <EmptyState
      v-if="!currentCourse"
      class="gf-panel"
      :icon="BookOpen"
      :title="t('schedule.emptyDetailGuide')"
      :description="t('schedule.majorHint')"
    />

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

        <div class="mt-1.5 flex min-h-[18px] items-center gap-1 text-[11px] text-base-content/55">
          <template v-if="briefLoading">
            <span>{{ t('schedule.loading') }}</span>
          </template>
          <template v-else-if="briefError">
            <span class="text-base-content/40">{{ t('schedule.loadFailed') }}</span>
          </template>
          <template v-else-if="brief">
            <Star v-if="brief.ratingAvg != null" class="h-3 w-3 fill-amber-400 text-amber-400" />
            <span v-if="brief.ratingAvg != null" class="font-medium text-base-content/75">
              {{ brief.ratingAvg.toFixed(1) }}
            </span>
            <span>{{ t('schedule.reviewCount', { count: brief.reviewCount }) }}</span>
          </template>
        </div>
      </div>
      <ul v-if="currentCourse.courseDetail.length" class="gf-scrollbar-thin max-h-96 divide-y divide-line/60 overflow-y-auto overscroll-contain">
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
  </div>
</template>
