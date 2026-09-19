<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import type { CampusCalendarRules } from '@gooseforum/client'
import { calendarRulesAPI, teachingDateLabel } from '@/runtime/calendar-rules-api'
const rules = ref<CampusCalendarRules | null>(null), error = ref(''), loading = ref(false)
const controller = new AbortController()
async function load(event: Event) {
  if (!(event.target as HTMLDetailsElement).open || loading.value) return
  loading.value = true; error.value = ''; rules.value = null
  try { rules.value = (await calendarRulesAPI.read(false, controller.signal)).rules }
  catch (e) { if (!controller.signal.aborted) error.value = e instanceof Error ? e.message : '规则暂不可用' }
  finally { loading.value = false }
}
onBeforeUnmount(() => controller.abort())
</script>
<template>
  <details class="border-b border-line px-4 py-3 text-xs leading-6" @toggle="load">
    <summary class="cursor-pointer text-base-content/60">查看已发布的调休规则</summary>
    <p v-if="loading" class="mt-2 text-base-content/55">正在读取…</p>
    <p v-if="error" role="alert" class="mt-2 text-error">{{ error }}</p>
    <div v-if="rules" class="mt-2 space-y-2">
      <p class="text-base-content/55">仅影响开启调休后的日历导出；页面课表仍显示学校原始安排。导出时使用最新已发布规则。</p>
      <p v-if="!rules.holidays.length && !rules.moves.length" class="text-base-content/55">管理员尚未设置调休，按学校原课表导出。</p>
      <p v-for="h in rules.holidays" :key="h.startDate"><span class="gf-badge mr-2">停课</span>{{ h.name }} · {{ teachingDateLabel(h.startDate) }} 至 {{ teachingDateLabel(h.endDate) }}</p>
      <p v-for="m in rules.moves" :key="m.fromDate"><span class="gf-badge mr-2">补课</span>{{ m.name }} · {{ teachingDateLabel(m.toDate) }} 上 {{ teachingDateLabel(m.fromDate) }} 的课</p>
    </div>
  </details>
</template>
