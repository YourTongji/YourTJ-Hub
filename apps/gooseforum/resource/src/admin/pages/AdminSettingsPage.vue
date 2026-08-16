<script setup lang="ts">
import { adminText } from '@/admin/runtime/i18n-text'
import httpNotifyGuideZh from '@/admin/docs/http-notify-guide.zh.md?raw'
import httpNotifyGuideEn from '@/admin/docs/http-notify-guide.en.md?raw'
import httpNotifyGuideJa from '@/admin/docs/http-notify-guide.ja.md?raw'

import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import MarkdownIt from 'markdown-it'
import { Bot, CheckCircle2, Code, FileText, Globe, GripVertical, HardDrive, KeyRound, Loader2, MailCheck, Plus, RefreshCw, Save, ScrollText, Send, Shield, Sparkles, Trash2, Upload, Webhook } from '@lucide/vue'
import AdminActionButton from '@/admin/components/AdminActionButton.vue'
import { BasicPage } from '@/admin/components/global-layout'
import { Button } from '@/admin/components/ui/button'
import { Badge } from '@/admin/components/ui/badge'
import { Input } from '@/admin/components/ui/input'
import { Textarea } from '@/admin/components/ui/textarea'
import { Switch } from '@/admin/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/admin/components/ui/tabs'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/admin/components/ui/dialog'
import {
  createStorageMigrateTask,
  getAiSummarySettings,
  getAnnouncement,
  getMCPSettings,
  getMailSettings,
  getHttpNotifySettings,
  getOnesystemSettings,
  getPkSyncStatus,
  getPostingSettings,
  getRateLimitSettings,
  getSecuritySettings,
  getSiteSettings,
  getStorageMigrateTasks,
  getStorageSettings,
  getTermsOfService,
  saveAnnouncement,
  saveHttpNotifySettings,
  saveMailSettings,
  saveOnesystemSettings,
  savePostingSettings,
  saveAiSummarySettings,
  saveMCPSettings,
  saveRateLimitSettings,
  saveSecuritySettings,
  saveSiteSettings,
  saveStorageSettings,
  saveTermsOfService,
  syncPkCalendar,
  testMailConnection,
  testStorageConnection,
  uploadAdminImage,
} from '@/admin/runtime/api'
import { adminToast } from '@/admin/runtime/toast'
import { resolveApiMessage } from '@/runtime/api-message'
import type {
  AdminPayload,
  AiSummarySettings,
  ManageHomeProps,
  AdminTaskRow,
  AnnouncementConfig,
  HttpNotifyEndpoint,
  HttpNotifySettings,
  MailSettings,
  MCPSettings,
  OnesystemSettings,
  PkSyncStatusItem,
  PostingSettings,
  RateLimitSettings,
  SecuritySettings,
  SiteSettings,
  StorageSettings,
  TermsOfServiceConfig,
} from '@/admin/types'

type Kind = 'site-info' | 'mail' | 'security' | 'posting' | 'rate-limit' | 'mcp' | 'ai-summary' | 'http-notify' | 'announcement' | 'storage' | 'terms' | 'onesystem'

const props = defineProps<{
  payload: AdminPayload<ManageHomeProps>
  kind: Kind
}>()

const { locale } = useI18n()
const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const httpNotifyTab = ref('config')
const error = ref('')
const testEmail = ref('')
const newAllowedDomain = ref('')
const newExtension = ref('')
const newReservedUsername = ref('')
const newBannedUsername = ref('')
const newSensitiveWord = ref('')
const clearAfterMigrate = ref(false)
const migrateTasks = ref<AdminTaskRow[]>([])
const migrateConfirm = ref(false)
const migrating = ref(false)

// ---- 一系统同步（issue #248 排课数据自愈入口）----
const onesystemForm = reactive<{ cookie: string, cookieConfigured: boolean }>({
  cookie: '',
  cookieConfigured: false,
})
const savingCookie = ref(false)
const syncForm = reactive<{ term: string, depth: number }>({ term: '', depth: 1 })
const syncingPk = ref(false)
const syncStatusItems = ref<PkSyncStatusItem[]>([])
const syncStatusLoading = ref(false)
let syncPollTimer: ReturnType<typeof setInterval> | null = null

interface MigrateTaskPayload {
  lastId?: number
  total?: number
  processed?: number
  failed?: number
  clearAfterMigrate?: boolean
}

function parseMigrateTask(task: AdminTaskRow): MigrateTaskPayload {
  try {
    const raw = JSON.parse(task.taskJson || '{}') as unknown
    if (typeof raw === 'object' && raw !== null) {
      return raw as MigrateTaskPayload
    }
    return {}
  } catch {
    return {}
  }
}

// 渲染用：为每条迁移任务预解析 taskJson，展示进度与失败数，避免模板内重复解析。
const migrateTaskRows = computed(() =>
  migrateTasks.value.map((task) => ({ ...task, payload: parseMigrateTask(task) }))
)

const siteForm = reactive<SiteSettings>({
  siteName: '',
  siteUrl: '',
  siteLogo: '',
  siteEmail: '',
  siteDescription: '',
  siteKeywords: '',
  externalLinks: '',
})

const mailForm = reactive<MailSettings>({
  enableMail: true,
  smtpHost: '',
  smtpPort: 587,
  useSSL: false,
  smtpUsername: '',
  smtpPassword: '',
  fromName: '',
  fromEmail: '',
})

const securityForm = reactive<SecuritySettings>({
  enableSignup: true,
  enableEmailVerification: false,
  allowedDomains: [],
  reservedUsernames: [],
  bannedUsernames: [],
  sensitiveWords: [],
  sensitiveAction: 'block',
  captchaRequired: true,
})

const rateLimitForm = reactive<RateLimitSettings>({
  enabled: true,
  skipAdmin: true,
  actions: [],
  newUserCaptchaAfterPosts: 3,
  newUserCaptchaDays: 7,
  minSubmitSeconds: 1,
})

const mcpForm = reactive<MCPSettings>({
  enabled: false,
  writes: false,
})
const aiSummaryForm = reactive<AiSummarySettings>({
  enabled: false,
  globalPerMinute: 5,
})

const postingForm = reactive<PostingSettings>({
  textControl: {
    minPostLength: 5,
    maxPostLength: 50000,
    minTitleLength: 5,
    maxTitleLength: 100,
    newUserPostCooldownMinutes: 0,
  },
  uploadControl: {
    allowAttachments: true,
    authorizedExtensions: ['.jpg', '.jpeg', '.png', '.gif', '.webp'],
    maxAttachmentSizeKb: 5120,
    maxDailyUploadsPerUser: 10,
    newUserUploadCooldownMinutes: 0,
  },
  llms: {
    enabled: false,
    fullText: false,
    files: false,
  },
})

const storageForm = reactive<StorageSettings>({
  provider: 'local',
  endpoint: '',
  bucket: '',
  region: '',
  bucketLookup: 'auto',
  secure: true,
  accessKey: '',
  secretKey: '',
  publicUrlPrefix: '',
})

const termsForm = reactive<TermsOfServiceConfig>({
  enabled: false,
  content: '',
})

const httpNotifyEvents = computed(() => {
  locale.value
  return [
    { value: 'topic.published', label: adminText('k00ck') },
    { value: 'topic.updated', label: adminText('k00cl') },
    { value: 'comment.created', label: adminText('k00cm') },
    { value: 'user.signup', label: adminText('k00cn') },
    { value: 'moderation.report.created', label: adminText('k00co') },
  ]
})

const httpNotifyForm = reactive<HttpNotifySettings>({
  enabled: false,
  endpoints: [],
})

const guideMarkdown = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
})

const announcementForm = reactive<AnnouncementConfig>({
  enabled: false,
  content: '',
  items: [],
})

const pageMeta = computed(() => {
  locale.value
  const meta: Record<Kind, { title: string, description: string }> = {
    'site-info': { title: adminText('k0001'), description: adminText('k0002') },
    mail: { title: adminText('k0003'), description: adminText('k0004') },
    security: { title: adminText('k0005'), description: adminText('k0006') },
    posting: { title: adminText('k0007'), description: adminText('k0008') },
    'rate-limit': { title: adminText('k00ig'), description: adminText('k00ij') },
    mcp: { title: adminText('k00mj'), description: adminText('k00mk') },
    'ai-summary': { title: adminText('k00p0'), description: adminText('k00p1') },
    'http-notify': { title: adminText('k00cj'), description: adminText('k00cp') },
    announcement: { title: adminText('k0009'), description: adminText('k000a') },
    storage: { title: adminText('k00fn'), description: adminText('k00fo') },
    terms: { title: adminText('k00gp'), description: adminText('k00gq') },
    onesystem: { title: adminText('k00s4'), description: adminText('k00s5') },
  }
  return meta[props.kind]
})
const allowedDomains = computed(() => {
  return securityForm.allowedDomains
})

