<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { campusMapDemoEvents } from '@/site/campus-map/campus-map-demo'
import { officialLocationTarget, parseOfficialLocation } from '@/site/campus-map/official-location'

const emit = defineEmits<{ select: [target: ReturnType<typeof officialLocationTarget> | null] }>()
const { t } = useI18n()
const selected = ref<number | null>(null)

function choose(index: number) {
  selected.value = selected.value === index ? null : index
  const course = campusMapDemoEvents[index]!
  emit('select', selected.value === null ? null : officialLocationTarget(course.campus, course.room) ?? null)
}

function location(campus: string, room: string) {
  const parsed = parseOfficialLocation(room)
  return [campus, parsed?.building, parsed?.room || (!parsed ? room : '')].filter(Boolean).join(' · ')
}
</script>

<template>
  <section class="atlas-mine" aria-labelledby="atlas-mine-title">
    <p class="atlas-mine__eyebrow">{{ t('campusMap.mine.source') }}</p>
    <h2 id="atlas-mine-title">{{ t('campusMap.mine.title') }}</h2>
    <p class="atlas-mine__note" role="note">示例数据：仅用于本地界面预览，不是本人课表。</p>
    <div class="atlas-mine__list" aria-live="polite">
      <button
        v-for="(course, index) in campusMapDemoEvents"
        :key="course.name"
        type="button"
        class="atlas-mine__course"
        :aria-pressed="selected === index"
        @click="choose(index)"
      >
        <span class="atlas-mine__period">第 {{ course.start }}–{{ course.end }} 节 · 第 6 周</span>
        <strong>{{ course.name }}</strong>
        <small>{{ location(course.campus, course.room) }}</small>
      </button>
    </div>
    <div v-if="selected !== null" class="atlas-mine__selected" role="status">
      <strong>{{ campusMapDemoEvents[selected]!.name }}</strong>
      <p>{{ location(campusMapDemoEvents[selected]!.campus, campusMapDemoEvents[selected]!.room) }}</p>
    </div>
  </section>
</template>

<style scoped>
.atlas-mine { display: flex; min-height: 100%; flex-direction: column; gap: 12px; padding: 20px; color: #24342f; }
.atlas-mine__eyebrow { margin: 0; color: #5d7469; font-size: 11px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; }
.atlas-mine h2 { margin: 0; font-size: 21px; font-weight: 650; }
.atlas-mine__note { margin: 0; color: #62746d; font-size: 13px; line-height: 1.55; }
.atlas-mine__list { display: grid; gap: 8px; overflow: auto; }
.atlas-mine__course { display: grid; gap: 4px; border: 1px solid #e1e8e3; border-radius: 12px; background: white; padding: 12px; color: inherit; text-align: left; cursor: pointer; }
.atlas-mine__course[aria-pressed="true"] { border-color: #58846b; box-shadow: 0 0 0 2px #58846b22; }
.atlas-mine__period, .atlas-mine__course small { color: #6c7e74; font-size: 11px; }
.atlas-mine__course strong, .atlas-mine__selected strong { font-size: 14px; }
.atlas-mine__selected { display: grid; gap: 6px; border-top: 1px solid #e1e8e3; padding-top: 12px; font-size: 13px; }
.atlas-mine__selected p { margin: 0; color: #62746d; line-height: 1.5; }
</style>
