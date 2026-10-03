<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, CheckCircle2, ChevronDown, ChevronLeft, ChevronRight, Eye, Gauge, KeyRound, Loader2, PlugZap, RefreshCw, Save, ScanEye, ShieldAlert, Trash2, Undo2, X, XCircle } from '@lucide/vue'
import { BasicPage } from '@/admin/components/global-layout'
import AdminSection from '@/admin/components/AdminSection.vue'
import { Badge } from '@/admin/components/ui/badge'
import { Button } from '@/admin/components/ui/button'
import { Input } from '@/admin/components/ui/input'
import { Switch } from '@/admin/components/ui/switch'
import { Textarea } from '@/admin/components/ui/textarea'
import {
  getAiModerationSettings,
  labelAiModerationDecision,
  listAiModerationDecisions,
  replayAiModeration,
  saveAiModerationSettings,
  testAiModerationConnection,
} from '@/admin/runtime/api'
import { adminToast } from '@/admin/runtime/toast'
import type { AiModerationConnectionCheck, AiModerationDecision, AiModerationOptions, AiModerationPolicyRule, AiModerationReplayReport, AiModerationSettingsInput } from '@/admin/types'
import { policyName, reasonText } from '@/admin/utils/aiModerationReasons'

// AI 图文审查（issue #975）：配置、决策记录标注与离线阈值回放。
const { t } = useI18n()
const text = (key: string, values: Record<string, unknown> = {}) => t(`aiModerationAdmin.${key}`, values)
const cap = (value: string) => value.charAt(0).toUpperCase() + value.slice(1)

const JEV_PRESETS = [
  { label: 'OpenRouter', endpoint: 'https://openrouter.ai/api/alpha/decisions', model: 'typesafe/jev-1.13' },
  { label: 'TypeSafe', endpoint: 'https://api.typesafe.ai/v1/systemone', model: 'jev-latest' },
]

const loading = ref(false)
const saving = ref(false)
const form = reactive<AiModerationSettingsInput & { jevApiKeyConfigured: boolean, visionApiKeyConfigured: boolean }>({
  enabled: false,
  mode: 'shadow',
  textModeration: false,
  jevEndpoint: '',
  jevModel: '',
  jevTimeoutMs: 10000,
  jevRetries: 0,
  visionBaseUrl: '',
  visionModel: '',
  visionTimeoutMs: 20000,
  policyRevision: '',
  policies: [],
  defaultReviewThreshold: 0.5,
  defaultBlockThreshold: 0.9,
  reviewNeededThreshold: 0.7,
  severityBlockThreshold: 2.5,
  externalImageAction: 'review',
  maxImagesPerDecision: 6,
  globalRequestsPerMinute: 30,
  perUserRequestsPerMinute: 5,
  jevApiKey: '',
  visionApiKey: '',
  clearJevApiKey: false,
  clearVisionApiKey: false,
  jevApiKeyConfigured: false,
  visionApiKeyConfigured: false,
})

// optionalNumber 输入框清空时 v-model.number 产生 ''：一律视为“未覆盖”。
function optionalNumber(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : undefined
}

function options(): AiModerationOptions {
  return {
    enabled: form.enabled,
    mode: form.mode,
    textModeration: form.textModeration,
    jevEndpoint: form.jevEndpoint.trim(),
    jevModel: form.jevModel.trim(),
    jevTimeoutMs: Number(form.jevTimeoutMs) || 0,
    jevRetries: Number(form.jevRetries) || 0,
    visionBaseUrl: form.visionBaseUrl.trim(),
    visionModel: form.visionModel.trim(),
    visionTimeoutMs: Number(form.visionTimeoutMs) || 0,
    policyRevision: form.policyRevision.trim(),
    policies: form.policies.map(rule => ({
      ...rule,
      reviewThreshold: optionalNumber(rule.reviewThreshold),
      blockThreshold: optionalNumber(rule.blockThreshold),
    })),
    defaultReviewThreshold: Number(form.defaultReviewThreshold) || 0,
    defaultBlockThreshold: Number(form.defaultBlockThreshold) || 0,
    reviewNeededThreshold: Number(form.reviewNeededThreshold) || 0,
    severityBlockThreshold: Number(form.severityBlockThreshold) || 0,
    externalImageAction: form.externalImageAction,
    maxImagesPerDecision: Number(form.maxImagesPerDecision) || 0,
    globalRequestsPerMinute: Number(form.globalRequestsPerMinute) || 0,
    perUserRequestsPerMinute: Number(form.perUserRequestsPerMinute) || 0,
  }
}