const httpNotifyGuideHtml = computed(() => {
  const guideLocale = String(locale.value).split(/[-_]/)[0]
  const guides = {
    zh: httpNotifyGuideZh,
    en: httpNotifyGuideEn,
    ja: httpNotifyGuideJa,
  }
  return guideMarkdown.render(guides[guideLocale as keyof typeof guides] || httpNotifyGuideZh)
})

function toBool(value: unknown, fallback = false) {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value === 1
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    if (['true', '1', 'yes', 'on', 'enabled'].includes(normalized)) return true
    if (['false', '0', 'no', 'off', 'disabled', ''].includes(normalized)) return false
  }
  return fallback
}

function normalizeSite(settings: Partial<SiteSettings> = {}) {
  return {
    siteName: settings.siteName ?? '',
    siteUrl: settings.siteUrl ?? '',
    siteLogo: settings.siteLogo ?? '',
    siteEmail: settings.siteEmail ?? '',
    siteDescription: settings.siteDescription ?? '',
    siteKeywords: settings.siteKeywords ?? '',
    externalLinks: settings.externalLinks ?? '',
  } satisfies SiteSettings
}

function normalizeMail(settings: Partial<MailSettings> = {}) {
  return {
    enableMail: toBool(settings.enableMail, false),
    smtpHost: settings.smtpHost ?? '',
    smtpPort: Number(settings.smtpPort ?? 587),
    useSSL: toBool(settings.useSSL, false),
    smtpUsername: settings.smtpUsername ?? '',
    smtpPassword: settings.smtpPassword ?? '',
    fromName: settings.fromName ?? '',
    fromEmail: settings.fromEmail ?? '',
  } satisfies MailSettings
}

function normalizeSecurity(settings: Partial<SecuritySettings> = {}) {
  return {
    enableSignup: toBool(settings.enableSignup, true),
    enableEmailVerification: toBool(settings.enableEmailVerification, false),
    allowedDomains: Array.isArray(settings.allowedDomains)
      ? settings.allowedDomains.map(item => String(item).trim().toLowerCase()).filter(Boolean)
      : [],
    reservedUsernames: Array.isArray(settings.reservedUsernames)
      ? settings.reservedUsernames.map(item => String(item).trim()).filter(Boolean)
      : [],
    bannedUsernames: Array.isArray(settings.bannedUsernames)
      ? settings.bannedUsernames.map(item => String(item).trim()).filter(Boolean)
      : [],
    sensitiveWords: Array.isArray(settings.sensitiveWords)
      ? settings.sensitiveWords.map(item => String(item).trim()).filter(Boolean)
      : [],
    sensitiveAction: settings.sensitiveAction === 'review' ? 'review' : 'block',
    captchaRequired: toBool(settings.captchaRequired, true),
  } satisfies SecuritySettings
}

function normalizeRateLimit(settings: Partial<RateLimitSettings> = {}) {
  return {
    enabled: toBool(settings.enabled, true),
    skipAdmin: toBool(settings.skipAdmin, true),
    actions: Array.isArray(settings.actions)
      ? settings.actions.map(rule => ({
        action: rule.action ?? '',
        windowSeconds: Math.max(Number(rule.windowSeconds ?? 60), 1),
        limitPerIp: Math.max(Number(rule.limitPerIp ?? 0), 0),
        limitPerUser: Math.max(Number(rule.limitPerUser ?? 0), 0),
      }))
      : [],
    newUserCaptchaAfterPosts: Math.max(Number(settings.newUserCaptchaAfterPosts ?? 0), 0),
    newUserCaptchaDays: Math.max(Number(settings.newUserCaptchaDays ?? 7), 0),
    minSubmitSeconds: Math.max(Number(settings.minSubmitSeconds ?? 1), 0),
  } satisfies RateLimitSettings
}

function normalizeMCP(settings: Partial<MCPSettings> = {}) {
  return {
    enabled: toBool(settings.enabled, false),
    writes: toBool(settings.writes, false),
  } satisfies MCPSettings
}
function normalizeAiSummary(settings: Partial<AiSummarySettings> = {}) {
  return {
    enabled: toBool(settings.enabled, false),
    globalPerMinute: Math.max(Number(settings.globalPerMinute ?? 5), 0),
  } satisfies AiSummarySettings
}

function normalizePosting(settings: Partial<PostingSettings> = {}) {
  return {
    textControl: {
      minPostLength: Number(settings.textControl?.minPostLength ?? 5),
      maxPostLength: Number(settings.textControl?.maxPostLength ?? 50000),
      minTitleLength: Number(settings.textControl?.minTitleLength ?? 5),
      maxTitleLength: Number(settings.textControl?.maxTitleLength ?? 100),
      newUserPostCooldownMinutes: Number(settings.textControl?.newUserPostCooldownMinutes ?? 0),
    },
    uploadControl: {
      allowAttachments: toBool(settings.uploadControl?.allowAttachments, true),
      authorizedExtensions: Array.isArray(settings.uploadControl?.authorizedExtensions)
        ? settings.uploadControl.authorizedExtensions.map(item => String(item).trim().toLowerCase()).filter(Boolean)
        : ['.jpg', '.jpeg', '.png', '.gif', '.webp'],
      maxAttachmentSizeKb: Number(settings.uploadControl?.maxAttachmentSizeKb ?? 5120),
      maxDailyUploadsPerUser: Number(settings.uploadControl?.maxDailyUploadsPerUser ?? 10),
      newUserUploadCooldownMinutes: Number(settings.uploadControl?.newUserUploadCooldownMinutes ?? 1440),
    },
    llms: {
      enabled: toBool(settings.llms?.enabled, false),
      fullText: toBool(settings.llms?.fullText, false),
      files: toBool(settings.llms?.files, false),
    },
  } satisfies PostingSettings
}

function normalizeHttpNotify(settings: Partial<HttpNotifySettings> = {}) {
  return {
    enabled: toBool(settings.enabled, false),
    endpoints: Array.isArray(settings.endpoints)
      ? settings.endpoints.map(endpoint => normalizeEndpoint(endpoint)).filter(endpoint => endpoint.url)
      : [],
  } satisfies HttpNotifySettings
}

function normalizeEndpoint(endpoint: Partial<HttpNotifyEndpoint> = {}) {
  const events = Array.isArray(endpoint.events)
    ? endpoint.events.map(item => String(item).trim()).filter(Boolean)
    : []
  const enabled = toBool(endpoint.enabled, true)
  return {
    id: endpoint.id || crypto.randomUUID(),
    name: endpoint.name ?? '',
    enabled,
    url: endpoint.url?.trim() ?? '',
    secret: endpoint.secret ?? '',
    events,
    timeoutSeconds: Math.min(Math.max(Number(endpoint.timeoutSeconds ?? 2), 1), 15),
    failureCount: enabled ? 0 : Number(endpoint.failureCount ?? 0),
    lastError: enabled ? '' : endpoint.lastError ?? '',
    abnormalTerminated: enabled ? false : toBool(endpoint.abnormalTerminated, false),
  } satisfies HttpNotifyEndpoint
}

