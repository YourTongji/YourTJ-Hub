<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { OfficialMapLocation } from '@/site/campus-map/official-location'

defineProps<{ locations: OfficialMapLocation[]; selectedIndex: number | null }>()
const emit = defineEmits<{ select: [location: OfficialMapLocation, index: number] }>()
const { t } = useI18n()

function label(location: OfficialMapLocation) {
  return [location.campusText, location.condition, location.building, location.room, location.address].filter(Boolean).join(' ')
}
</script>

<template>
  <div class="atlas-location-choices">
    <p v-if="locations.some(location => location.reviewPending)" class="atlas-location-choices__hint">{{ t('campusMap.locations.reviewPending') }}</p>
    <p v-if="locations.length > 1" class="atlas-location-choices__hint">{{ t('campusMap.locations.choose') }}</p>
    <div class="atlas-location-choices__list" :aria-label="t('campusMap.locations.label')">
      <button
        v-for="(location, index) in locations"
        :key="`${index}-${location.raw}`"
        type="button"
        :disabled="!location.target"
        :aria-pressed="selectedIndex === index"
        @click="emit('select', location, index)"
      >
        <span>{{ location.raw }}</span>
        <small v-if="label(location) && label(location) !== location.raw">{{ label(location) }}</small>
        <small v-if="location.unassignedConditions?.length">{{ t('campusMap.locations.unassigned', { conditions: location.unassignedConditions.join('；') }) }}</small>
        <small v-if="!location.target" class="atlas-location-choices__unmapped">{{ t(`campusMap.locations.hints.${location.hint ?? 'unmapped'}`) }}</small>
      </button>
    </div>
  </div>
</template>

<style scoped>
.atlas-location-choices { display: grid; gap: 7px; }
.atlas-location-choices__hint { margin: 0; color: #62746d; font-size: 12px; line-height: 1.5; }
.atlas-location-choices__list { display: grid; gap: 6px; }
.atlas-location-choices button { display: grid; gap: 3px; min-height: 44px; border: 1px solid #dce4de; border-radius: 9px; background: white; padding: 9px 10px; color: #24342f; font: inherit; font-size: 12px; text-align: left; overflow-wrap: anywhere; cursor: pointer; }
.atlas-location-choices button[aria-pressed='true'] { border-color: #58846b; box-shadow: 0 0 0 2px #58846b22; }
.atlas-location-choices button:disabled { background: #f6f6f0; cursor: default; }
.atlas-location-choices small { color: #62746d; font-size: 11px; line-height: 1.4; }
.atlas-location-choices .atlas-location-choices__unmapped { color: #8a5522; }
</style>