async function load() {
  loading.value = true
  try {
    const view = await getAiModerationSettings()
    Object.assign(form, view, { jevApiKey: '', visionApiKey: '', clearJevApiKey: false, clearVisionApiKey: false })
  } catch (err) {
    adminToast.error(err, text('loadFailed'))
  } finally {
    loading.value = false
  }
}

// ---- 测试连接（使用当前表单值，无需保存）----
type ProviderTarget = 'jev' | 'vision'
const testing = reactive<Record<ProviderTarget, boolean>>({ jev: false, vision: false })
const testResults = reactive<Record<ProviderTarget, AiModerationConnectionCheck | null>>({ jev: null, vision: null })

function formInput(): AiModerationSettingsInput {
  return {
    ...options(),
    jevApiKey: form.jevApiKey?.trim() || undefined,
    visionApiKey: form.visionApiKey?.trim() || undefined,
    clearJevApiKey: form.clearJevApiKey || undefined,
    clearVisionApiKey: form.clearVisionApiKey || undefined,
  }
}

async function testConnection(target: ProviderTarget) {
  if (testing[target]) return
  testing[target] = true
  testResults[target] = null
  try {
    testResults[target] = await testAiModerationConnection(target, formInput())
  } catch (err) {
    adminToast.error(err, text('testRequestFailed'))
  } finally {
    testing[target] = false
  }
}

// testResultText 把服务端分类转为可操作的说明（如文本模型不支持图片输入）。
function testResultText(target: ProviderTarget, check: AiModerationConnectionCheck) {
  const status = check.httpStatus ?? 0
  if (check.kind === 'ok') return text('testOk', { model: check.model || '-', ms: check.latencyMs })
  if (check.kind === 'uncertain') return text('testOkUncertain', { model: check.model || '-', ms: check.latencyMs })
  if (check.kind === 'not_configured') return text('kindNotConfigured')
  if (target === 'vision' && check.kind === 'http_404') return text('kindVisionNoImage')
  if (status === 401 || status === 403 || check.kind === 'auth') return text('kindAuth', { status })
  if (check.kind === 'refused') return text('kindRefused')
  if (check.kind === 'invalid' || check.kind === 'jev_malformed') return text('kindInvalid')
  if (check.kind.endsWith('timeout')) return text('kindTimeout')
  if (check.kind === 'jev_rate_limited' || status === 402 || status === 429) return text('kindRateLimited', { status })
  if (status >= 500 || check.kind === 'jev_upstream') return text('kindUpstream', { status })
  if (status > 0) return text('kindHttp', { status })
  return text('kindError')
}

async function save() {
  if (saving.value) return
  saving.value = true
  try {
    await saveAiModerationSettings(formInput())
    adminToast.success(text('saved'))
    await load()
  } catch (err) {
    adminToast.error(err, text('saveFailed'))
  } finally {
    saving.value = false
  }
}

// 命中违规时的处理：统一转人工 / 统一直接拦截；各规则动作不一致时显示“按规则分别设置”。
const violationAction = computed<'review' | 'block' | 'mixed'>(() => {
  const actions = new Set(form.policies.map(rule => rule.action))
  return actions.size === 1 ? [...actions][0] : 'mixed'
})

function setViolationAction(action: 'review' | 'block') {
  form.policies.forEach((rule) => { rule.action = action })
}

const showAdvanced = ref(false)
const fmt = (value: unknown) => (Number(value) || 0).toFixed(2)
const pct = (value: number) => `${Math.round(Math.min(Math.max(value, 0), 1) * 100)}%`

// 与后端 BlockThresholdFor 一致：规则覆盖优先，拦截线不低于转人工线。
function ruleLines(rule: AiModerationPolicyRule) {
  const review = optionalNumber(rule.reviewThreshold) ?? (Number(form.defaultReviewThreshold) || 0)
  const block = Math.max(optionalNumber(rule.blockThreshold) ?? (Number(form.defaultBlockThreshold) || 0), review)
  return { review, block }
}

const globalLines = computed(() => {
  const review = Number(form.defaultReviewThreshold) || 0
  return { review, block: Math.max(Number(form.defaultBlockThreshold) || 0, review) }
})

function ruleSummary(rule: AiModerationPolicyRule) {
  const { review, block } = ruleLines(rule)
  if (rule.action === 'block') {
    return text('ruleSummaryBlock', { review: fmt(review), block: fmt(block), severity: (Number(form.severityBlockThreshold) || 0).toFixed(1) })
  }
  return text('ruleSummaryReview', { review: fmt(review) })
}

const globalOutcomeText = computed(() => {
  const values = { review: fmt(globalLines.value.review), block: fmt(globalLines.value.block) }
  if (violationAction.value === 'block') return text('onBlockLineBlockHint', values)
  if (violationAction.value === 'review') return text('onBlockLineReviewHint', values)
  return text('onBlockLineMixedHint')
})