function isHttpUrl(value: string) {
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function validateHttpNotify(settings: HttpNotifySettings) {
  if (!settings.enabled) return true
  const enabledEndpoints = settings.endpoints.filter(endpoint => endpoint.enabled)
  if (enabledEndpoints.length === 0) {
    adminToast.warning(adminText('k00d2'))
    return false
  }
  for (const endpoint of enabledEndpoints) {
    const name = endpoint.name || endpoint.url || adminText('k00cw')
    if (!endpoint.url) {
      adminToast.warning(adminText('k00d3', { name }))
      return false
    }
    if (!isHttpUrl(endpoint.url)) {
      adminToast.warning(adminText('k00d4', { name }))
      return false
    }
    if (!endpoint.events.length) {
      adminToast.warning(adminText('k00d5', { name }))
      return false
    }
    if (!Number.isFinite(endpoint.timeoutSeconds) || endpoint.timeoutSeconds < 1 || endpoint.timeoutSeconds > 15) {
      adminToast.warning(adminText('k00d6', { name }))
      return false
    }
  }
  return true
}

function normalizeAnnouncement(settings: Partial<AnnouncementConfig> = {}) {
  const items = settings.items ?? []
  // 旧版单则数据迁移为一条 legacy 公告，保证编辑入口不丢失
  if (items.length === 0 && settings.content) {
    return {
      enabled: toBool(settings.enabled, false),
      content: settings.content ?? '',
      items: [{ id: 'legacy', title: '', content: settings.content, enabled: true }],
    } satisfies AnnouncementConfig
  }
  return {
    enabled: toBool(settings.enabled, false),
    content: settings.content ?? '',
    items,
  } satisfies AnnouncementConfig
}

function serializeAnnouncement(): AnnouncementConfig {
  const items = (announcementForm.items ?? []).filter((item) => item.content.trim())
  const isLegacyOnly = items.length === 1 && items[0].id === 'legacy'
  return {
    enabled: announcementForm.enabled,
    content: isLegacyOnly ? items[0].content : announcementForm.content,
    items: isLegacyOnly ? [] : items,
  }
}

function addAnnouncementItem() {
  if (!announcementForm.items) announcementForm.items = []
  announcementForm.items.push({
    id: `ann-${Date.now().toString(36)}`,
    title: '',
    content: '',
    enabled: true,
  })
}

function removeAnnouncementItem(index: number) {
  announcementForm.items?.splice(index, 1)
}

// 拖拽排序状态：dragstart 记录源下标，dragover 记录目标下标
const draggingAnnouncementIndex = ref<number | null>(null)
const dragOverAnnouncementIndex = ref<number | null>(null)

function onAnnouncementDragStart(index: number, event: DragEvent) {
  if (!event.dataTransfer) return
  draggingAnnouncementIndex.value = index
  event.dataTransfer.effectAllowed = 'move'
  event.dataTransfer.setData('text/plain', String(index))
}

function onAnnouncementDragOver(index: number, event: DragEvent) {
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'
  if (index !== draggingAnnouncementIndex.value) {
    dragOverAnnouncementIndex.value = index
  }
}

function onAnnouncementDrop(index: number, event: DragEvent) {
  event.preventDefault()
  moveAnnouncementItem(draggingAnnouncementIndex.value, index)
  dragOverAnnouncementIndex.value = null
}

function onAnnouncementDragEnd() {
  draggingAnnouncementIndex.value = null
  dragOverAnnouncementIndex.value = null
}

// 将公告从 fromIndex 移动到 toIndex（数组顺序即首页轮播的展示顺序）
function moveAnnouncementItem(fromIndex: number | null, toIndex: number) {
  if (fromIndex === null || fromIndex === toIndex) return
  const items = announcementForm.items ?? []
  if (fromIndex < 0 || fromIndex >= items.length || toIndex < 0 || toIndex > items.length) return
  const [moved] = items.splice(fromIndex, 1)
  items.splice(toIndex, 0, moved)
}

function normalizeStorage(settings: Partial<StorageSettings> = {}) {
  return {
    provider: settings.provider === 's3' ? 's3' : 'local',
    endpoint: settings.endpoint ?? '',
    bucket: settings.bucket ?? '',
    region: settings.region ?? '',
    bucketLookup: settings.bucketLookup === 'dns' || settings.bucketLookup === 'path' ? settings.bucketLookup : 'auto',
    secure: toBool(settings.secure, true),
    accessKey: settings.accessKey ?? '',
    secretKey: settings.secretKey ?? '',
    publicUrlPrefix: settings.publicUrlPrefix ?? '',
  } satisfies StorageSettings
}

function normalizeTerms(settings: Partial<TermsOfServiceConfig> = {}) {
  return {
    enabled: toBool(settings.enabled, false),
    content: settings.content ?? '',
  } satisfies TermsOfServiceConfig
}


async function uploadImage(target: 'siteLogo', event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    const data = await uploadAdminImage(file)
    siteForm[target] = data.url || ''
    adminToast.success(adminText('k000b'))
  } catch (err) {
    adminToast.error(err, adminText('k000c'))
  } finally {
    input.value = ''
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    if (props.kind === 'site-info') Object.assign(siteForm, normalizeSite(await getSiteSettings()))
    else if (props.kind === 'mail') Object.assign(mailForm, normalizeMail(await getMailSettings()))
    else if (props.kind === 'security') Object.assign(securityForm, normalizeSecurity(await getSecuritySettings()))
    else if (props.kind === 'posting') Object.assign(postingForm, normalizePosting(await getPostingSettings()))
    else if (props.kind === 'rate-limit') Object.assign(rateLimitForm, normalizeRateLimit(await getRateLimitSettings()))
    else if (props.kind === 'mcp') Object.assign(mcpForm, normalizeMCP(await getMCPSettings()))
    else if (props.kind === 'ai-summary') Object.assign(aiSummaryForm, normalizeAiSummary(await getAiSummarySettings()))
    else if (props.kind === 'http-notify') Object.assign(httpNotifyForm, normalizeHttpNotify(await getHttpNotifySettings()))
    else if (props.kind === 'storage') {
      Object.assign(storageForm, normalizeStorage(await getStorageSettings()))
      await loadMigrateTasks()
    }
    else if (props.kind === 'terms') Object.assign(termsForm, normalizeTerms(await getTermsOfService()))
    else if (props.kind === 'onesystem') {
      await Promise.all([loadOnesystem(), refreshSyncStatus()])
    }
    else Object.assign(announcementForm, normalizeAnnouncement(await getAnnouncement()))
  } catch (err) {
    error.value = err instanceof Error ? err.message : adminText('k000d')
  } finally {
    loading.value = false
  }
}

async function save() {
  const httpNotifySettings = props.kind === 'http-notify' ? normalizeHttpNotify(httpNotifyForm) : null
  if (httpNotifySettings && !validateHttpNotify(httpNotifySettings)) return

  saving.value = true
  try {
    if (props.kind === 'site-info') await saveSiteSettings(normalizeSite(siteForm))
    else if (props.kind === 'mail') await saveMailSettings(normalizeMail(mailForm))
    else if (props.kind === 'security') await saveSecuritySettings(normalizeSecurity(securityForm))
    else if (props.kind === 'posting') await savePostingSettings(normalizePosting(postingForm))
    else if (props.kind === 'rate-limit') await saveRateLimitSettings(normalizeRateLimit(rateLimitForm))
    else if (props.kind === 'mcp') await saveMCPSettings(normalizeMCP(mcpForm))
    else if (props.kind === 'ai-summary') await saveAiSummarySettings(normalizeAiSummary(aiSummaryForm))
    else if (props.kind === 'http-notify') await saveHttpNotifySettings(httpNotifySettings!)
    else if (props.kind === 'storage') await saveStorageSettings(normalizeStorage(storageForm))
    else if (props.kind === 'terms') await saveTermsOfService(normalizeTerms(termsForm))
    else await saveAnnouncement(serializeAnnouncement())
    adminToast.success(adminText('k000e'))
  } catch (err) {
    adminToast.error(err, adminText('k000f'))
  } finally {
    saving.value = false
  }
}

async function sendTestMail() {
  if (!testEmail.value.trim()) {
    adminToast.warning(adminText('k000g'))
    return
  }
  testing.value = true
  try {
    const response = await testMailConnection(normalizeMail(mailForm), testEmail.value.trim())
    adminToast.success(resolveApiMessage(response.result || response, adminText('k000h')))
  } catch (err) {
    adminToast.error(err, adminText('k000i'))
  } finally {
    testing.value = false
  }
}

async function testStorage() {
  testing.value = true
  try {
    const response = await testStorageConnection(normalizeStorage(storageForm))
    if (response.success) {
      adminToast.success(resolveApiMessage(response, adminText('k00g7')))
    } else {
      adminToast.error(new Error(resolveApiMessage(response, adminText('k00g8'))), adminText('k00g8'))
    }
  } catch (err) {
    adminToast.error(err, adminText('k00g8'))
  } finally {
    testing.value = false
  }
}

async function loadMigrateTasks() {
  try {
    migrateTasks.value = await getStorageMigrateTasks()
  } catch (err) {
    adminToast.error(err, adminText('k00ii'))
  }
}

async function confirmMigrate() {
  if (storageForm.provider !== 's3') return
  migrateConfirm.value = false
  migrating.value = true
  try {
    await createStorageMigrateTask(clearAfterMigrate.value)
    adminToast.success(adminText('k00ie'))
    await loadMigrateTasks()
  } catch (err) {
    adminToast.error(err, adminText('k00ii'))
  } finally {
    migrating.value = false
  }
}

function migrateTaskStatus(status: number) {
  const statusKey = ['k00i0', 'k00i1', 'k00i2', 'k00i3', 'k00i4'][status] || 'k00i3'
  return adminText(statusKey)
}

function addReservedUsername() {
  const name = newReservedUsername.value.trim()
  if (!name || securityForm.reservedUsernames.includes(name)) return
  securityForm.reservedUsernames.push(name)
  newReservedUsername.value = ''
}

function removeReservedUsername(name: string) {
  securityForm.reservedUsernames = securityForm.reservedUsernames.filter(item => item !== name)
}

function addBannedUsername() {
  const name = newBannedUsername.value.trim()
  if (!name || securityForm.bannedUsernames.includes(name)) return
  securityForm.bannedUsernames.push(name)
  newBannedUsername.value = ''
}

