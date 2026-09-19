<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import type { CampusCalendarRules, CampusCalendarSettings } from '@gooseforum/client'
import { CalendarDays, Sparkles, Plus, Trash2, ArrowRight, Save } from '@lucide/vue'
import { BasicPage } from '@/admin/components/global-layout'
import AdminSection from '@/admin/components/AdminSection.vue'
import { Button } from '@/admin/components/ui/button'
import { Input } from '@/admin/components/ui/input'
import { Textarea } from '@/admin/components/ui/textarea'
import { calendarRulesAPI, teachingDateLabel } from '@/runtime/calendar-rules-api'

const rules = ref<CampusCalendarRules>({ holidays: [], moves: [] })
const revision = ref(''), year = ref(new Date().getFullYear()), notice = ref('')
const loading = ref(true), busy = ref(false), error = ref(''), success = ref(''), warnings = ref<string[]>([])
const controller = new AbortController()
const dirty = ref(false)
const count = computed(() => rules.value.holidays.length + rules.value.moves.length)
function changed() { dirty.value = true; success.value = '' }
function accept(settings: CampusCalendarSettings) { rules.value = settings.rules; revision.value = settings.revision; dirty.value = false }
async function load() {
  loading.value = true; error.value = ''
  try { accept(await calendarRulesAPI.read(true, controller.signal)); warnings.value = [] }
  catch (e) { if (!controller.signal.aborted) error.value = String(e instanceof Error ? e.message : e) }
  finally { loading.value = false }
}
async function parse() {
  busy.value = true; error.value = ''; success.value = ''
  try {
    const draft = await calendarRulesAPI.parse({ year: Number(year.value), text: notice.value }, controller.signal)
    // Add to the draft without deleting previously published dates. Exact duplicates
    // are ignored; other conflicts remain visible for editing and server validation.
    const merge = <T,>(existing: T[], incoming: T[]) => [...existing, ...incoming.filter(v => !existing.some(e => JSON.stringify(e) === JSON.stringify(v)))]
    rules.value = { holidays: merge(rules.value.holidays, draft.rules.holidays), moves: merge(rules.value.moves, draft.rules.moves) }
    warnings.value = draft.warnings; changed()
    success.value = '已加入草稿，请核对下方日期和教学对应关系，再点击“应用规则”。'
  } catch (e) { if (!controller.signal.aborted) error.value = e instanceof Error ? e.message : '解析失败' }
  finally { busy.value = false }
}
async function save() {
  busy.value = true; error.value = ''; success.value = ''
  try { accept(await calendarRulesAPI.save({ revision: revision.value, rules: rules.value }, controller.signal)); success.value = '规则已应用。之后开启调休的课程日历导出将使用这些规则。' }
  catch (e) { if (!controller.signal.aborted) error.value = e instanceof Error ? e.message : '保存失败' }
  finally { busy.value = false }
}
onMounted(load)
onBeforeUnmount(() => controller.abort())
</script>