const policyLabels = computed(() => Object.fromEntries(form.policies.map(rule => [rule.key, rule.label])))

function applyPreset(preset: typeof JEV_PRESETS[number]) {
  form.jevEndpoint = preset.endpoint
  form.jevModel = preset.model
}

// ---- 决策记录与人工标注 ----
const decisions = ref<AiModerationDecision[]>([])
const decisionsLoading = ref(false)
const decisionPage = ref(1)
const decisionTotal = ref(0)
const decisionPageSize = 10
const filters = reactive({ finalAction: '', humanAction: '', mode: '' })
const expanded = ref<number | null>(null)
const decisionPages = computed(() => Math.max(1, Math.ceil(decisionTotal.value / decisionPageSize)))

async function loadDecisions() {
  decisionsLoading.value = true
  try {
    const result = await listAiModerationDecisions({
      page: decisionPage.value,
      pageSize: decisionPageSize,
      finalAction: filters.finalAction || undefined,
      humanAction: filters.humanAction || undefined,
      mode: filters.mode || undefined,
    })
    decisions.value = result.items || []
    decisionTotal.value = result.total || 0
  } catch (err) {
    adminToast.error(err, text('decisionsFailed'))
  } finally {
    decisionsLoading.value = false
  }
}

function changeDecisionPage(next: number) {
  decisionPage.value = next
  void loadDecisions()
}

function applyFilters() {
  decisionPage.value = 1
  void loadDecisions()
}

async function label(item: AiModerationDecision, value: 'approved' | 'rejected') {
  try {
    await labelAiModerationDecision(item.id, value)
    item.humanAction = value
  } catch (err) {
    adminToast.error(err, text('labelFailed'))
  }
}

function topProbabilities(item: AiModerationDecision) {
  return Object.entries(item.signals?.ruleProbabilities ?? {})
    .sort((a, b) => b[1] - a[1])
    .slice(0, 3)
}

function actionVariant(action: string) {
  if (action === 'block') return 'destructive'
  if (action === 'review') return 'secondary'
  return 'outline'
}

function subjectLink(item: AiModerationDecision) {
  return item.subjectType === 'topic' && item.subjectId > 0 ? `/p/post/${item.subjectId}` : ''
}

function formatTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getMonth() + 1}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// ---- 离线阈值回放 ----
const replay = ref<AiModerationReplayReport | null>(null)
const replaying = ref(false)

async function runReplay() {
  replaying.value = true
  try {
    replay.value = await replayAiModeration(options())
  } catch (err) {
    adminToast.error(err, text('replayFailed'))
  } finally {
    replaying.value = false
  }
}

onMounted(() => {
  void load()
  void loadDecisions()
})
</script>