function removeBannedUsername(name: string) {
  securityForm.bannedUsernames = securityForm.bannedUsernames.filter(item => item !== name)
}

function addSensitiveWord() {
  const word = newSensitiveWord.value.trim()
  if (!word || securityForm.sensitiveWords.includes(word)) return
  securityForm.sensitiveWords.push(word)
  newSensitiveWord.value = ''
}

function removeSensitiveWord(word: string) {
  securityForm.sensitiveWords = securityForm.sensitiveWords.filter(item => item !== word)
}

function addAllowedDomain() {
  const domain = newAllowedDomain.value.trim().toLowerCase()
  if (!domain || allowedDomains.value.includes(domain)) return
  const next = [...allowedDomains.value, domain]
  securityForm.allowedDomains = next
  newAllowedDomain.value = ''
}

function removeAllowedDomain(domain: string) {
  const next = allowedDomains.value.filter(item => item !== domain)
  securityForm.allowedDomains = next
}

function addExtension() {
  const ext = newExtension.value.trim().toLowerCase()
  if (!ext) return
  if (!ext.startsWith('.')) {
    adminToast.warning(adminText('k000j'))
    return
  }
  if (!postingForm.uploadControl.authorizedExtensions.includes(ext)) {
    postingForm.uploadControl.authorizedExtensions.push(ext)
  }
  newExtension.value = ''
}

function removeExtension(ext: string) {
  postingForm.uploadControl.authorizedExtensions = postingForm.uploadControl.authorizedExtensions.filter(item => item !== ext)
}

function addHttpEndpoint() {
  httpNotifyForm.endpoints.push({
    id: crypto.randomUUID(),
    name: '',
    enabled: true,
    url: '',
    secret: '',
    events: ['topic.published'],
    timeoutSeconds: 2,
    failureCount: 0,
    lastError: '',
    abnormalTerminated: false,
  })
}

function removeHttpEndpoint(index: number) {
  httpNotifyForm.endpoints.splice(index, 1)
}

function toggleEndpointEvent(endpoint: HttpNotifyEndpoint, eventName: string, checked: boolean) {
  if (checked && !endpoint.events.includes(eventName)) {
    endpoint.events.push(eventName)
  }
  if (!checked) {
    endpoint.events = endpoint.events.filter(item => item !== eventName)
  }
}

function onEndpointEventChange(endpoint: HttpNotifyEndpoint, eventName: string, event: Event) {
  toggleEndpointEvent(endpoint, eventName, (event.target as HTMLInputElement).checked)
}

// ---- 一系统同步（issue #248）----

async function loadOnesystem() {
  try {
    const settings = await getOnesystemSettings()
    onesystemForm.cookieConfigured = settings.cookieConfigured
  } catch (err) {
    error.value = err instanceof Error ? err.message : adminText('k000d')
  }
}

async function refreshSyncStatus() {
  syncStatusLoading.value = true
  try {
    syncStatusItems.value = await getPkSyncStatus()
  } catch (err) {
    adminToast.error(err, adminText('k00s3'))
  } finally {
    syncStatusLoading.value = false
  }
}

async function saveCookie() {
  savingCookie.value = true
  try {
    await saveOnesystemSettings(onesystemForm.cookie.trim())
    onesystemForm.cookieConfigured = onesystemForm.cookie.trim() !== ''
    onesystemForm.cookie = ''
    adminToast.success(adminText('k00sv'))
  } catch (err) {
    adminToast.error(err, adminText('k00s1'))
  } finally {
    savingCookie.value = false
  }
}

async function clearCookie() {
  onesystemForm.cookie = ''
  savingCookie.value = true
  try {
    await saveOnesystemSettings('')
    onesystemForm.cookieConfigured = false
    adminToast.success(adminText('k00sw'))
  } catch (err) {
    adminToast.error(err, adminText('k00s1'))
  } finally {
    savingCookie.value = false
  }
}

async function startSync() {
  const term = syncForm.term.trim()
  if (!term) {
    adminToast.warning(adminText('k00sy'))
    return
  }
  syncingPk.value = true
  try {
    const depth = Math.min(Math.max(syncForm.depth || 1, 1), 8)
    const result = await syncPkCalendar(term, depth)
    adminToast.success(adminText('k00sq', { term: result.term || term }))
    startSyncPolling(result.calendarId)
  } catch (err) {
    adminToast.error(err, adminText('k00s2'))
  } finally {
    syncingPk.value = false
  }
}

/** 同步为后台异步：每 3s 只轮询本次取得租约的目标学期。 */
function startSyncPolling(calendarId: number) {
  stopSyncPolling()
  const poll = () => {
    void refreshSyncStatus().then(() => {
      const item = syncStatusItems.value.find((status) => status.calendarId === calendarId)
      if (!item || item.status === 'running') return
      stopSyncPolling()
      if (item.status === 'failed') {
        adminToast.error(adminText('k00sz'))
      } else {
        adminToast.success(adminText('k00sr'))
      }
    })
  }
  poll()
  syncPollTimer = setInterval(poll, 3000)
}

function stopSyncPolling() {
  if (syncPollTimer !== null) {
    clearInterval(syncPollTimer)
    syncPollTimer = null
  }
}

function statusLabel(status: string) {
  switch (status) {
    case 'running': return adminText('k00ss')
    case 'completed': return adminText('k00st')
    case 'failed': return adminText('k00su')
    default: return adminText('k00sx')
  }
}

function statusVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'running': return 'secondary'
    case 'completed': return 'default'
    case 'failed': return 'destructive'
    default: return 'outline'
  }
}

function formatTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

watch(() => props.kind, () => {
  stopSyncPolling()
  void load()
})

onMounted(load)

onUnmounted(stopSyncPolling)
</script>