<template>
  <BasicPage title="校园调休规则" description="将学校通知整理成准确的放假与补课日期，供 Web 和 App 导出课程日历时使用。">
    <div class="space-y-5">
      <div v-if="error" role="alert" class="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">{{ error }}</div>
      <div v-if="success" role="status" class="rounded-lg border bg-muted/30 p-4 text-sm">{{ success }}</div>
      <div class="flex flex-wrap items-center justify-between gap-3 text-sm">
        <span class="text-muted-foreground">{{ loading ? '正在加载规则…' : `${count} 条规则 · ${dirty ? '有未应用的修改' : '已发布'}` }}</span>
        <Button variant="outline" :disabled="busy || loading" @click="load">重新加载已发布规则</Button>
      </div>
      <fieldset :disabled="busy || loading || !revision" class="min-w-0 space-y-5 disabled:opacity-70">
        <AdminSection body-class="space-y-4 p-5">
          <h2 class="flex items-center gap-2 font-semibold"><Sparkles class="size-4" /> 从学校通知生成草稿</h2>
          <p class="text-sm leading-6 text-muted-foreground">使用“<a href="/admin/settings/ai-summary" class="underline underline-offset-4">AI 课程总结</a>”中配置的模型。只提交下方通知和年份；不提交个人课表。也可跳过 AI，直接编辑规则。</p>
          <label class="block max-w-40 space-y-2 text-sm">通知年份<Input v-model.number="year" aria-label="通知年份" type="number" min="2000" max="2100" /></label>
          <label class="block space-y-2 text-sm">学校放假与教学安排通知<Textarea v-model="notice" aria-label="学校通知" :maxlength="12000" rows="5" placeholder="例如：国庆节10月1日至7日放假。9月20日安排10月6日的教学工作。" /></label>
          <Button :disabled="!notice.trim() || busy || loading" @click="parse"><Sparkles class="size-4" />{{ busy ? '处理中…' : 'AI 生成草稿' }}</Button>
          <p class="text-xs text-muted-foreground">生成结果会追加到当前草稿。AI 不明确的安排会提示核实，应用前请对照学校通知检查。</p>
        </AdminSection>
        <div v-if="warnings.length" class="rounded-lg border bg-muted/30 p-4 text-sm" role="alert">
          <p class="font-medium">需要核实</p><ul class="mt-2 list-inside list-disc space-y-1"><li v-for="(warning, i) in warnings" :key="i">{{ warning }}</li></ul>
        </div>
        <AdminSection body-class="space-y-4 p-5">
          <div class="flex flex-wrap items-center justify-between gap-2"><h2 class="flex items-center gap-2 font-semibold"><CalendarDays class="size-4" /> 放假停课 <span class="text-muted-foreground">{{ rules.holidays.length }}</span></h2><Button variant="outline" @click="rules.holidays.push({ name: '', startDate: '', endDate: '' }); changed()"><Plus class="size-4" />添加假期</Button></div>
          <p class="text-sm text-muted-foreground">首尾日期均停课；已指定补课去向的课程会移到实际补课日。</p>
          <p v-if="!rules.holidays.length" class="py-4 text-center text-sm text-muted-foreground">暂无放假规则</p>
          <div v-for="(holiday, i) in rules.holidays" :key="i" class="grid gap-3 rounded-lg border bg-muted/10 p-3 md:grid-cols-[1fr_1fr_1fr_auto]">
            <label class="space-y-1 text-xs">假期名称<Input v-model="holiday.name" :aria-label="`假期 ${i + 1} 名称`" maxlength="80" @update:model-value="changed" /></label>
            <label class="space-y-1 text-xs">开始日期<Input v-model="holiday.startDate" :aria-label="`假期 ${i + 1} 开始日期`" type="date" @update:model-value="changed" /><span class="block text-muted-foreground">{{ teachingDateLabel(holiday.startDate) }}</span></label>
            <label class="space-y-1 text-xs">结束日期<Input v-model="holiday.endDate" :aria-label="`假期 ${i + 1} 结束日期`" type="date" @update:model-value="changed" /><span class="block text-muted-foreground">{{ teachingDateLabel(holiday.endDate) }}</span></label>
            <Button variant="ghost" :aria-label="`删除假期 ${i + 1}`" class="self-center" @click="rules.holidays.splice(i, 1); changed()"><Trash2 class="size-4" /></Button>
          </div>
        </AdminSection>
        <AdminSection body-class="space-y-4 p-5">
          <div class="flex flex-wrap items-center justify-between gap-2"><h2 class="font-semibold">教学日期调整 <span class="text-muted-foreground">{{ rules.moves.length }}</span></h2><Button variant="outline" @click="rules.moves.push({ name: '', fromDate: '', toDate: '' }); changed()"><Plus class="size-4" />添加补课</Button></div>
          <p class="text-sm leading-6 text-muted-foreground">原教学日 → 实际上课日。使用原教学日所在周的课程和单双周安排，替换实际补课日原本的课表。</p>
          <p v-if="!rules.moves.length" class="py-4 text-center text-sm text-muted-foreground">暂无补课规则</p>
          <div v-for="(move, i) in rules.moves" :key="i" class="grid items-start gap-3 rounded-lg border bg-muted/10 p-3 md:grid-cols-[1fr_1fr_auto_1fr_auto]">
            <label class="space-y-1 text-xs">调整名称<Input v-model="move.name" :aria-label="`补课 ${i + 1} 名称`" maxlength="80" @update:model-value="changed" /></label>
            <label class="space-y-1 text-xs">原教学日（补哪天的课）<Input v-model="move.fromDate" :aria-label="`补课 ${i + 1} 原教学日`" type="date" @update:model-value="changed" /><span class="block text-muted-foreground">{{ teachingDateLabel(move.fromDate) }}</span></label>
            <ArrowRight class="hidden size-4 self-center text-muted-foreground md:block" />
            <label class="space-y-1 text-xs">实际上课日<Input v-model="move.toDate" :aria-label="`补课 ${i + 1} 实际上课日`" type="date" @update:model-value="changed" /><span class="block text-muted-foreground">{{ teachingDateLabel(move.toDate) }}</span></label>
            <Button variant="ghost" :aria-label="`删除补课 ${i + 1}`" class="self-center" @click="rules.moves.splice(i, 1); changed()"><Trash2 class="size-4" /></Button>
          </div>
        </AdminSection>
        <div class="flex flex-wrap items-center gap-4"><Button :disabled="!dirty || busy || loading" @click="save"><Save class="size-4" />应用规则</Button><span class="text-xs text-muted-foreground">应用后影响后续导出；已导入日历 App 的文件不会自动更新。清空列表后应用可撤销全部规则。</span></div>
      </fieldset>
    </div>
  </BasicPage>
</template>