<template>
  <BasicPage :title="text('title')" :description="text('description')" sticky>
    <template #actions>
      <Button variant="outline" size="sm" type="button" :disabled="loading" @click="load">
        <RefreshCw class="size-4" :class="loading ? 'animate-spin' : ''" />{{ text('reload') }}
      </Button>
      <Button size="sm" type="button" :disabled="saving || loading" @click="save">
        <Save class="size-4" />{{ text('save') }}
      </Button>
    </template>

    <div class="max-w-4xl space-y-6">
      <AdminSection>
        <div class="space-y-5 p-4">
          <div class="flex items-center justify-between gap-4 rounded-lg border bg-muted/10 p-4">
            <div>
              <div class="flex items-center gap-2 text-base font-medium"><ScanEye class="size-4" />{{ text('enabled') }}</div>
              <p class="mt-1 text-sm text-muted-foreground">{{ text('enabledHint') }}</p>
            </div>
            <Switch v-model="form.enabled" />
          </div>
          <div class="grid gap-2 text-sm font-medium">
            {{ text('mode') }}
            <div class="flex w-fit max-w-full flex-wrap rounded-lg border bg-muted/20 p-1">
              <button
                v-for="value in ['shadow', 'enforce', 'deferred'] as const"
                :key="value"
                type="button"
                class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
                :class="form.mode === value ? 'bg-background shadow-xs' : 'text-muted-foreground hover:text-foreground'"
                @click="form.mode = value"
              >
                {{ text(`mode${cap(value)}`) }}
              </button>
            </div>
            <span class="text-xs font-normal text-muted-foreground">{{ text(`modeHint${cap(form.mode)}`) }}</span>
          </div>
          <label class="flex items-center justify-between gap-4 text-sm font-medium">
            <span>
              {{ text('textModeration') }}
              <span class="block text-xs font-normal text-muted-foreground">{{ text('textModerationHint') }}</span>
            </span>
            <Switch v-model="form.textModeration" />
          </label>
          <p class="flex gap-2 rounded-md border border-warning/30 bg-warning/5 p-3 text-xs leading-5 text-muted-foreground">
            <ShieldAlert class="mt-0.5 size-4 shrink-0 text-warning" />{{ text('privacyNotice') }}
          </p>
        </div>
      </AdminSection>

      <AdminSection>
        <div class="grid gap-5 p-4">
          <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium"><Gauge class="size-5 text-muted-foreground" />{{ text('jevTitle') }}</div>
          <div class="flex flex-wrap gap-2">
            <Button v-for="preset in JEV_PRESETS" :key="preset.label" type="button" variant="outline" size="sm" @click="applyPreset(preset)">
              {{ text('preset', { name: preset.label }) }}
            </Button>
          </div>
          <label class="grid gap-2 text-sm font-medium">
            {{ text('jevEndpoint') }}
            <Input v-model="form.jevEndpoint" placeholder="https://openrouter.ai/api/alpha/decisions" />
          </label>
          <label class="grid gap-2 text-sm font-medium">
            {{ text('jevModel') }}
            <Input v-model="form.jevModel" placeholder="typesafe/jev-1.13" />
          </label>
          <label class="grid gap-2 text-sm font-medium">
            <span class="flex items-center gap-2"><KeyRound class="size-4" />{{ text('jevApiKey') }}</span>
            <div class="flex items-center gap-2">
              <Input v-model="form.jevApiKey" type="password" autocomplete="new-password" placeholder="sk-..." :disabled="form.clearJevApiKey" />
              <Badge :variant="form.jevApiKeyConfigured ? 'default' : 'outline'" class="shrink-0">
                {{ form.jevApiKeyConfigured ? text('keyConfigured') : text('keyMissing') }}
              </Badge>
            </div>
            <div v-if="form.jevApiKeyConfigured" class="flex items-center gap-2 text-xs font-normal">
              <Button v-if="!form.clearJevApiKey" type="button" variant="outline" size="sm" class="h-7 text-xs text-destructive hover:text-destructive" @click="form.clearJevApiKey = true; form.jevApiKey = ''">
                <Trash2 class="size-3.5" />{{ text('clearKey') }}
              </Button>
              <template v-else>
                <Badge variant="destructive">{{ text('clearKeyPending') }}</Badge>
                <Button type="button" variant="ghost" size="sm" class="h-7 text-xs" @click="form.clearJevApiKey = false">
                  <Undo2 class="size-3.5" />{{ text('clearKeyUndo') }}
                </Button>
              </template>
            </div>
            <span class="text-xs font-normal text-muted-foreground">{{ text('keyHint') }}</span>
          </label>
          <div class="grid gap-5 sm:grid-cols-2">
            <label class="grid gap-2 text-sm font-medium">{{ text('timeoutMs') }}<Input v-model.number="form.jevTimeoutMs" type="number" min="1000" max="60000" step="500" /></label>
            <label class="grid gap-2 text-sm font-medium">
              {{ text('jevRetries') }}
              <select v-model.number="form.jevRetries" class="h-9 rounded-md border bg-background px-2 text-sm">
                <option :value="0">0</option>
                <option :value="1">1</option>
              </select>
              <span class="text-xs font-normal text-muted-foreground">{{ text('jevRetriesHint') }}</span>
            </label>
          </div>
          <div class="flex flex-wrap items-center gap-3">
            <Button type="button" variant="secondary" size="sm" :disabled="testing.jev" @click="testConnection('jev')">
              <Loader2 v-if="testing.jev" class="size-4 animate-spin" /><PlugZap v-else class="size-4" />{{ text('testConnection') }}
            </Button>
            <span v-if="testResults.jev" class="flex items-center gap-1.5 text-xs" :class="testResults.jev.ok ? 'text-success' : 'text-destructive'">
              <CheckCircle2 v-if="testResults.jev.ok" class="size-4" /><XCircle v-else class="size-4" />{{ testResultText('jev', testResults.jev) }}
            </span>
            <span v-else class="text-xs text-muted-foreground">{{ text('testHint') }}</span>
          </div>
        </div>
      </AdminSection>

      <AdminSection>
        <div class="grid gap-5 p-4">
          <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium"><Eye class="size-5 text-muted-foreground" />{{ text('visionTitle') }}</div>
          <p class="text-xs text-muted-foreground">{{ text('visionHint') }}</p>
          <label class="grid gap-2 text-sm font-medium">
            {{ text('visionBaseUrl') }}
            <Input v-model="form.visionBaseUrl" placeholder="https://openrouter.ai/api/v1" />
          </label>
          <label class="grid gap-2 text-sm font-medium">
            {{ text('visionModel') }}
            <Input v-model="form.visionModel" placeholder="inclusionai/ling-3.0-flash-vl" />
            <span class="text-xs font-normal text-muted-foreground">{{ text('visionModelHint') }}</span>
          </label>
          <label class="grid gap-2 text-sm font-medium">
            <span class="flex items-center gap-2"><KeyRound class="size-4" />{{ text('visionApiKey') }}</span>
            <div class="flex items-center gap-2">
              <Input v-model="form.visionApiKey" type="password" autocomplete="new-password" placeholder="sk-..." :disabled="form.clearVisionApiKey" />
              <Badge :variant="form.visionApiKeyConfigured ? 'default' : 'outline'" class="shrink-0">
                {{ form.visionApiKeyConfigured ? text('keyConfigured') : text('keyMissing') }}
              </Badge>
            </div>
            <div v-if="form.visionApiKeyConfigured" class="flex items-center gap-2 text-xs font-normal">
              <Button v-if="!form.clearVisionApiKey" type="button" variant="outline" size="sm" class="h-7 text-xs text-destructive hover:text-destructive" @click="form.clearVisionApiKey = true; form.visionApiKey = ''">
                <Trash2 class="size-3.5" />{{ text('clearKey') }}
              </Button>
              <template v-else>
                <Badge variant="destructive">{{ text('clearKeyPending') }}</Badge>
                <Button type="button" variant="ghost" size="sm" class="h-7 text-xs" @click="form.clearVisionApiKey = false">
                  <Undo2 class="size-3.5" />{{ text('clearKeyUndo') }}
                </Button>
              </template>
            </div>
          </label>
          <label class="grid gap-2 text-sm font-medium sm:max-w-xs">{{ text('timeoutMs') }}<Input v-model.number="form.visionTimeoutMs" type="number" min="1000" max="60000" step="500" /></label>
          <div class="flex flex-wrap items-center gap-3">
            <Button type="button" variant="secondary" size="sm" :disabled="testing.vision" @click="testConnection('vision')">
              <Loader2 v-if="testing.vision" class="size-4 animate-spin" /><PlugZap v-else class="size-4" />{{ text('testConnection') }}
            </Button>
            <span v-if="testResults.vision" class="flex items-center gap-1.5 text-xs" :class="testResults.vision.ok ? 'text-success' : 'text-destructive'">
              <CheckCircle2 v-if="testResults.vision.ok" class="size-4" /><XCircle v-else class="size-4" />{{ testResultText('vision', testResults.vision) }}
            </span>
            <span v-else class="text-xs text-muted-foreground">{{ text('testHint') }}</span>
          </div>
        </div>
      </AdminSection>

      <AdminSection>
        <div class="grid gap-5 p-4">
          <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium"><ShieldAlert class="size-5 text-muted-foreground" />{{ text('policyTitle') }}</div>
          <p class="text-sm text-muted-foreground">{{ text('policyHint') }}</p>

          <div class="grid gap-4 rounded-lg border bg-muted/10 p-4">
            <div class="grid gap-1.5">
              <div class="flex h-2.5 overflow-hidden rounded-full bg-muted">
                <div class="bg-success/70" :style="{ width: pct(globalLines.review) }" />
                <div class="bg-warning/70" :style="{ width: pct(globalLines.block - globalLines.review) }" />
                <div :class="violationAction === 'review' ? 'bg-warning/70' : 'bg-destructive/70'" :style="{ width: pct(1 - globalLines.block) }" />
              </div>
              <div class="flex justify-between text-xs text-muted-foreground">
                <span>{{ text('bandPublish') }}</span>
                <span>{{ text('bandReview') }}</span>
                <span>{{ violationAction === 'review' ? text('bandReview') : text('bandBlock') }}</span>
              </div>
            </div>
            <div class="grid gap-4 sm:grid-cols-2">
              <label class="grid gap-2 text-sm font-medium">
                {{ text('reviewLine') }}
                <Input v-model.number="form.defaultReviewThreshold" type="number" min="0" max="1" step="0.01" />
                <span class="text-xs font-normal text-muted-foreground">{{ text('reviewLineHint') }}</span>
              </label>
              <label class="grid gap-2 text-sm font-medium">
                {{ text('blockLine') }}
                <Input v-model.number="form.defaultBlockThreshold" type="number" min="0" max="1" step="0.01" />
                <span class="text-xs font-normal text-muted-foreground">{{ text('blockLineHint') }}</span>
              </label>
            </div>
            <div class="grid gap-2 text-sm font-medium">
              {{ text('onBlockLine') }}
              <div class="inline-flex w-fit rounded-lg border bg-muted/20 p-1">
                <button
                  v-for="value in ['block', 'review'] as const"
                  :key="value"
                  type="button"
                  class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
                  :class="violationAction === value ? 'bg-background shadow-xs' : 'text-muted-foreground hover:text-foreground'"
                  @click="setViolationAction(value)"
                >
                  {{ text(value === 'block' ? 'onBlockLineBlock' : 'onBlockLineReview') }}
                </button>
                <span
                  class="rounded-md px-3 py-1.5 text-sm font-medium"
                  :class="violationAction === 'mixed' ? 'bg-background shadow-xs' : 'text-muted-foreground'"
                >{{ text('onBlockLineMixed') }}</span>
              </div>
              <span class="text-xs font-normal leading-5 text-muted-foreground">{{ globalOutcomeText }}</span>
            </div>
          </div>

          <div class="divide-y rounded-lg border">
            <div v-for="rule in form.policies" :key="rule.key" class="grid gap-3 p-4" :class="rule.enabled ? '' : 'opacity-60'">
              <div class="flex flex-wrap items-center gap-3">
                <Switch v-model="rule.enabled" :aria-label="text('ruleEnabled')" />
                <Input v-model="rule.label" class="h-8 w-40" :aria-label="text('ruleName')" />
                <code class="text-xs text-muted-foreground">{{ rule.key }}</code>
                <label class="ml-auto flex items-center gap-2 text-xs text-muted-foreground">
                  {{ text('onBlockLine') }}
                  <select v-model="rule.action" class="h-8 rounded-md border bg-background px-2 text-sm text-foreground" :disabled="!rule.enabled">
                    <option value="block">{{ text('onBlockLineBlock') }}</option>
                    <option value="review">{{ text('onBlockLineReview') }}</option>
                  </select>
                </label>
              </div>
              <div v-if="rule.enabled" class="grid gap-1.5">
                <div class="flex h-1.5 overflow-hidden rounded-full bg-muted">
                  <div class="bg-success/70" :style="{ width: pct(ruleLines(rule).review) }" />
                  <div class="bg-warning/70" :style="{ width: pct(ruleLines(rule).block - ruleLines(rule).review) }" />
                  <div :class="rule.action === 'block' ? 'bg-destructive/70' : 'bg-warning/70'" :style="{ width: pct(1 - ruleLines(rule).block) }" />
                </div>
                <p class="text-xs leading-5 text-muted-foreground">{{ ruleSummary(rule) }}</p>
              </div>
              <label class="grid gap-1 text-xs text-muted-foreground">
                {{ text('ruleDefinition') }}
                <Textarea v-model="rule.definition" rows="2" :disabled="!rule.enabled" class="text-sm text-foreground" />
              </label>
              <div class="grid gap-3 sm:grid-cols-2">
                <label class="grid gap-1 text-xs text-muted-foreground">{{ text('ruleReviewLine') }}<Input v-model.number="rule.reviewThreshold" type="number" min="0" max="1" step="0.01" :placeholder="text('useDefault', { value: fmt(form.defaultReviewThreshold) })" :disabled="!rule.enabled" /></label>
                <label class="grid gap-1 text-xs text-muted-foreground">{{ text('ruleBlockLine') }}<Input v-model.number="rule.blockThreshold" type="number" min="0" max="1" step="0.01" :placeholder="text('useDefault', { value: fmt(form.defaultBlockThreshold) })" :disabled="!rule.enabled || rule.action !== 'block'" /></label>
              </div>
            </div>
          </div>

          <div class="rounded-lg border">
            <button type="button" class="flex w-full items-center justify-between px-4 py-3 text-sm font-medium" :aria-expanded="showAdvanced" @click="showAdvanced = !showAdvanced">
              {{ text('advancedTitle') }}
              <ChevronDown class="size-4 transition-transform" :class="showAdvanced ? 'rotate-180' : ''" />
            </button>
            <div v-if="showAdvanced" class="grid gap-4 border-t p-4 sm:grid-cols-2">
              <label class="grid gap-2 text-sm font-medium">
                {{ text('severityLine') }}
                <Input v-model.number="form.severityBlockThreshold" type="number" min="0" max="3" step="0.1" />
                <span class="text-xs font-normal text-muted-foreground">{{ text('severityLineHint') }}</span>
              </label>
              <label class="grid gap-2 text-sm font-medium">
                {{ text('reviewNeededLine') }}
                <Input v-model.number="form.reviewNeededThreshold" type="number" min="0" max="1" step="0.01" />
                <span class="text-xs font-normal text-muted-foreground">{{ text('reviewNeededLineHint') }}</span>
              </label>
              <label class="grid gap-2 text-sm font-medium">
                {{ text('policyRevision') }}
                <Input v-model="form.policyRevision" />
                <span class="text-xs font-normal text-muted-foreground">{{ text('policyRevisionHint') }}</span>
              </label>
            </div>
          </div>
        </div>
      </AdminSection>

      <AdminSection>
        <div class="grid gap-5 p-4">
          <div class="border-b pb-2 text-lg font-medium">{{ text('guardTitle') }}</div>
          <div class="grid gap-5 sm:grid-cols-2">
            <label class="grid gap-2 text-sm font-medium">
              {{ text('externalImageAction') }}
              <select v-model="form.externalImageAction" class="h-9 rounded-md border bg-background px-2 text-sm">
                <option value="review">{{ text('actionReview') }}</option>
                <option value="block">{{ text('actionBlock') }}</option>
              </select>
              <span class="text-xs font-normal text-muted-foreground">{{ text('externalImageHint') }}</span>
            </label>
            <label class="grid gap-2 text-sm font-medium">{{ text('maxImages') }}<Input v-model.number="form.maxImagesPerDecision" type="number" min="1" max="20" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ text('globalPerMinute') }}<Input v-model.number="form.globalRequestsPerMinute" type="number" min="1" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ text('perUserPerMinute') }}<Input v-model.number="form.perUserRequestsPerMinute" type="number" min="1" /></label>
          </div>
          <p class="text-xs text-muted-foreground">{{ text('guardHint') }}</p>
        </div>
      </AdminSection>

      <AdminSection>
        <template #header>
          <div class="flex flex-wrap items-center justify-between gap-3 px-1 py-1">
            <div class="text-base font-medium">{{ text('decisionsTitle') }}</div>
            <div class="flex flex-wrap items-center gap-2">
              <select v-model="filters.mode" class="h-8 rounded-md border bg-background px-2 text-xs" @change="applyFilters">
                <option value="">{{ text('filterAllModes') }}</option>
                <option value="shadow">{{ text('modeShadow') }}</option>
                <option value="enforce">{{ text('modeEnforce') }}</option>
                <option value="deferred">{{ text('modeDeferred') }}</option>
              </select>
              <select v-model="filters.finalAction" class="h-8 rounded-md border bg-background px-2 text-xs" @change="applyFilters">
                <option value="">{{ text('filterAllActions') }}</option>
                <option value="allow">{{ text('actionAllow') }}</option>
                <option value="review">{{ text('actionReview') }}</option>
                <option value="block">{{ text('actionBlock') }}</option>
              </select>
              <select v-model="filters.humanAction" class="h-8 rounded-md border bg-background px-2 text-xs" @change="applyFilters">
                <option value="">{{ text('filterAllLabels') }}</option>
                <option value="none">{{ text('labelNone') }}</option>
                <option value="approved">{{ text('labelApproved') }}</option>
                <option value="rejected">{{ text('labelRejected') }}</option>
              </select>
              <Button variant="outline" size="icon" type="button" class="size-8" :disabled="decisionPage <= 1" @click="changeDecisionPage(decisionPage - 1)"><ChevronLeft class="size-4" /></Button>
              <span class="text-xs text-muted-foreground">{{ decisionPage }} / {{ decisionPages }}</span>
              <Button variant="outline" size="icon" type="button" class="size-8" :disabled="decisionPage >= decisionPages" @click="changeDecisionPage(decisionPage + 1)"><ChevronRight class="size-4" /></Button>
              <Button variant="outline" size="icon" type="button" class="size-8" :disabled="decisionsLoading" @click="loadDecisions"><RefreshCw class="size-4" :class="decisionsLoading ? 'animate-spin' : ''" /></Button>
            </div>
          </div>
        </template>
        <div v-if="decisions.length === 0" class="px-4 py-10 text-center text-sm text-muted-foreground">{{ decisionsLoading ? text('loading') : text('decisionsEmpty') }}</div>
        <div v-else class="divide-y">
          <article v-for="item in decisions" :key="item.id" class="space-y-2 px-4 py-3 text-sm">
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-mono text-xs text-muted-foreground">#{{ item.id }} · {{ formatTime(item.createdAt) }}</span>
              <Badge variant="outline">{{ text(`mode${cap(item.mode)}`) }}</Badge>
              <Badge :variant="actionVariant(item.finalAction)">{{ text(`action${cap(item.finalAction)}`) }}</Badge>
              <a v-if="subjectLink(item)" :href="subjectLink(item)" target="_blank" class="text-xs text-primary hover:underline">{{ item.subjectType }} #{{ item.subjectId }}</a>
              <span v-else class="text-xs text-muted-foreground">{{ item.subjectType }} #{{ item.subjectId || '-' }}</span>
              <span v-if="item.errorKind" class="font-mono text-xs text-muted-foreground">{{ item.errorKind }}</span>
              <div class="ml-auto flex items-center gap-1.5">
                <Badge v-if="item.humanAction" :variant="item.humanAction === 'rejected' ? 'destructive' : 'default'">{{ text(`label${cap(item.humanAction)}`) }}</Badge>
                <Button type="button" size="sm" variant="outline" class="h-7 text-xs" @click="label(item, 'approved')"><Check class="size-3.5" />{{ text('markApproved') }}</Button>
                <Button type="button" size="sm" variant="outline" class="h-7 text-xs text-destructive hover:text-destructive" @click="label(item, 'rejected')"><X class="size-3.5" />{{ text('markRejected') }}</Button>
              </div>
            </div>
            <ul v-if="item.reasons?.length" class="space-y-0.5 text-xs leading-5">
              <li v-for="(reason, index) in item.reasons" :key="index">{{ reasonText(t, reason, policyLabels) }}</li>
            </ul>
            <p v-else-if="item.finalAction === 'allow'" class="text-xs leading-5">{{ text('reasonClean') }}</p>
            <div class="flex flex-wrap gap-2 text-xs text-muted-foreground">
              <span v-for="[key, value] in topProbabilities(item)" :key="key" :class="item.triggeredPolicies.includes(key) ? 'font-semibold text-foreground' : ''">{{ policyName(t, key, policyLabels) }} {{ value.toFixed(2) }}</span>
              <span v-if="item.signals?.severity !== undefined">{{ text('metricSeverity') }} {{ item.signals.severity.toFixed(1) }}</span>
              <span v-if="item.signals?.reviewNeeded !== undefined">{{ text('metricReviewNeeded') }} {{ item.signals.reviewNeeded.toFixed(2) }}</span>
              <span>{{ item.latencyMs }} ms</span>
              <button v-if="item.images.length" type="button" class="text-primary hover:underline" @click="expanded = expanded === item.id ? null : item.id">{{ text('imagesCount', { count: item.images.length }) }}</button>
            </div>
            <ul v-if="expanded === item.id" class="space-y-1 rounded-md bg-muted/30 p-2 text-xs">
              <li v-for="(image, index) in item.images" :key="index" class="break-all">
                <Badge variant="outline" class="mr-1 px-1 py-0 text-[10px]">{{ image.status }}</Badge>{{ image.url || image.fileName }}
                <p v-if="image.evidence" class="mt-0.5 text-muted-foreground">{{ image.evidence }}</p>
              </li>
            </ul>
          </article>
        </div>
      </AdminSection>

      <AdminSection>
        <div class="grid gap-4 p-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <div class="text-base font-medium">{{ text('replayTitle') }}</div>
              <p class="mt-1 text-xs text-muted-foreground">{{ text('replayHint') }}</p>
            </div>
            <Button type="button" variant="secondary" :disabled="replaying" @click="runReplay">
              <RefreshCw class="size-4" :class="replaying ? 'animate-spin' : ''" />{{ text('replayRun') }}
            </Button>
          </div>
          <div v-if="replay" class="grid gap-3 text-sm">
            <div class="flex flex-wrap gap-4 text-xs text-muted-foreground">
              <span>{{ text('replaySamples', { count: replay.samples }) }}</span>
              <span>{{ text('replayFalseBlock', { count: replay.falseBlock }) }}</span>
              <span>{{ text('replayMissed', { count: replay.missedViolation }) }}</span>
              <span>{{ text('replayReviewRate', { rate: (replay.reviewRate * 100).toFixed(1) }) }}</span>
              <span>{{ text('replayChanged', { count: replay.changed }) }}</span>
            </div>
            <table class="w-full max-w-md border text-center text-xs">
              <thead class="bg-muted/30">
                <tr>
                  <th class="border p-1.5">{{ text('replayHuman') }}</th>
                  <th v-for="action in ['allow', 'review', 'block'] as const" :key="action" class="border p-1.5">{{ text(`action${cap(action)}`) }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="human in ['approved', 'rejected'] as const" :key="human">
                  <td class="border p-1.5">{{ text(`label${cap(human)}`) }}</td>
                  <td v-for="action in ['allow', 'review', 'block'] as const" :key="action" class="border p-1.5 font-mono">{{ replay.matrix[human]?.[action] ?? 0 }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </AdminSection>
    </div>
  </BasicPage>
</template>