<template>
  <BasicPage :title="pageMeta.title" :description="pageMeta.description" sticky>
    <template #actions>
      <Button v-if="kind !== 'onesystem'" type="button" :disabled="saving" @click="save">
        <Loader2 v-if="saving" class="size-4 animate-spin" />
        <Save v-else class="size-4" />
        {{ adminText('k004f') }}
      </Button>
    </template>

      <div v-if="loading" class="flex h-[400px] items-center justify-center">
        <Loader2 class="size-8 animate-spin text-primary" />
      </div>
      <div v-else-if="error" class="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">{{ error }}</div>

      <form v-else-if="kind === 'site-info'" class="space-y-10" @submit.prevent="save">
        <div class="grid gap-10 md:grid-cols-2">
          <section class="space-y-6">
            <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium"><Globe class="size-5 text-muted-foreground" />{{ adminText('k007z') }}</div>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k0080') }}<Input v-model="siteForm.siteName" placeholder="YourTJHub" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k0081') }}<Input v-model="siteForm.siteUrl" placeholder="https://example.com" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k0082') }}<Input v-model="siteForm.siteEmail" placeholder="contact@example.com" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k0083') }}
              <div class="flex items-start gap-4">
                <div class="grid size-20 shrink-0 place-items-center overflow-hidden rounded-lg border bg-muted">
                  <img v-if="siteForm.siteLogo" :src="siteForm.siteLogo" class="size-full object-cover" alt="Logo" />
                  <Upload v-else class="size-8 text-muted-foreground/50" />
                </div>
                <div class="grid flex-1 gap-2">
                  <Input v-model="siteForm.siteLogo" placeholder="Logo URL" />
                  <label class="inline-flex h-9 w-fit cursor-pointer items-center gap-2 rounded-md border bg-background px-3 text-sm font-medium shadow-xs hover:bg-accent">
                    {{ adminText('k0084') }}
                    <input class="hidden" type="file" accept="image/*" @change="uploadImage('siteLogo', $event)" />
                  </label>
                </div>
              </div>
            </label>
          </section>
          <section class="space-y-6">
            <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium"><FileText class="size-5 text-muted-foreground" />{{ adminText('k008b') }}</div>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k008c') }}<Textarea v-model="siteForm.siteDescription" class="min-h-24" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k008d') }}<Input v-model="siteForm.siteKeywords" placeholder="forum, community" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k008e') }}<Textarea v-model="siteForm.externalLinks" class="min-h-28 font-mono text-xs" /></label>
          </section>
        </div>
      </form>

      <form v-else-if="kind === 'mail'" class="grid gap-10 lg:grid-cols-[minmax(0,2fr)_minmax(260px,1fr)]" @submit.prevent="save">
        <section class="space-y-6">
          <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium">{{ adminText('k008j') }}</div>
          <div class="flex items-center justify-between rounded-lg border bg-muted/20 p-4">
            <div><div class="font-medium">{{ adminText('k008k') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k008l') }}</p></div>
            <Switch v-model="mailForm.enableMail" />
          </div>
          <div class="grid gap-6 md:grid-cols-2">
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k008m') }}<Input v-model="mailForm.smtpHost" :disabled="!mailForm.enableMail" placeholder="smtp.example.com" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k008n') }}<Input v-model.number="mailForm.smtpPort" :disabled="!mailForm.enableMail" type="number" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k008o') }}<Input v-model="mailForm.smtpUsername" :disabled="!mailForm.enableMail" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k008p') }}<Input v-model="mailForm.smtpPassword" :disabled="!mailForm.enableMail" type="password" /></label>
          </div>
          <div class="flex items-center justify-between rounded-lg border bg-muted/20 p-4">
            <div><div class="flex items-center gap-2 font-medium"><Shield class="size-4" />{{ adminText('k008q') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k008r') }}</p></div>
            <Switch v-model="mailForm.useSSL" :disabled="!mailForm.enableMail" />
          </div>
        </section>
        <aside class="space-y-8">
          <section class="space-y-4">
            <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium"><Send class="size-5 text-muted-foreground" />{{ adminText('k008s') }}</div>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k008t') }}<Input v-model="mailForm.fromName" :disabled="!mailForm.enableMail" placeholder="YourTJHub" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k008u') }}<Input v-model="mailForm.fromEmail" :disabled="!mailForm.enableMail" placeholder="noreply@example.com" /></label>
          </section>
          <section class="space-y-4">
            <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium">{{ adminText('k008v') }}</div>
            <div class="space-y-3 rounded-lg border p-4">
              <p class="text-sm text-muted-foreground">{{ adminText('k008w') }}</p>
              <Input v-model="testEmail" :disabled="!mailForm.enableMail" :placeholder="adminText('k004k')" />
              <Button class="w-full" type="button" variant="secondary" :disabled="testing || !mailForm.enableMail" @click="sendTestMail">
                <Loader2 v-if="testing" class="size-4 animate-spin" />
                <Send v-else class="size-4" />
                {{ adminText('k008x') }}
              </Button>
            </div>
          </section>
        </aside>
      </form>

      <form v-else-if="kind === 'security'" class="max-w-2xl space-y-8" @submit.prevent="save">
        <div class="flex items-center justify-between">
          <div><div class="text-base font-medium">{{ adminText('k008y') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k008z') }}</p></div>
          <Switch v-model="securityForm.enableSignup" />
        </div>
        <div class="flex items-center justify-between">
          <div><div class="flex items-center gap-2 text-base font-medium"><MailCheck class="size-4" />{{ adminText('k0090') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k0091') }}</p></div>
          <Switch v-model="securityForm.enableEmailVerification" />
        </div>
        <div class="flex items-center justify-between">
          <div><div class="flex items-center gap-2 text-base font-medium"><Shield class="size-4" />{{ adminText('k00it') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k00iu') }}</p></div>
          <Switch v-model="securityForm.captchaRequired" />
        </div>
        <div class="space-y-4">
          <div><div class="text-base font-medium">{{ adminText('k0092') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k0093') }}</p></div>
          <div class="flex gap-2">
            <Input v-model="newAllowedDomain" class="max-w-sm" :placeholder="adminText('k004l')" @keydown.enter.prevent="addAllowedDomain" />
            <Button type="button" variant="secondary" @click="addAllowedDomain"><Plus class="size-4" />{{ adminText('k0094') }}</Button>
          </div>
          <div class="flex flex-wrap gap-2">
            <span v-if="allowedDomains.length === 0" class="text-sm italic text-muted-foreground">{{ adminText('k0095') }}</span>
            <Badge v-for="domain in allowedDomains" :key="domain" variant="secondary" class="gap-2 px-3 py-1.5 text-sm font-normal">
              {{ domain }}
              <AdminActionButton compact tone="danger" class="-mr-1 size-5" :title="adminText('k005i')" @click="removeAllowedDomain(domain)">
                <Trash2 class="size-3.5" />
              </AdminActionButton>
            </Badge>
          </div>
        </div>
        <div class="space-y-4">
          <div><div class="text-base font-medium">{{ adminText('k00hr') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k00hs') }}</p></div>
          <div class="flex gap-2">
            <Input v-model="newReservedUsername" class="max-w-sm" :placeholder="adminText('k00eu')" @keydown.enter.prevent="addReservedUsername" />
            <Button type="button" variant="secondary" @click="addReservedUsername"><Plus class="size-4" />{{ adminText('k0094') }}</Button>
          </div>
          <div class="flex flex-wrap gap-2">
            <span v-if="securityForm.reservedUsernames.length === 0" class="text-sm italic text-muted-foreground">{{ adminText('k00et') }}</span>
            <Badge v-for="name in securityForm.reservedUsernames" :key="name" variant="secondary" class="gap-2 px-3 py-1.5 text-sm font-normal">
              {{ name }}
              <AdminActionButton compact tone="danger" class="-mr-1 size-5" :title="adminText('k005i')" @click="removeReservedUsername(name)">
                <Trash2 class="size-3.5" />
              </AdminActionButton>
            </Badge>
          </div>
        </div>
        <div class="space-y-4">
          <div><div class="text-base font-medium">{{ adminText('k00ht') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k00hu') }}</p></div>
          <div class="flex gap-2">
            <Input v-model="newBannedUsername" class="max-w-sm" :placeholder="adminText('k00eu')" @keydown.enter.prevent="addBannedUsername" />
            <Button type="button" variant="secondary" @click="addBannedUsername"><Plus class="size-4" />{{ adminText('k0094') }}</Button>
          </div>
          <div class="flex flex-wrap gap-2">
            <span v-if="securityForm.bannedUsernames.length === 0" class="text-sm italic text-muted-foreground">{{ adminText('k00et') }}</span>
            <Badge v-for="name in securityForm.bannedUsernames" :key="name" variant="secondary" class="gap-2 px-3 py-1.5 text-sm font-normal">
              {{ name }}
              <AdminActionButton compact tone="danger" class="-mr-1 size-5" :title="adminText('k005i')" @click="removeBannedUsername(name)">
                <Trash2 class="size-3.5" />
              </AdminActionButton>
            </Badge>
          </div>
        </div>
        <div class="space-y-4">
          <div><div class="text-base font-medium">{{ adminText('k00hv') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k00hw') }}</p></div>
          <div class="flex gap-2">
            <Input v-model="newSensitiveWord" class="max-w-sm" @keydown.enter.prevent="addSensitiveWord" />
            <Button type="button" variant="secondary" @click="addSensitiveWord"><Plus class="size-4" />{{ adminText('k0094') }}</Button>
          </div>
          <div class="flex flex-wrap gap-2">
            <span v-if="securityForm.sensitiveWords.length === 0" class="text-sm italic text-muted-foreground">{{ adminText('k00et') }}</span>
            <Badge v-for="word in securityForm.sensitiveWords" :key="word" variant="secondary" class="gap-2 px-3 py-1.5 text-sm font-normal">
              {{ word }}
              <AdminActionButton compact tone="danger" class="-mr-1 size-5" :title="adminText('k005i')" @click="removeSensitiveWord(word)">
                <Trash2 class="size-3.5" />
              </AdminActionButton>
            </Badge>
          </div>
        </div>
        <div>
          <div class="text-base font-medium">{{ adminText('k00hx') }}</div>
          <div class="mt-3 inline-flex rounded-lg border bg-muted/20 p-1">
            <button
              type="button"
              class="inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              :class="securityForm.sensitiveAction === 'block' ? 'bg-background shadow-xs' : 'text-muted-foreground hover:text-foreground'"
              @click="securityForm.sensitiveAction = 'block'"
            >
              <Shield class="size-4" />
              {{ adminText('k00hy') }}
            </button>
            <button
              type="button"
              class="inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              :class="securityForm.sensitiveAction === 'review' ? 'bg-background shadow-xs' : 'text-muted-foreground hover:text-foreground'"
              @click="securityForm.sensitiveAction = 'review'"
            >
              <CheckCircle2 class="size-4" />
              {{ adminText('k00hz') }}
            </button>
          </div>
        </div>
      </form>

      <form v-else-if="kind === 'rate-limit'" class="max-w-4xl space-y-8" @submit.prevent="save">
        <div class="flex items-center justify-between rounded-lg border bg-muted/10 p-4">
          <div><div class="flex items-center gap-2 text-base font-medium"><Shield class="size-4" />{{ adminText('k00ik') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k00ij') }}</p></div>
          <Switch v-model="rateLimitForm.enabled" />
        </div>
        <div class="flex items-center justify-between rounded-lg border bg-muted/10 p-4">
          <div><div class="text-base font-medium">{{ adminText('k00il') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k00ij') }}</p></div>
          <Switch v-model="rateLimitForm.skipAdmin" :disabled="!rateLimitForm.enabled" />
        </div>
        <section class="space-y-3">
          <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium"><FileText class="size-5 text-muted-foreground" />{{ adminText('k00ip') }}</div>
          <div class="grid gap-3">
            <div v-for="(rule, index) in rateLimitForm.actions" :key="rule.action" class="grid grid-cols-[minmax(120px,1fr)_110px_110px_110px] items-center gap-3 rounded-lg border p-3">
              <span class="truncate font-mono text-sm">{{ rule.action }}</span>
              <label class="grid gap-1 text-xs text-muted-foreground">{{ adminText('k00iq') }}<Input v-model.number="rateLimitForm.actions[index].windowSeconds" :disabled="!rateLimitForm.enabled" type="number" min="1" /></label>
              <label class="grid gap-1 text-xs text-muted-foreground">{{ adminText('k00ir') }}<Input v-model.number="rateLimitForm.actions[index].limitPerIp" :disabled="!rateLimitForm.enabled" type="number" min="0" /></label>
              <label class="grid gap-1 text-xs text-muted-foreground">{{ adminText('k00is') }}<Input v-model.number="rateLimitForm.actions[index].limitPerUser" :disabled="!rateLimitForm.enabled" type="number" min="0" /></label>
            </div>
          </div>
        </section>
        <section class="grid gap-6 sm:grid-cols-3">
          <label class="grid gap-2 text-sm font-medium">{{ adminText('k00im') }}<Input v-model.number="rateLimitForm.newUserCaptchaAfterPosts" :disabled="!rateLimitForm.enabled" type="number" min="0" /></label>
          <label class="grid gap-2 text-sm font-medium">{{ adminText('k00in') }}<Input v-model.number="rateLimitForm.newUserCaptchaDays" :disabled="!rateLimitForm.enabled" type="number" min="0" /></label>
          <label class="grid gap-2 text-sm font-medium">{{ adminText('k00io') }}<Input v-model.number="rateLimitForm.minSubmitSeconds" :disabled="!rateLimitForm.enabled" type="number" min="0" /></label>
        </section>
      </form>

      <form v-else-if="kind === 'mcp'" class="max-w-4xl space-y-8" @submit.prevent="save">
        <div class="flex items-center justify-between rounded-lg border bg-muted/10 p-4">
          <div><div class="flex items-center gap-2 text-base font-medium"><Bot class="size-4" />{{ adminText('k00ml') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k00mm') }}</p></div>
          <Switch v-model="mcpForm.enabled" />
        </div>
        <div class="flex items-center justify-between rounded-lg border bg-muted/10 p-4">
          <div><div class="text-base font-medium">{{ adminText('k00mn') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k00mo') }}</p></div>
          <Switch v-model="mcpForm.writes" :disabled="!mcpForm.enabled" />
        </div>
      </form>

      <form v-else-if="kind === 'ai-summary'" class="max-w-2xl space-y-8" @submit.prevent="save">
        <div class="flex items-center justify-between rounded-lg border bg-muted/10 p-4">
          <div>
            <div class="flex items-center gap-2 text-base font-medium"><Sparkles class="size-4" />{{ adminText('k00p4') }}</div>
            <p class="mt-1 text-sm text-muted-foreground">{{ adminText('k00p5') }}</p>
          </div>
          <Switch v-model="aiSummaryForm.enabled" />
        </div>
        <label class="grid gap-2 text-sm font-medium">
          {{ adminText('k00p6') }}
          <Input v-model.number="aiSummaryForm.globalPerMinute" type="number" min="0" :disabled="!aiSummaryForm.enabled" />
          <span class="text-xs font-normal text-muted-foreground">{{ adminText('k00p7') }}</span>
        </label>
      </form>

      <form v-else-if="kind === 'posting'" class="grid gap-12 lg:grid-cols-2" @submit.prevent="save">
        <section class="space-y-6">
          <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium"><FileText class="size-5 text-muted-foreground" />{{ adminText('k0096') }}</div>
          <div class="grid gap-6 sm:grid-cols-2">
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k0097') }}<Input v-model.number="postingForm.textControl.minTitleLength" type="number" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k0098') }}<Input v-model.number="postingForm.textControl.maxTitleLength" type="number" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k0099') }}<Input v-model.number="postingForm.textControl.minPostLength" type="number" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k009a') }}<Input v-model.number="postingForm.textControl.maxPostLength" type="number" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k009b') }}<Input v-model.number="postingForm.textControl.newUserPostCooldownMinutes" type="number" /></label>
          </div>
        </section>
        <section class="space-y-6">
          <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium"><Upload class="size-5 text-muted-foreground" />{{ adminText('k009c') }}</div>
          <div class="flex items-center justify-between rounded-lg border bg-muted/10 p-4">
            <div><div class="text-base font-medium">{{ adminText('k009d') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k009e') }}</p></div>
            <Switch v-model="postingForm.uploadControl.allowAttachments" />
          </div>
          <div class="grid gap-6 sm:grid-cols-2">
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k009f') }}<Input v-model.number="postingForm.uploadControl.maxDailyUploadsPerUser" :disabled="!postingForm.uploadControl.allowAttachments" type="number" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k009g') }}<Input v-model.number="postingForm.uploadControl.maxAttachmentSizeKb" :disabled="!postingForm.uploadControl.allowAttachments" type="number" /></label>
            <label class="grid gap-2 text-sm font-medium">{{ adminText('k009h') }}<Input v-model.number="postingForm.uploadControl.newUserUploadCooldownMinutes" :disabled="!postingForm.uploadControl.allowAttachments" type="number" /></label>
          </div>
          <div class="space-y-3">
            <div class="text-sm font-medium">{{ adminText('k009i') }}</div>
            <div class="flex gap-2">
              <Input v-model="newExtension" :placeholder="adminText('k004m')" @keydown.enter.prevent="addExtension" />
              <Button type="button" variant="secondary" @click="addExtension"><Plus class="size-4" />{{ adminText('k0094') }}</Button>
            </div>
            <div class="flex flex-wrap gap-2">
              <Badge v-for="ext in postingForm.uploadControl.authorizedExtensions" :key="ext" variant="secondary" class="gap-2 px-3 py-1.5 text-sm font-normal">
                {{ ext }}
                <AdminActionButton compact tone="danger" class="-mr-1 size-5" :title="adminText('k005i')" @click="removeExtension(ext)">
                  <Trash2 class="size-3.5" />
                </AdminActionButton>
              </Badge>
            </div>
          </div>
        </section>
        <section class="space-y-5 lg:col-span-2">
          <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium"><Bot class="size-5 text-muted-foreground" />{{ adminText('k00iv') }}</div>
          <p class="text-sm text-muted-foreground">{{ adminText('k00iw') }}</p>
          <div class="grid gap-4 lg:grid-cols-3">
            <div class="flex items-center justify-between gap-4 rounded-lg border bg-muted/10 p-4">
              <div><div class="font-medium">{{ adminText('k00ix') }}</div><p class="mt-1 text-sm text-muted-foreground">{{ adminText('k00iy') }}</p></div>
              <Switch v-model="postingForm.llms.enabled" />
            </div>
            <div class="flex items-center justify-between gap-4 rounded-lg border bg-muted/10 p-4">
              <div><div class="font-medium">{{ adminText('k00iz') }}</div><p class="mt-1 text-sm text-muted-foreground">{{ adminText('k00j3') }}</p></div>
              <Switch v-model="postingForm.llms.fullText" :disabled="!postingForm.llms.enabled" />
            </div>
            <div class="flex items-center justify-between gap-4 rounded-lg border bg-muted/10 p-4">
              <div><div class="font-medium">{{ adminText('k00j4') }}</div><p class="mt-1 text-sm text-muted-foreground">{{ adminText('k00j5') }}</p></div>
              <Switch v-model="postingForm.llms.files" :disabled="!postingForm.llms.enabled" />
            </div>
          </div>
        </section>
      </form>

      <div v-else-if="kind === 'http-notify'" class="max-w-5xl">
        <Tabs :key="String(locale)" v-model="httpNotifyTab" class="gap-5">
          <TabsList class="w-fit">
            <TabsTrigger value="config">{{ adminText('k00d0') }}</TabsTrigger>
            <TabsTrigger value="guide">{{ adminText('k00d1') }}</TabsTrigger>
          </TabsList>

          <TabsContent value="config">
            <form class="space-y-5" @submit.prevent="save">
              <div class="flex items-center justify-between border-b pb-4">
                <div>
                  <div class="flex items-center gap-2 text-base font-medium"><Webhook class="size-4" />{{ adminText('k00cq') }}</div>
                  <p class="text-sm text-muted-foreground">{{ adminText('k00cr') }}</p>
                </div>
                <Switch v-model="httpNotifyForm.enabled" />
              </div>

              <section class="space-y-3">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <div class="text-lg font-medium">{{ adminText('k00cs') }}</div>
                    <p class="text-sm text-muted-foreground">{{ adminText('k00ct') }}</p>
                  </div>
                  <Button type="button" variant="secondary" @click="addHttpEndpoint"><Plus class="size-4" />{{ adminText('k00cu') }}</Button>
                </div>

                <div v-if="httpNotifyForm.endpoints.length === 0" class="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
                  {{ adminText('k00cv') }}
                </div>

                <div v-for="(endpoint, index) in httpNotifyForm.endpoints" :key="endpoint.id || index" class="space-y-3 rounded-lg border bg-background p-3">
                  <div class="flex flex-wrap items-center justify-between gap-3">
                    <div class="flex min-w-0 items-center gap-3">
                      <Switch v-model="endpoint.enabled" :disabled="!httpNotifyForm.enabled" />
                      <div class="min-w-0">
                        <div class="flex flex-wrap items-center gap-2">
                          <span class="font-medium">{{ endpoint.name || adminText('k00cw') }}</span>
                          <Badge v-if="endpoint.abnormalTerminated" variant="destructive" class="px-2 py-0 text-xs">{{ adminText('k00cz') }}</Badge>
                        </div>
                        <div v-if="endpoint.lastError" class="mt-0.5 truncate text-xs text-muted-foreground">{{ endpoint.lastError }}</div>
                      </div>
                    </div>
                    <AdminActionButton compact tone="danger" :title="adminText('k005i')" @click="removeHttpEndpoint(index)">
                      <Trash2 class="size-4" />
                    </AdminActionButton>
                  </div>

                  <div class="grid gap-3 md:grid-cols-[minmax(140px,220px)_minmax(0,1fr)_120px]">
                    <label class="grid gap-2 text-sm font-medium">
                      {{ adminText('k0079') }}
                      <Input v-model="endpoint.name" :disabled="!httpNotifyForm.enabled" placeholder="Webhook" />
                    </label>
                    <label class="grid gap-2 text-sm font-medium">
                      URL
                      <Input v-model="endpoint.url" :disabled="!httpNotifyForm.enabled" placeholder="http://example.com/webhook" />
                    </label>
                    <label class="grid gap-2 text-sm font-medium">
                      {{ adminText('k00cx') }}
                      <Input v-model.number="endpoint.timeoutSeconds" :disabled="!httpNotifyForm.enabled" type="number" min="1" max="15" />
                    </label>
                  </div>

                  <label class="grid gap-2 text-sm font-medium">
                    Secret
                    <Input v-model="endpoint.secret" :disabled="!httpNotifyForm.enabled" type="password" autocomplete="new-password" />
                  </label>

                  <div class="space-y-2">
                    <div class="text-sm font-medium">{{ adminText('k00cy') }}</div>
                    <div class="flex flex-wrap gap-2">
                      <label v-for="item in httpNotifyEvents" :key="item.value" class="inline-flex items-center gap-2 rounded-md border px-2.5 py-1.5 text-sm">
                        <input
                          type="checkbox"
                          :disabled="!httpNotifyForm.enabled"
                          :checked="endpoint.events.includes(item.value)"
                          @change="onEndpointEventChange(endpoint, item.value, $event)"
                        />
                        {{ item.label }}
                      </label>
                    </div>
                  </div>
                </div>
              </section>
            </form>
          </TabsContent>

          <TabsContent value="guide">
            <div class="gf-admin-markdown rounded-lg border bg-background p-5" v-html="httpNotifyGuideHtml" />
          </TabsContent>
        </Tabs>
      </div>

      <form v-else-if="kind === 'storage'" class="max-w-3xl space-y-8" @submit.prevent="save">
        <div class="grid gap-6 md:grid-cols-2">
          <label class="grid gap-2 text-sm font-medium">{{ adminText('k00fp') }}
            <select v-model="storageForm.provider" class="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring">
              <option value="local">{{ adminText('k00fq') }}</option>
              <option value="s3">{{ adminText('k00fr') }}</option>
            </select>
          </label>
          <label class="grid gap-2 text-sm font-medium">{{ adminText('k00fv') }}
            <select v-model="storageForm.bucketLookup" class="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring" :disabled="storageForm.provider !== 's3'">
              <option value="auto">{{ adminText('k00fw') }}</option>
              <option value="dns">{{ adminText('k00g0') }}</option>
              <option value="path">{{ adminText('k00g1') }}</option>
            </select>
          </label>
        </div>
        <div class="grid gap-6 md:grid-cols-2">
          <label class="grid gap-2 text-sm font-medium">{{ adminText('k00fs') }}<Input v-model="storageForm.endpoint" :disabled="storageForm.provider !== 's3'" placeholder="https://s3.example.com" /></label>
          <label class="grid gap-2 text-sm font-medium">{{ adminText('k00ft') }}<Input v-model="storageForm.bucket" :disabled="storageForm.provider !== 's3'" /></label>
          <label class="grid gap-2 text-sm font-medium">{{ adminText('k00fu') }}<Input v-model="storageForm.region" :disabled="storageForm.provider !== 's3'" /></label>
          <label class="grid gap-2 text-sm font-medium">{{ adminText('k00g5') }}<Input v-model="storageForm.publicUrlPrefix" :disabled="storageForm.provider !== 's3'" placeholder="https://cdn.example.com" /></label>
        </div>
        <div class="grid gap-6 md:grid-cols-2">
          <label class="grid gap-2 text-sm font-medium">{{ adminText('k00g3') }}<Input v-model="storageForm.accessKey" :disabled="storageForm.provider !== 's3'" autocomplete="off" /></label>
          <label class="grid gap-2 text-sm font-medium">{{ adminText('k00g4') }}<Input v-model="storageForm.secretKey" :disabled="storageForm.provider !== 's3'" type="password" autocomplete="new-password" /></label>
        </div>
        <div class="flex items-center justify-between rounded-lg border bg-muted/20 p-4">
          <div><div class="flex items-center gap-2 font-medium"><HardDrive class="size-4" />{{ adminText('k00g2') }}</div></div>
          <Switch v-model="storageForm.secure" :disabled="storageForm.provider !== 's3'" />
        </div>

        <div class="flex flex-wrap items-center gap-3 border-t pt-6">
          <Button type="button" variant="secondary" :disabled="testing" @click="testStorage">
            <Loader2 v-if="testing" class="size-4 animate-spin" />
            <CheckCircle2 v-else class="size-4" />
            {{ adminText('k00g6') }}
          </Button>
          <Button type="button" variant="outline" :disabled="storageForm.provider !== 's3'" @click="migrateConfirm = true">
            <HardDrive class="size-4" />
            {{ adminText('k00g9') }}
          </Button>
          <label class="inline-flex items-center gap-2 text-sm text-muted-foreground">
            <input v-model="clearAfterMigrate" type="checkbox" class="size-4 rounded border" :disabled="storageForm.provider !== 's3'" />
            {{ adminText('k00ga') }}
          </label>
        </div>

        <section class="space-y-3">
          <div class="flex items-center gap-2 border-b pb-2 text-lg font-medium"><HardDrive class="size-5 text-muted-foreground" />{{ adminText('k00if') }}</div>
          <div v-if="migrateTaskRows.length === 0" class="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">{{ adminText('k00hq') }}</div>
          <div v-for="task in migrateTaskRows" :key="task.id" class="space-y-2 rounded-lg border bg-background p-3 text-sm">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <span class="font-mono text-xs text-muted-foreground">#{{ task.id }}</span>
                <Badge :variant="task.status === 2 ? 'default' : task.status === 3 ? 'destructive' : 'secondary'" class="px-2 py-0 text-xs">{{ migrateTaskStatus(task.status) }}</Badge>
              </div>
              <span class="text-xs text-muted-foreground">{{ task.createdAt || '-' }}</span>
            </div>
            <div v-if="task.payload.total != null || (task.payload.failed || 0) > 0" class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
              <span v-if="task.payload.total != null" class="text-muted-foreground">{{ adminText('k00ms', { processed: task.payload.processed || 0, total: task.payload.total }) }}</span>
              <span v-if="(task.payload.failed || 0) > 0" class="font-medium text-destructive">{{ adminText('k00mt', { failed: task.payload.failed }) }}</span>
            </div>
            <div v-if="task.lastError" class="truncate text-xs text-destructive">{{ task.lastError }}</div>
          </div>
        </section>
      </form>

      <form v-else-if="kind === 'terms'" class="max-w-3xl space-y-6" @submit.prevent="save">
        <div class="flex items-center justify-between">
          <div><div class="flex items-center gap-2 text-base font-medium"><ScrollText class="size-4" />{{ adminText('k00gr') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k00gq') }}</p></div>
          <Switch v-model="termsForm.enabled" />
        </div>
        <label class="grid gap-2 text-sm font-medium">
          {{ adminText('k00gs') }}
          <Textarea v-model="termsForm.content" class="min-h-64 resize-y font-mono text-sm" :placeholder="adminText('k004n')" />
        </label>
      </form>


      <div v-else-if="kind === 'onesystem'" class="max-w-3xl space-y-8">
        <!-- Cookie 凭证配置 -->
        <form class="space-y-4 rounded-lg border border-border bg-card p-5" @submit.prevent="saveCookie">
          <div class="flex items-center gap-2 text-base font-medium"><KeyRound class="size-4 text-muted-foreground" />{{ adminText('k00s6') }}</div>
          <p class="text-sm text-muted-foreground">{{ adminText('k00s7') }}</p>
          <div class="flex items-center gap-2">
            <Badge :variant="onesystemForm.cookieConfigured ? 'default' : 'outline'">
              {{ onesystemForm.cookieConfigured ? adminText('k00s8') : adminText('k00s9') }}
            </Badge>
          </div>
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00sa') }}
            <Textarea v-model="onesystemForm.cookie" :placeholder="adminText('k00sb')" rows="2" autocomplete="off" />
            <span class="text-xs font-normal text-muted-foreground">{{ adminText('k00sc') }}</span>
          </label>
          <div class="flex gap-2">
            <Button type="submit" :disabled="savingCookie">
              <Loader2 v-if="savingCookie" class="size-4 animate-spin" />
              <Save v-else class="size-4" />
              {{ adminText('k00sd') }}
            </Button>
            <Button type="button" variant="outline" :disabled="savingCookie || !onesystemForm.cookieConfigured" @click="clearCookie">
              {{ adminText('k00se') }}
            </Button>
          </div>
        </form>

        <!-- 排课数据同步（issue #248）-->
        <div class="space-y-4 rounded-lg border border-border bg-card p-5">
          <div class="flex items-center gap-2 text-base font-medium"><RefreshCw class="size-4 text-muted-foreground" />{{ adminText('k00sf') }}</div>
          <p class="text-sm text-muted-foreground">{{ adminText('k00sg') }}</p>
          <div class="flex flex-wrap items-end gap-3">
            <label class="grid min-w-0 flex-1 gap-2 text-sm font-medium">
              {{ adminText('k00sh') }}
              <Input v-model="syncForm.term" :placeholder="adminText('k00si')" />
            </label>
            <label class="grid w-28 gap-2 text-sm font-medium">
              {{ adminText('k00sj') }}
              <Input v-model.number="syncForm.depth" type="number" min="1" max="8" />
            </label>
            <Button type="button" :disabled="syncingPk" @click="startSync">
              <Loader2 v-if="syncingPk" class="size-4 animate-spin" />
              <RefreshCw v-else class="size-4" />
              {{ adminText('k00sk') }}
            </Button>
          </div>

          <div class="border-t pt-4">
            <div class="mb-2 flex items-center justify-between">
              <span class="text-sm font-medium">{{ adminText('k00sl') }}</span>
              <Button type="button" variant="ghost" size="sm" @click="refreshSyncStatus">
                <RefreshCw class="size-3.5" />
                {{ adminText('k00sm') }}
              </Button>
            </div>
            <div v-if="syncStatusLoading" class="flex items-center justify-center gap-2 py-6 text-sm text-muted-foreground">
              <Loader2 class="size-4 animate-spin" />
              {{ adminText('k00sn') }}
            </div>
            <p v-else-if="syncStatusItems.length === 0" class="py-6 text-center text-sm text-muted-foreground">{{ adminText('k00so') }}</p>
            <ul v-else class="divide-y divide-border">
              <li v-for="item in syncStatusItems" :key="item.calendarId" class="flex items-start gap-3 py-2.5">
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="font-medium">{{ item.calendarName || String(item.calendarId) }}</span>
                    <Badge :variant="statusVariant(item.status)">{{ statusLabel(item.status) }}</Badge>
                    <span v-if="item.status === 'running'" class="text-xs text-muted-foreground">{{ item.lastCommittedPage }}/{{ item.totalPages }}</span>
                  </div>
                  <p v-if="item.errorMsg" class="mt-1 text-xs text-destructive">{{ item.errorMsg }}</p>
                  <p class="mt-0.5 text-xs text-muted-foreground">
                    {{ adminText('k00sp', { rows: item.rowsWritten }) }}
                    <span v-if="item.finishedAt"> · {{ formatTime(item.finishedAt) }}</span>
                  </p>
                </div>
              </li>
            </ul>
          </div>
        </div>
      </div>

      <form v-else class="max-w-3xl space-y-6" @submit.prevent="save">
        <div class="flex items-center justify-between">
          <div><div class="text-base font-medium">{{ adminText('k009j') }}</div><p class="text-sm text-muted-foreground">{{ adminText('k009k') }}</p></div>
          <Switch v-model="announcementForm.enabled" />
        </div>

        <div class="space-y-4">
          <div
            v-for="(item, index) in announcementForm.items"
            :key="item.id"
            draggable="true"
            class="rounded-lg border border-border bg-card p-4 transition-[opacity,box-shadow] duration-150"
            :class="[
              draggingAnnouncementIndex === index && 'opacity-50',
              dragOverAnnouncementIndex === index && 'ring-2 ring-primary/60',
            ]"
            @dragstart="onAnnouncementDragStart(index, $event)"
            @dragover="onAnnouncementDragOver(index, $event)"
            @drop="onAnnouncementDrop(index, $event)"
            @dragend="onAnnouncementDragEnd"
          >
            <div class="mb-3 flex items-center justify-between gap-2">
              <div class="flex items-center gap-2 text-sm font-medium">
                <GripVertical class="size-4 cursor-grab text-muted-foreground hover:text-foreground active:cursor-grabbing" :title="adminText('k00j2')" />
                {{ adminText('k0009') }} #{{ index + 1 }}
              </div>
              <div class="flex items-center gap-3">
                <label class="flex items-center gap-1.5 text-sm text-muted-foreground">
                  {{ adminText('k00e2') }}
                  <Switch v-model="item.enabled" />
                </label>
                <Button variant="ghost" type="button" class="size-8" @click="removeAnnouncementItem(index)">
                  <Trash2 class="size-4" />
                  <span class="sr-only">{{ adminText('k00j1') }}</span>
                </Button>
              </div>
            </div>
            <label class="grid gap-1.5 text-sm font-medium">
              {{ adminText('k00j0') }}
              <Input v-model="item.title" class="mt-0" :placeholder="adminText('k00j0')" />
            </label>
            <label class="mt-3 grid gap-1.5 text-sm font-medium">
              {{ adminText('k009l') }}
              <Textarea v-model="item.content" class="min-h-32 resize-y font-mono text-sm" :placeholder="adminText('k004n')" />
            </label>
          </div>

          <Button variant="outline" type="button" @click="addAnnouncementItem">
            <Plus class="size-4" />
            {{ adminText('k00j2') }}
          </Button>
        </div>
      </form>

      <Dialog :open="migrateConfirm" @update:open="migrateConfirm = $event">
        <DialogContent class="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{{ adminText('k00g9') }}</DialogTitle>
            <DialogDescription>{{ adminText('k00gc') }}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" type="button" @click="migrateConfirm = false">{{ adminText('k009q') }}</Button>
            <Button type="button" :disabled="migrating" @click="confirmMigrate">
              <Loader2 v-if="migrating" class="size-4 animate-spin" />
              <HardDrive v-else class="size-4" />
              {{ adminText('k00gb') }}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

    </BasicPage>
</template>

<style scoped>
.gf-admin-markdown {
  color: var(--foreground);
  font-size: 0.875rem;
  line-height: 1.7;
}

.gf-admin-markdown :deep(h2) {
  margin-bottom: 0.75rem;
  font-size: 1.125rem;
  font-weight: 600;
}

.gf-admin-markdown :deep(h3) {
  margin-top: 1.25rem;
  margin-bottom: 0.5rem;
  font-weight: 600;
}

.gf-admin-markdown :deep(p),
.gf-admin-markdown :deep(ul) {
  margin-bottom: 0.75rem;
}

.gf-admin-markdown :deep(ul) {
  padding-left: 1.25rem;
  list-style: disc;
}

.gf-admin-markdown :deep(code) {
  border-radius: 0.25rem;
  background: var(--muted);
  padding: 0.125rem 0.25rem;
  font-size: 0.8125rem;
}

.gf-admin-markdown :deep(pre) {
  margin: 0.75rem 0 1rem;
  overflow-x: auto;
  border-radius: 0.5rem;
  background: var(--muted);
  padding: 0.875rem;
}

.gf-admin-markdown :deep(pre code) {
  background: transparent;
  padding: 0;
}
</style>
