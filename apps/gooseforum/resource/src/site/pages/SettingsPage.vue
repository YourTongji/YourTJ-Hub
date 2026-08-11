<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import {
  Archive,
  ArrowLeft,
  Ban,
  CalendarDays,
  Camera,
  Check,
  FileText,
  ImagePlus,
  KeyRound,
  Link as LinkIcon,
  Loader2,
  Mail,
  Pencil,
  Feather,
  RotateCcw,
  Settings2,
  Shield,
  Sparkles,
  Trash2,
  UserRound,
  X,
} from '@lucide/vue'
import {
  changePassword,
  disableTotp,
  enableTotp,
  getOAuthBindings,
  getTotpSetup,
  getTotpStatus,
  listSessions,
  resendActivationEmail,
  revokeAllSessions,
  revokeSession,
  getDeletedContent,
  purgeDeletedContent,
  restoreDeletedContent,
  privacyEraseContent,
  getMyContent,
  batchDeleteContent,
  closeAccount,
  logout,
  savePresetAvatar,
  saveUserEmail,
  saveUserInfo,
  saveUserName,
  saveUserProfileCover,
  unbindOAuth,
  wearBadge,
  type OAuthBindingsPayload,
  type DeletedContentItem,
  type DeletedContentListResult,
  type DeletedContentType,
  type MyContentItem,
  type TotpEnablePayload,
  type TotpSetupPayload,
  type UserSessionPayload,
} from '@/runtime/api'
import { formatDate, formatNumber } from '@/runtime/format'
import { useFlashMessages, type FlashMessageType } from '@/runtime/flash-message'
import { ApiResponseError } from '@/runtime/api'
import { toDataURL } from 'qrcode'
import { useAvatarCropUpload } from '@/site/composables/useAvatarCropUpload'
import { useCoverCropUpload, COVER_ASPECT_RATIO } from '@/site/composables/useCoverCropUpload'
import AvatarImageEditor from '@/site/components/AvatarImageEditor.vue'
import CoverImageEditor from '@/site/components/CoverImageEditor.vue'
import SectionHeader from '@/site/components/SectionHeader.vue'
import SiteSelect from '@/site/components/SiteSelect.vue'
import {
  applyAppearanceSettings,
  loadAppearanceSettings,
  loadLocalFonts,
  quoteFontFamily,
  resetAppearanceSettings,
  saveAppearanceSettings,
  MAX_CUSTOM_CSS_LENGTH,
  type AppearanceSettings,
  type FontZone,
  type LocalFontInfo,
} from '@/runtime/appearance-settings'
import UserAvatar from '@/site/components/UserAvatar.vue'
import { badgeClass, badgeIconURL, badgeTooltip } from '@/site/utils/badge-style'
import { socialIcons, socialLabels } from '@/site/utils/social-icons'
import type { LayoutPayload, SettingsPageProps } from '@gooseforum/client'
import { useI18n } from 'vue-i18n'
import { supportedLocales } from '@/runtime/i18n'

const page = defineProps<{
  layout: LayoutPayload
  props: SettingsPageProps
}>()

const { t, locale } = useI18n()
const tabKeys = ['profile', 'account', 'privacy', 'binding', 'security', 'content', 'deleted', 'general'] as const
type TabKey = (typeof tabKeys)[number]

const activeTab = ref<TabKey>('profile')
const status = ref('')
const error = ref('')
const savingProfile = ref(false)
const savingUsername = ref(false)
const savingEmail = ref(false)
const sendingActivationEmail = ref(false)
const savingPassword = ref(false)
const loadingBindings = ref(false)
const bindingAction = ref('')
const sessions = ref<UserSessionPayload[]>([])
const loadingSessions = ref(false)
const revokingSessionId = ref(0)
const revokingAll = ref(false)
const revokeAllConfirmOpen = ref(false)
const totpEnabled = ref(false)
const totpLoading = ref(false)
const totpSetup = ref<TotpSetupPayload | null>(null)
const totpSetupPending = ref(false)
const totpSetupPassword = ref('')
const totpSetupCode = ref('')
const totpRecoveryCodes = ref<string[]>([])
const totpDisableCode = ref('')
const totpQrUrl = ref('')
const deletedTopics = ref<DeletedContentItem[]>([])
const deletedPosts = ref<DeletedContentItem[]>([])
const loadingDeletedContent = ref(false)
const deletedContentLoaded = ref(false)
const deletedContentAction = ref('')
const deletedTopicCursor = ref(0)
const deletedPostCursor = ref(0)
const hasMoreDeletedTopics = ref(false)
const hasMoreDeletedPosts = ref(false)
// 内容管理（PRD R9）：本人公开话题/回复，勾选批量删除 + 频率二次确认
const myTopics = ref<MyContentItem[]>([])
const myPosts = ref<MyContentItem[]>([])
const loadingMyContent = ref(false)
const myContentLoaded = ref(false)
const myContentType = ref<DeletedContentType>('topic')
const selectedMyContentIds = ref<number[]>([])
const batchDeleting = ref(false)
const batchDeleteConfirmOpen = ref(false)
const batchDeletePassword = ref('')
const batchDeleteError = ref('')
// 注销账号（PRD R10）：密码二次认证 + 输入「注销」确认
const accountCloseOpen = ref(false)
const accountCloseMode = ref<'anonymize' | 'delete'>('anonymize')
const accountClosePassword = ref('')
const accountCloseConfirmText = ref('')
const accountCloseSubmitting = ref(false)
const accountCloseError = ref('')
const editingUsername = ref(false)
const editingEmail = ref(false)
/** 签名单行展示的上限字数（信息栏与公开资料表单共用） */
const SIGNATURE_MAX_LENGTH = 40
/** 简介的上限字数（信息栏与公开资料表单共用） */
const BIO_MAX_LENGTH = 80
/** 简介/签名允许的最多换行次数（保留前 N 个 \n，多余丢弃） */
const MAX_PROFILE_NEWLINES = 4

/** 信息栏就地编辑：同一时间只开一个字段 */
type InlineProfileField = 'bio' | 'signature'
const inlineEditingField = ref<InlineProfileField | null>(null)
const inlineDraft = ref('')
const savingInlineProfile = ref(false)
const inlineFieldRef = ref<HTMLTextAreaElement | null>(null)
/** 清空确认态：用户清空有内容的字段后点保存，先弹出确认条再落库 */
const inlineConfirmClear = ref<InlineProfileField | null>(null)
const savingPresetAvatar = ref('')
const savingWornBadge = ref(false)
const presetAvatarDraft = ref(page.props.user.avatarUrl)
const wornBadgeCode = ref(page.props.user.wornBadgeCode || '')
const coverUrl = ref(page.props.user.profileCoverUrl || '')
const bindings = ref<OAuthBindingsPayload>({})
const { push: pushFlash } = useFlashMessages()
const avatarEditorRef = ref<{ save: () => void } | null>(null)
const {
  avatarInput,
  uploadingAvatar,
  avatarCropOpen,
  avatarUrl,
  avatarImageUrl,
  chooseAvatar: openAvatarPicker,
  handleAvatarChange,
  closeAvatarCrop,
  saveAvatarFromCanvas,
} = useAvatarCropUpload({
  initialAvatarUrl: page.props.user.avatarUrl,
  // 头像相关反馈统一走全局通知横幅，避免打断设置页表单区域
  onStatus: (message) => pushMediaFlash(message, 'success'),
  onError: (message) => pushMediaFlash(message, 'error'),
})
const coverEditorRef = ref<{ save: () => void } | null>(null)
const {
  coverInput,
  coverCropOpen,
  coverImageUrl,
  uploadingCover,
  chooseCover: openCoverPicker,
  handleCoverChange,
  closeCoverCrop,
  saveCoverFromCanvas,
} = useCoverCropUpload({
  // 封面相关反馈统一走全局通知横幅（成功 / 错误 / 尺寸不足）
  onStatus: (message) => pushMediaFlash(message, 'success'),
  onError: (message) => pushMediaFlash(message, 'error'),
  onWarning: (message) => pushMediaFlash(message, 'warning'),
})

const socialKeys = ['github', 'twitter', 'linkedIn', 'weibo', 'bilibili', 'zhihu'] as const

const profileForm = reactive({
  nickname: page.props.user.nickname || '',
  locale: page.props.user.locale || String(locale.value),
  bio: page.props.user.bio || '',
  signature: page.props.user.signature || '',
  websiteName: page.props.user.websiteName || '',
  website: page.props.user.website || '',
  externalInformation: buildExternalInfo(),
})

const usernameForm = reactive({
  username: page.props.user.username,
})

const emailForm = reactive({
  email: page.props.user.email,
  password: '',
})

const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const privacy = reactive({
  showTopics: true,
  showFollowing: true,
  emailNotifications: true,
})

const displayName = computed(() => profileForm.nickname || usernameForm.username)
const hasProfileBio = computed(() => Boolean(profileForm.bio.trim()))
const hasProfileSignature = computed(() => Boolean(profileForm.signature.trim()))
/**
 * 设置页信息栏：简介与签名分两行，避免「仅签名」时编辑目标歧义。
 * 公开主页仍用 bio||signature 合成主行（见 UserPage）。
 */
const profileBioIsEmpty = computed(() => !hasProfileBio.value)
const showSignatureQuote = computed(() => hasProfileSignature.value)
const showSignatureAddSlot = computed(() => !hasProfileSignature.value && inlineEditingField.value !== 'signature')
const isEditingBio = computed(() => inlineEditingField.value === 'bio')
const isEditingSignature = computed(() => inlineEditingField.value === 'signature')
const profileCoverStyle = computed(() => {
  const activeCoverUrl = coverUrl.value.trim()
  const defaultCover = 'linear-gradient(135deg, var(--gf-color-base-200) 0%, var(--gf-color-info-content) 52%, var(--gf-color-base-200) 100%)'
  if (!activeCoverUrl) {
    return {
      backgroundImage: defaultCover,
    }
  }
  return {
    backgroundImage: `url(${JSON.stringify(activeCoverUrl)}), ${defaultCover}`,
  }
})
const hasStatus = computed(() => Boolean(status.value || error.value))
const statsItems = computed(() => [
  { label: t('user.stats.topics'), value: page.props.stats.topicCount },
  { label: t('user.stats.replies'), value: page.props.stats.replyCount },
  { label: t('user.stats.likesReceived'), value: page.props.stats.likeReceivedCount },
  { label: t('user.stats.likesGiven'), value: page.props.stats.likeGivenCount },
  { label: t('user.stats.followers'), value: page.props.stats.followerCount },
  { label: t('user.stats.following'), value: page.props.stats.followingCount },
  { label: t('user.stats.bookmarks'), value: page.props.stats.collectionCount },
])
const socialItems = computed(() => socialKeys.map((key) => ({
  key,
  label: socialLabels[key],
  icon: socialIcons[key],
})))
const providers = computed(() => [
  { key: 'github', label: 'GitHub', supported: true },
  { key: 'google', label: 'Google', supported: false },
])
const localeOptions = computed(() => supportedLocales.map(item => ({
  value: item,
  label: t(`locale.${item}`),
})))
const presetAvatars = Array.from({ length: 12 }, (_, index) => `/static/pic/${index + 1}.webp`)
const presetAvatarChanged = computed(() => presetAvatarDraft.value !== avatarUrl.value)
const avatarPreviewUrl = computed(() => presetAvatarChanged.value ? presetAvatarDraft.value : avatarUrl.value)
const profileUrl = computed(() => `/u/${page.props.user.id}`)
const userBadges = computed(() => page.props.user.badges || [])
const wearableBadges = computed(() => page.props.user.wearableBadges || [])
const wornBadgePreview = computed(() => wearableBadges.value.find(item => item.code === wornBadgeCode.value) || null)

const easterEggMessages: Array<{ type: FlashMessageType; message: string }> = [
  { type: 'success', message: t('settings.easterEgg.success') },
  { type: 'info', message: t('settings.easterEgg.info') },
  { type: 'warning', message: t('settings.easterEgg.warning') },
  { type: 'error', message: t('settings.easterEgg.error') },
]

watch(
  () => page.props.user.id,
  () => {
    avatarUrl.value = page.props.user.avatarUrl
    presetAvatarDraft.value = page.props.user.avatarUrl
    usernameForm.username = page.props.user.username
    emailForm.email = page.props.user.email
    profileForm.nickname = page.props.user.nickname || ''
    profileForm.locale = page.props.user.locale || String(locale.value)
    profileForm.bio = page.props.user.bio || ''
    profileForm.signature = page.props.user.signature || ''
    profileForm.websiteName = page.props.user.websiteName || ''
    profileForm.website = page.props.user.website || ''
    profileForm.externalInformation = buildExternalInfo()
    coverUrl.value = page.props.user.profileCoverUrl || ''
    wornBadgeCode.value = page.props.user.wornBadgeCode || ''
  },
)

watch(avatarUrl, (next) => {
  presetAvatarDraft.value = next
})

onMounted(() => {
  const urlTab = new URL(window.location.href).searchParams.get('tab')
  if (tabKeys.includes(urlTab as TabKey)) activeTab.value = urlTab as TabKey

  const savedPrivacy = localStorage.getItem('goose-privacy-settings')
  if (savedPrivacy) {
    Object.assign(privacy, JSON.parse(savedPrivacy))
  }
  void loadBindings()
  void loadSessions()
  void loadTotpStatus()
})

watch(activeTab, (tab) => {
  if (tab === 'deleted' && !deletedContentLoaded.value) void loadDeletedContent()
  if (tab === 'content' && !myContentLoaded.value) void loadMyContent()
})

function buildExternalInfo() {
  const info: Record<string, { link?: string }> = {}
  for (const key of socialKeys) {
    info[key] = { link: page.props.user.externalInformation?.[key]?.link || '' }
  }
  return info
}

function setActiveTab(key: TabKey) {
  activeTab.value = key
  const url = new URL(window.location.href)
  if (key === 'profile') url.searchParams.delete('tab')
  else url.searchParams.set('tab', key)
  history.replaceState(history.state, '', url)
}

function settingsTabLabel(key: string, fallback?: string) {
  if (key === 'profile') return t('settings.tabs.profile')
  if (key === 'account') return t('settings.tabs.account')
  if (key === 'privacy') return t('settings.tabs.privacy')
  if (key === 'binding') return t('settings.tabs.binding')
  if (key === 'security') return t('settings.tabs.security')
  if (key === 'content') return t('settings.tabs.content')
  if (key === 'deleted') return t('settings.tabs.deleted')
  if (key === 'general') return t('settings.tabs.general')
  return fallback || key
}

const deletedContentItems = computed(() => [
  ...deletedTopics.value,
  ...deletedPosts.value,
].sort((left, right) => right.id - left.id))

function deletedContentLabel(item: DeletedContentItem) {
  return item.contentType === 'topic'
    ? t('settings.deleted.topicLabel')
    : t('settings.deleted.replyLabel', { no: item.postNo || item.id })
}

function deletedContentTitle(item: DeletedContentItem) {
  return item.title || item.excerpt || t('settings.deleted.untitled')
}

function deletedContentActionKey(item: DeletedContentItem, action: 'restore' | 'purge' | 'privacy') {
  return `${action}:${item.contentType}:${item.id}`
}

async function loadDeletedContent(reset = true) {
  if (loadingDeletedContent.value) return
  if (!reset && !hasMoreDeletedTopics.value && !hasMoreDeletedPosts.value) return
  loadingDeletedContent.value = true
  try {
    if (reset) {
      deletedTopicCursor.value = 0
      deletedPostCursor.value = 0
      hasMoreDeletedTopics.value = false
      hasMoreDeletedPosts.value = false
    }
    const emptyResult = (nextCursorId: number): DeletedContentListResult => ({
      items: [],
      hasMore: false,
      nextCursorId,
    })
    const [topicResult, postResult] = await Promise.all([
      reset || hasMoreDeletedTopics.value
        ? getDeletedContent('topic', deletedTopicCursor.value)
        : Promise.resolve(emptyResult(deletedTopicCursor.value)),
      reset || hasMoreDeletedPosts.value
        ? getDeletedContent('post', deletedPostCursor.value)
        : Promise.resolve(emptyResult(deletedPostCursor.value)),
    ])
    deletedTopics.value = reset ? topicResult.items : [...deletedTopics.value, ...topicResult.items]
    deletedPosts.value = reset ? postResult.items : [...deletedPosts.value, ...postResult.items]
    deletedTopicCursor.value = topicResult.nextCursorId
    deletedPostCursor.value = postResult.nextCursorId
    hasMoreDeletedTopics.value = topicResult.items.length > 0 && topicResult.nextCursorId > 0
    hasMoreDeletedPosts.value = postResult.items.length > 0 && postResult.nextCursorId > 0
    deletedContentLoaded.value = true
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.deletedContentLoadFailed'))
  } finally {
    loadingDeletedContent.value = false
  }
}

async function loadMoreDeletedContent() {
  await loadDeletedContent(false)
}

// 内容管理（PRD R9）：加载本人公开内容，支持批量删除
const myContentItems = computed(() => myContentType.value === 'topic' ? myTopics.value : myPosts.value)

function myContentItemKey(item: MyContentItem) {
  return `${item.contentType}:${item.id}`
}

function isMyContentSelected(item: MyContentItem) {
  return selectedMyContentIds.value.includes(item.id)
}

function toggleMyContent(item: MyContentItem) {
  if (isMyContentSelected(item)) {
    selectedMyContentIds.value = selectedMyContentIds.value.filter(id => id !== item.id)
  } else {
    selectedMyContentIds.value = [...selectedMyContentIds.value, item.id]
  }
}

async function loadMyContent() {
  if (loadingMyContent.value) return
  loadingMyContent.value = true
  batchDeleteError.value = ''
  try {
    const result = await getMyContent(myContentType.value)
    if (myContentType.value === 'topic') {
      myTopics.value = result.items
    } else {
      myPosts.value = result.items
    }
    myContentLoaded.value = true
  } catch (error) {
    batchDeleteError.value = error instanceof Error ? error.message : t('api.deletedContentLoadFailed')
  } finally {
    loadingMyContent.value = false
  }
}

function switchMyContentType(type: DeletedContentType) {
  if (myContentType.value === type) return
  myContentType.value = type
  selectedMyContentIds.value = []
  if (type === 'topic' && myTopics.value.length === 0 && myContentLoaded.value) void loadMyContent()
  if (type === 'post' && myPosts.value.length === 0 && myContentLoaded.value) void loadMyContent()
}

async function runBatchDelete(force: boolean) {
  if (batchDeleting.value || selectedMyContentIds.value.length === 0) return
  batchDeleting.value = true
  batchDeleteError.value = ''
  try {
    const result = await batchDeleteContent(myContentType.value, selectedMyContentIds.value, force, batchDeletePassword.value)
    if (myContentType.value === 'topic') {
      myTopics.value = myTopics.value.filter(item => !selectedMyContentIds.value.includes(item.id))
    } else {
      myPosts.value = myPosts.value.filter(item => !selectedMyContentIds.value.includes(item.id))
    }
    selectedMyContentIds.value = []
    batchDeleteConfirmOpen.value = false
    batchDeletePassword.value = ''
    pushFlash(
      result.failed > 0
        ? t('settings.content.batchDeletePartial', { succeeded: result.succeeded, failed: result.failed })
        : t('settings.content.batchDeleteSuccess', { count: result.succeeded }),
      result.failed > 0 ? 'warning' : 'success',
    )
  } catch (error) {
    if (error instanceof ApiResponseError && error.messageCode === 'content.batchDelete.confirmRequired') {
      batchDeleteConfirmOpen.value = true
      return
    }
    if (error instanceof ApiResponseError && error.messageCode === 'auth.credentials.invalid') {
      batchDeleteError.value = t('settings.content.batchPasswordMismatch')
      return
    }
    batchDeleteError.value = error instanceof Error ? error.message : t('api.topicDeleteFailed')
  } finally {
    batchDeleting.value = false
  }
}

async function requestBatchDelete() {
  if (selectedMyContentIds.value.length === 0 || batchDeleting.value) return
  batchDeleteConfirmOpen.value = false
  await runBatchDelete(false)
}

function confirmForceBatchDelete() {
  if (batchDeleting.value) return
  void runBatchDelete(true)
}

function cancelBatchDeleteConfirm() {
  if (batchDeleting.value) return
  batchDeleteConfirmOpen.value = false
  batchDeletePassword.value = ''
}

// 注销账号（PRD R10）
function openAccountCloseDialog() {
  accountCloseOpen.value = true
  accountCloseMode.value = 'anonymize'
  accountClosePassword.value = ''
  accountCloseConfirmText.value = ''
  accountCloseError.value = ''
}

function closeAccountCloseDialog() {
  if (accountCloseSubmitting.value) return
  accountCloseOpen.value = false
}

async function submitAccountClose() {
  if (accountCloseSubmitting.value) return
  if (accountCloseConfirmText.value.trim() !== '注销') {
    accountCloseError.value = t('settings.account.closeConfirmMismatch')
    return
  }
  if (!accountClosePassword.value) {
    accountCloseError.value = t('settings.account.closePasswordRequired')
    return
  }
  accountCloseSubmitting.value = true
  accountCloseError.value = ''
  try {
    await closeAccount(accountCloseMode.value, accountClosePassword.value)
    await logout()
    window.location.href = '/'
  } catch (error) {
    if (error instanceof ApiResponseError && error.messageCode === 'auth.credentials.invalid') {
      accountCloseError.value = t('settings.account.closePasswordMismatch')
      return
    }
    accountCloseError.value = error instanceof Error ? error.message : t('api.operationFailed')
  } finally {
    accountCloseSubmitting.value = false
  }
}

async function restoreDeletedItem(item: DeletedContentItem) {
  if (!item.canRestore || deletedContentAction.value) return
  if (!window.confirm(t('settings.deleted.restoreConfirm'))) return
  deletedContentAction.value = deletedContentActionKey(item, 'restore')
  try {
    await restoreDeletedContent(item.contentType as DeletedContentType, item.id)
    pushFlash(t('settings.deleted.restoreSuccess'), 'success')
    await loadDeletedContent()
  } catch (err) {
    pushFlash(err instanceof Error ? err.message : t('api.contentRestoreFailed'), 'error')
  } finally {
    deletedContentAction.value = ''
  }
}

async function purgeDeletedItem(item: DeletedContentItem) {
  if (!item.canPermanent || deletedContentAction.value) return
  if (!window.confirm(t('settings.deleted.purgeConfirm'))) return
  deletedContentAction.value = deletedContentActionKey(item, 'purge')
  try {
    await purgeDeletedContent(item.contentType as DeletedContentType, item.id)
    pushFlash(t('settings.deleted.purgeSuccess'), 'success')
    await loadDeletedContent()
  } catch (err) {
    pushFlash(err instanceof Error ? err.message : t('api.contentPurgeFailed'), 'error')
  } finally {
    deletedContentAction.value = ''
  }
}

/** 隐私紧急删除（PRD R8）：跳过 30 天恢复窗口，全渠道立即彻底删除。 */
async function privacyEraseDeletedItem(item: DeletedContentItem) {
  if (deletedContentAction.value) return
  if (!window.confirm(t('settings.deleted.privacyEraseConfirm'))) return
  deletedContentAction.value = deletedContentActionKey(item, 'privacy')
  try {
    await privacyEraseContent(item.contentType as DeletedContentType, item.id)
    pushFlash(t('settings.deleted.privacyEraseSuccess'), 'success')
    await loadDeletedContent()
  } catch (err) {
    pushFlash(err instanceof Error ? err.message : t('api.contentPurgeFailed'), 'error')
  } finally {
    deletedContentAction.value = ''
  }
}

function triggerAvatarFlash() {
  const item = easterEggMessages[Math.floor(Math.random() * easterEggMessages.length)]
  pushFlash(item.message, item.type)
}

// 头像 / 封面编辑的反馈：走全局横幅，不占用设置页卡片内嵌状态条
function pushMediaFlash(message: string, type: FlashMessageType = 'info') {
  const trimmed = message.trim()
  if (!trimmed) return
  // 避免与内嵌表单状态条同时出现同一条媒体提示
  if (status.value === trimmed) status.value = ''
  if (error.value === trimmed) error.value = ''
  pushFlash(trimmed, type)
}

function showStatus(message: string) {
  error.value = ''
  status.value = message
  window.setTimeout(() => {
    if (status.value === message) status.value = ''
  }, 3000)
}

function showError(message: string) {
  status.value = ''
  error.value = message
}

function selectPresetAvatar(url: string) {
  if (savingPresetAvatar.value || uploadingAvatar.value) return
  presetAvatarDraft.value = url
}

function chooseCustomAvatar() {
  openAvatarPicker()
}

function selectWornBadge(code: string) {
  if (savingWornBadge.value) return
  const badge = userBadges.value.find(item => item.code === code)
  if (code !== '' && !badge?.isWearable) return
  wornBadgeCode.value = code
}

async function applyPresetAvatar() {
  if (savingPresetAvatar.value || uploadingAvatar.value || !presetAvatarChanged.value) return
  savingPresetAvatar.value = presetAvatarDraft.value
  try {
    avatarUrl.value = await savePresetAvatar(presetAvatarDraft.value)
    presetAvatarDraft.value = avatarUrl.value
    pushMediaFlash(t('settings.status.avatarSaved'), 'success')
  } catch (err) {
    pushMediaFlash(err instanceof Error ? err.message : t('api.avatarPresetFailed'), 'error')
  } finally {
    savingPresetAvatar.value = ''
  }
}

async function applyWornBadge() {
  if (savingWornBadge.value) return
  savingWornBadge.value = true
  try {
    await wearBadge(wornBadgeCode.value)
    showStatus(t('settings.status.badgeSaved'))
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.badgeWearFailed'))
  } finally {
    savingWornBadge.value = false
  }
}

function beginInlineEdit(field: InlineProfileField) {
  if (savingInlineProfile.value || savingProfile.value) return
  inlineEditingField.value = field
  inlineConfirmClear.value = null
  inlineDraft.value = field === 'bio' ? profileForm.bio : profileForm.signature
  void nextTick(() => {
    const fieldElement = inlineFieldRef.value
    if (!fieldElement) return
    fieldElement.focus()
    const textLength = fieldElement.value.length
    fieldElement.setSelectionRange(textLength, textLength)
  })
}

function cancelInlineEdit() {
  if (savingInlineProfile.value) return
  inlineEditingField.value = null
  inlineConfirmClear.value = null
  inlineDraft.value = ''
}

function onInlineKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    cancelInlineEdit()
    return
  }
  if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
    event.preventDefault()
    void saveInlineEdit()
  }
}

/** 限制换行次数：最多保留 MAX_PROFILE_NEWLINES 个换行，多余丢弃 */
function clampNewlines(value: string): string {
  let newlinesSeen = 0
  let result = ''
  for (const char of value) {
    if (char === '\n') {
      if (newlinesSeen >= MAX_PROFILE_NEWLINES) continue
      newlinesSeen += 1
    }
    result += char
  }
  return result
}

function onInlineInput(event: Event) {
  const nextValue = clampNewlines((event.target as HTMLTextAreaElement).value)
  inlineDraft.value = nextValue
  // 重新输入内容时退出清空确认态
  if (nextValue.trim() && inlineConfirmClear.value) {
    inlineConfirmClear.value = null
  }
}

async function saveInlineEdit() {
  const field = inlineEditingField.value
  if (!field || savingInlineProfile.value) return

  const nextValue = clampNewlines(inlineDraft.value)
  const previousValue = field === 'bio' ? profileForm.bio : profileForm.signature
  // 内容没变化：直接关闭，不产生网络改动
  if (nextValue === previousValue) {
    cancelInlineEdit()
    return
  }
  // 清空有内容的字段：先弹确认条，确认后才真正落库
  if (!nextValue.trim() && previousValue.trim() && inlineConfirmClear.value !== field) {
    inlineConfirmClear.value = field
    return
  }

  if (field === 'bio') profileForm.bio = nextValue
  else profileForm.signature = nextValue

  const normalized = normalizeSocialLinks()
  if (!normalized) {
    // 社交链接校验失败时回滚本字段，避免信息栏与未提交状态脱节
    if (field === 'bio') profileForm.bio = previousValue
    else profileForm.signature = previousValue
    return
  }

  savingInlineProfile.value = true
  try {
    await saveUserInfo({ ...profileForm, externalInformation: normalized })
    inlineEditingField.value = null
    inlineConfirmClear.value = null
    inlineDraft.value = ''
    showStatus(t('settings.status.profileSaved'))
  } catch (err) {
    if (field === 'bio') profileForm.bio = previousValue
    else profileForm.signature = previousValue
    inlineConfirmClear.value = null
    showError(err instanceof Error ? err.message : t('api.profileSaveFailed'))
  } finally {
    savingInlineProfile.value = false
  }
}

async function saveProfile() {
  // 兜底：简介/签名来自信息栏就地编辑（已 clamp），此处再保险一次防止未来入口绕过
  profileForm.bio = clampNewlines(profileForm.bio)
  profileForm.signature = clampNewlines(profileForm.signature)

  const normalized = normalizeSocialLinks()
  if (!normalized) return

  savingProfile.value = true
  try {
    await saveUserInfo({ ...profileForm, externalInformation: normalized })
    showStatus(t('settings.status.profileSaved'))
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.profileSaveFailed'))
  } finally {
    savingProfile.value = false
  }
}

// 各社交平台的裸 ID 前缀；填写完整链接时保持原样（仅允许 http/https）。
const socialBaseUrls: Record<string, string> = {
  github: 'https://github.com/',
  twitter: 'https://twitter.com/',
  linkedIn: 'https://www.linkedin.com/in/',
  weibo: 'https://weibo.com/',
  bilibili: 'https://space.bilibili.com/',
  zhihu: 'https://www.zhihu.com/people/',
}

// 保存前规范化社交链接：完整链接校验协议，裸 ID / 用户名自动补全为完整链接。
// 非法链接返回 undefined 并提示错误。
function normalizeSocialLinks(): Record<string, { link?: string }> | undefined {
  const normalized: Record<string, { link?: string }> = {}
  const invalidLabels: string[] = []

  for (const item of socialItems.value) {
    const rawValue = profileForm.externalInformation[item.key]?.link?.trim() || ''
    if (!rawValue) {
      normalized[item.key] = { link: '' }
      continue
    }
    if (/^[a-z][a-z0-9+.-]*:\/\//i.test(rawValue)) {
      try {
        const parsed = new URL(rawValue)
        if (parsed.protocol === 'http:' || parsed.protocol === 'https:') {
          normalized[item.key] = { link: parsed.toString() }
          continue
        }
      } catch {
        // 落到下方 invalid 分支
      }
      invalidLabels.push(item.label)
      continue
    }
    normalized[item.key] = { link: socialBaseUrls[item.key] + rawValue }
  }

  if (invalidLabels.length > 0) {
    showError(t('settings.social.invalidLinks', { platforms: invalidLabels.join(', ') }))
    return undefined
  }
  return normalized
}

// 封面浮层编辑器保存：按当前视图输出 canvas → 上传并保存
async function onCoverSave(canvas: HTMLCanvasElement) {
  const coverUrlFromUpload = await saveCoverFromCanvas(canvas)
  if (!coverUrlFromUpload) return
  try {
    await saveUserProfileCover(coverUrlFromUpload)
    coverUrl.value = coverUrlFromUpload
    closeCoverCrop()
    pushMediaFlash(t('user.coverSaved'), 'success')
  } catch (err) {
    pushMediaFlash(err instanceof Error ? err.message : t('api.coverSaveFailed'), 'error')
  }
}

// 头像编辑器保存：canvas → 上传
async function onAvatarSave(canvas: HTMLCanvasElement) {
  const url = await saveAvatarFromCanvas(canvas)
  if (url) {
    closeAvatarCrop()
    pushMediaFlash(t('settings.status.avatarSaved'), 'success')
  }
}

function saveAvatarViaEditor() {
  avatarEditorRef.value?.save()
}

async function saveUsername() {
  const username = usernameForm.username.trim()
  if (!username) return showError(t('settings.validation.usernameRequired'))

  savingUsername.value = true
  try {
    await saveUserName(username)
    editingUsername.value = false
    showStatus(t('settings.status.usernameSaved'))
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.usernameSaveFailed'))
  } finally {
    savingUsername.value = false
  }
}

function cancelUsernameEdit() {
  usernameForm.username = page.props.user.username
  editingUsername.value = false
}

async function saveEmail() {
  const email = emailForm.email.trim()
  if (!email) return showError(t('settings.validation.emailRequired'))
  if (!emailForm.password) return showError(t('settings.validation.passwordRequired'))

  savingEmail.value = true
  try {
    await saveUserEmail(email, emailForm.password)
    emailForm.email = email
    editingEmail.value = false
    emailForm.password = ''
    showStatus(t('settings.status.emailSaved'))
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.emailSaveFailed'))
  } finally {
    savingEmail.value = false
  }
}

function cancelEmailEdit() {
  emailForm.email = page.props.user.email
  emailForm.password = ''
  editingEmail.value = false
}

async function sendActivationEmail() {
  sendingActivationEmail.value = true
  try {
    showStatus(await resendActivationEmail())
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.activationEmailSendFailed'))
  } finally {
    sendingActivationEmail.value = false
  }
}

async function submitPassword() {
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    return showError(t('auth.validation.passwordMismatch'))
  }

  savingPassword.value = true
  try {
    await changePassword(passwordForm.oldPassword, passwordForm.newPassword)
    passwordForm.oldPassword = ''
    passwordForm.newPassword = ''
    passwordForm.confirmPassword = ''
    showStatus(t('settings.status.passwordChanged'))
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.passwordChangeFailed'))
  } finally {
    savingPassword.value = false
  }
}

function savePrivacy() {
  localStorage.setItem('goose-privacy-settings', JSON.stringify(privacy))
  showStatus(t('settings.status.privacySaved'))
}

const appearance = reactive<AppearanceSettings>(loadAppearanceSettings())

const fontFamilyOptions = computed(() => [
  { value: 'system', label: t('settings.general.fontSystem') },
  { value: 'serif', label: t('settings.general.fontSerif') },
  { value: 'kai', label: t('settings.general.fontKai') },
  { value: 'hei', label: t('settings.general.fontHei') },
  { value: 'mono', label: t('settings.general.fontMono') },
  { value: 'custom', label: t('settings.general.fontCustom') },
])

const fontZones = computed(() => [
  { key: 'ui' as FontZone, label: t('settings.general.zoneUi'), description: t('settings.general.zoneUiDescription') },
  { key: 'body' as FontZone, label: t('settings.general.zoneBody'), description: t('settings.general.zoneBodyDescription') },
  { key: 'code' as FontZone, label: t('settings.general.zoneCode'), description: t('settings.general.zoneCodeDescription') },
])

function clampSizes() {
  for (const zone of ['ui', 'body', 'code'] as FontZone[]) {
    const raw = Number(appearance.zones[zone].size)
    appearance.zones[zone].size = Number.isFinite(raw) ? Math.min(24, Math.max(12, Math.round(raw))) : 12
  }
}

function previewAppearance() {
  applyAppearanceSettings({ ...appearance })
}

function saveAppearance() {
  clampSizes()
  appearance.customCss = appearance.customCss.slice(0, MAX_CUSTOM_CSS_LENGTH)
  saveAppearanceSettings({ ...appearance })
}

/** 本地字体加载状态（按 zone 独立）：idle / loading / loaded / unsupported / error */
type LocalFontsStatus = 'idle' | 'loading' | 'loaded' | 'unsupported' | 'error'
const localFontsStatus = reactive<Record<FontZone, LocalFontsStatus>>({ ui: 'idle', body: 'idle', code: 'idle' })
const localFonts = ref<LocalFontInfo[]>([])

async function handleLoadLocalFonts(zone: FontZone) {
  if (localFontsStatus[zone] === 'loading') return
  localFontsStatus[zone] = 'loading'
  const result = await loadLocalFonts()
  if (result.status === 'ok') {
    localFonts.value = result.fonts
    localFontsStatus[zone] = 'loaded'
  } else if (result.status === 'unsupported') {
    localFontsStatus[zone] = 'unsupported'
  } else {
    localFontsStatus[zone] = 'error'
  }
}

function selectLocalFont(zone: FontZone, font: LocalFontInfo) {
  appearance.zones[zone].customFamily = font.family
  saveAppearance()
}

function clearCustomFont(zone: FontZone) {
  appearance.zones[zone].customFamily = ''
  saveAppearance()
}

let cssPreviewTimer: number | undefined
function previewCssDebounced() {
  window.clearTimeout(cssPreviewTimer)
  cssPreviewTimer = window.setTimeout(previewAppearance, 300)
}

const cssFileInput = ref<HTMLInputElement | null>(null)
function triggerCssFileImport() {
  cssFileInput.value?.click()
}
function onCssFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => {
    const text = typeof reader.result === 'string' ? reader.result : ''
    appearance.customCss = text.slice(0, MAX_CUSTOM_CSS_LENGTH)
    saveAppearance()
  }
  reader.onerror = () => {
    showError(t('settings.general.cssImportFailed'))
  }
  reader.readAsText(file)
}
function clearCustomCss() {
  appearance.customCss = ''
  saveAppearance()
}

function confirmResetAppearance() {
  if (!window.confirm(t('settings.general.resetConfirm'))) return
  resetAppearanceSettings()
  Object.assign(appearance, loadAppearanceSettings())
}

async function loadBindings() {
  loadingBindings.value = true
  try {
    bindings.value = await getOAuthBindings()
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.bindingsLoadFailed'))
  } finally {
    loadingBindings.value = false
  }
}

async function loadSessions() {
  loadingSessions.value = true
  try {
    sessions.value = await listSessions()
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.sessionsLoadFailed'))
  } finally {
    loadingSessions.value = false
  }
}

async function loadTotpStatus() {
  try {
    const status = await getTotpStatus()
    totpEnabled.value = Boolean(status.enabled)
  } catch {
    // 状态查询失败时保持当前值（默认未启用），不打断设置页其余功能。
  }
}

async function handleRevokeSession(id: number) {
  revokingSessionId.value = id
  try {
    await revokeSession(id)
    await loadSessions()
    showStatus(t('settings.status.sessionRevoked'))
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.sessionRevokeFailed'))
  } finally {
    revokingSessionId.value = 0
  }
}

async function handleRevokeAll() {
  revokeAllConfirmOpen.value = false
  revokingAll.value = true
  try {
    await revokeAllSessions()
    // 当前会话已被吊销，后续请求都会 401；直接跳登录页让中间件重定向。
    window.location.href = '/login'
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.sessionRevokeAllFailed'))
  } finally {
    revokingAll.value = false
  }
}


async function startTotpSetup() {
  if (!totpSetupPassword.value) {
    showError(t('settings.security.totpPasswordRequired'))
    return
  }
  totpLoading.value = true
  try {
    totpSetup.value = await getTotpSetup(totpSetupPassword.value)
    totpSetupPassword.value = ''
    totpSetupCode.value = ''
    totpRecoveryCodes.value = []
    totpQrUrl.value = ''
    if (totpSetup.value.otpauthUrl) {
      totpQrUrl.value = await toDataURL(totpSetup.value.otpauthUrl, { width: 180, margin: 1 })
    }
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.totpSetupFailed'))
  } finally {
    totpLoading.value = false
  }
}

async function confirmTotpEnable() {
  if (!totpSetupCode.value) {
    showError(t('settings.security.totpCodeRequired'))
    return
  }
  totpLoading.value = true
  try {
    const result = await enableTotp(totpSetupCode.value)
    totpRecoveryCodes.value = result.recoveryCodes || []
    totpEnabled.value = true
    showStatus(t('settings.status.totpEnabled'))
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.totpEnableFailed'))
  } finally {
    totpLoading.value = false
  }
}

async function confirmTotpDisable() {
  if (!totpDisableCode.value) {
    showError(t('settings.security.totpCodeRequired'))
    return
  }
  totpLoading.value = true
  try {
    await disableTotp(totpDisableCode.value)
    totpEnabled.value = false
    totpSetup.value = null
    totpRecoveryCodes.value = []
    totpDisableCode.value = ''
    showStatus(t('settings.status.totpDisabled'))
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.totpDisableFailed'))
  } finally {
    totpLoading.value = false
  }
}

function closeTotpSetup() {
  totpSetup.value = null
  totpSetupPending.value = false
  totpSetupPassword.value = ''
  totpSetupCode.value = ''
  totpRecoveryCodes.value = []
  totpQrUrl.value = ''
}

function sessionDeviceLabel(session: UserSessionPayload) {
  const ua = session.userAgent || ''
  if (ua.includes('Windows')) return 'Windows'
  if (ua.includes('Macintosh') || ua.includes('Mac OS')) return 'macOS'
  if (ua.includes('iPhone') || ua.includes('iPad')) return 'iOS'
  if (ua.includes('Android')) return 'Android'
  if (ua.includes('Linux')) return 'Linux'
  return 'Unknown device'
}

function sessionBrowserLabel(session: UserSessionPayload) {
  const ua = session.userAgent || ''
  if (ua.includes('Edg/')) return 'Edge'
  if (ua.includes('Firefox/')) return 'Firefox'
  if (ua.includes('Chrome/')) return 'Chrome'
  if (ua.includes('Safari/')) return 'Safari'
  return 'Browser'
}

function isBound(provider: string) {
  return Boolean(bindings.value[provider]?.bound)
}

function providerActionLabel(provider: { key: string; supported: boolean }) {
  if (!provider.supported) return t('settings.binding.unsupported')
  return isBound(provider.key) ? t('settings.binding.disconnect') : t('settings.binding.connect')
}

async function toggleBinding(provider: string) {
  const item = providers.value.find((entry) => entry.key === provider)
  if (!item?.supported) return
  if (!isBound(provider)) {
    window.location.href = `/api/auth/${provider}`
    return
  }

  bindingAction.value = provider
  try {
    await unbindOAuth(provider)
    await loadBindings()
    showStatus(t('settings.status.bindingDisconnected'))
  } catch (err) {
    showError(err instanceof Error ? err.message : t('api.unbindFailed'))
  } finally {
    bindingAction.value = ''
  }
}
</script>

<template>
    <main class="min-w-0 pb-8">
      <section class="gf-card overflow-visible">
        <!-- 编辑资料页：封面右上角「设置封面」；选图后在封面区浮层编辑（非弹层） -->
        <!-- overflow-visible 保持悬浮 tooltip 不被卡片裁剪；封面图由自身圆角裁剪，封面编辑浮层不裁剪 -->
        <div
          class="relative h-36 border-b border-line bg-base-300 bg-cover bg-center sm:h-60"
          :class="coverCropOpen ? 'overflow-visible' : 'overflow-hidden rounded-t-[calc(var(--gf-radius-box)-1px)]'"
          :style="coverCropOpen ? undefined : profileCoverStyle"
        >
          <button
            v-if="!coverCropOpen"
            type="button"
            class="absolute right-3 top-3 z-10 inline-flex h-9 items-center gap-1.5 rounded-md bg-black/40 px-3 text-sm font-semibold text-white/90 backdrop-blur-sm transition hover:bg-black/55 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-white/30 disabled:cursor-wait disabled:opacity-70"
            :aria-label="t('user.editCover')"
            :disabled="uploadingCover"
            @click="openCoverPicker"
          >
            <Loader2 v-if="uploadingCover" class="h-4 w-4 animate-spin" />
            <ImagePlus v-else class="h-4 w-4" />
            {{ t('user.editCover') }}
          </button>
          <!-- 浮层：预览铺满封面高度，操作条向下盖在头像上层 -->
          <div v-if="coverCropOpen && coverImageUrl" class="absolute inset-0 z-20">
            <CoverImageEditor
              ref="coverEditorRef"
              float-mode
              :image-url="coverImageUrl"
              :aspect-ratio="COVER_ASPECT_RATIO"
              :saving="uploadingCover"
              @save="onCoverSave"
              @cancel="closeCoverCrop"
            />
          </div>
          <input
            ref="coverInput"
            type="file"
            class="hidden"
            accept="image/png,image/jpeg,image/webp,image/gif,image/bmp,image/avif"
            @change="handleCoverChange"
          />
        </div>
        <div class="relative z-0 px-4 pb-4 sm:px-5">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <!-- 移动端：头像单独一行盖封面，文字全宽在下方（与 UserPage 一致）；桌面端并排 -->
            <div class="flex min-w-0 flex-1 flex-col gap-2 sm:flex-row sm:items-start sm:gap-4">
              <button
                type="button"
                class="group relative -mt-9 h-24 w-24 shrink-0 rounded-full border-2 border-base-100 bg-base-100 shadow-sm outline-none focus-visible:ring-4 focus-visible:ring-primary/20 sm:-mt-10 sm:h-28 sm:w-28"
                :disabled="uploadingAvatar"
                :aria-label="t('settings.avatar.upload')"
                @click="chooseCustomAvatar"
              >
                <UserAvatar :src="avatarPreviewUrl" :alt="usernameForm.username" :badge="wornBadgePreview" size="large" class="h-full w-full rounded-full" img-class="rounded-full transition group-hover:brightness-90" />
                <span class="pointer-events-none absolute inset-0 flex items-center justify-center rounded-full text-neutral-content">
                  <Loader2 v-if="uploadingAvatar" class="h-8 w-8 animate-spin opacity-100" />
                  <Camera v-else class="h-8 w-8 opacity-0 drop-shadow transition group-hover:opacity-100" />
                </span>
              </button>

              <div class="min-w-0 flex-1 sm:pt-3">
                <div class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 sm:gap-y-2">
                  <h2 class="truncate text-xl font-bold leading-tight tracking-tight text-base-content sm:text-2xl">{{ displayName }}</h2>
                  <span class="gf-badge gf-badge-info rounded text-[11px]">{{ t('settings.editing') }}</span>
                  <button
                    type="button"
                    class="inline-flex h-7 w-7 items-center justify-center rounded outline-none transition hover:bg-base-300 focus-visible:ring-4 focus-visible:ring-primary/20"
                    :aria-label="t('settings.easterEgg.aria')"
                    :title="t('settings.easterEgg.title')"
                    @click="triggerAvatarFlash"
                  >
                    <Sparkles class="h-4 w-4 text-primary" />
                  </button>
                </div>
                <p class="mt-0.5 text-[13px] font-medium text-base-content/50 sm:mt-1">@{{ usernameForm.username }}</p>
                <!-- 简介：展示 / 双击就地编辑（与下方表单共享 profileForm） -->
                <div
                  class="gf-profile-inline mt-2"
                  :class="isEditingBio ? 'gf-profile-inline--editing' : 'gf-profile-inline--idle'"
                >
                  <div v-if="!isEditingBio" class="flex items-start gap-1">
                    <button
                      type="button"
                      class="gf-profile-inline__body min-w-0 flex-1 text-left"
                      :title="t('settings.profile.editBioHint')"
                      :aria-label="t('settings.profile.editBioAria')"
                      @dblclick="beginInlineEdit('bio')"
                    >
                      <p
                        class="gf-profile-bio"
                        :class="{ 'gf-profile-bio--empty': profileBioIsEmpty }"
                      >
                        {{ profileBioIsEmpty ? t('settings.profile.addBio') : profileForm.bio }}
                      </p>
                    </button>
                    <button
                      type="button"
                      class="gf-profile-inline__edit-btn mt-0.5"
                      :aria-label="t('settings.profile.editBioAria')"
                      :title="t('settings.profile.editBioHint')"
                      @click="beginInlineEdit('bio')"
                    >
                      <Pencil class="h-3.5 w-3.5" />
                    </button>
                  </div>
                  <div v-else class="gf-profile-inline__editor">
                    <textarea
                      ref="inlineFieldRef"
                      :value="inlineDraft"
                      class="gf-profile-inline__textarea"
                      rows="3"
                      :maxlength="BIO_MAX_LENGTH"
                      :disabled="savingInlineProfile"
                      :aria-label="t('settings.profile.bio')"
                      @input="onInlineInput"
                      @keydown="onInlineKeydown"
                    />
                    <div v-if="inlineConfirmClear !== 'bio'" class="gf-profile-inline__toolbar">
                      <span class="gf-profile-inline__hint">{{ t('settings.profile.inlineSaveHint') }}</span>
                      <span class="gf-profile-inline__count">{{ inlineDraft.length }}/{{ BIO_MAX_LENGTH }}</span>
                      <button
                        type="button"
                        class="gf-profile-inline__action gf-profile-inline__action--save"
                        :disabled="savingInlineProfile"
                        :aria-label="t('common.save')"
                        :title="t('common.save')"
                        @click="saveInlineEdit"
                      >
                        <Loader2 v-if="savingInlineProfile" class="h-4 w-4 animate-spin" />
                        <Check v-else class="h-4 w-4" />
                      </button>
                      <button
                        type="button"
                        class="gf-profile-inline__action"
                        :disabled="savingInlineProfile"
                        :aria-label="t('common.cancel')"
                        :title="t('common.cancel')"
                        @click="cancelInlineEdit"
                      >
                        <X class="h-4 w-4" />
                      </button>
                    </div>
                    <div v-else class="gf-profile-inline__confirm" role="alert">
                      <span class="gf-profile-inline__confirm-text">{{ t('settings.profile.confirmClearBio') }}</span>
                      <button
                        type="button"
                        class="gf-profile-inline__confirm-btn gf-profile-inline__confirm-btn--clear"
                        :disabled="savingInlineProfile"
                        @click="saveInlineEdit"
                      >
                        {{ t('settings.profile.confirmClearAction') }}
                      </button>
                      <button
                        type="button"
                        class="gf-profile-inline__confirm-btn"
                        :disabled="savingInlineProfile"
                        @click="inlineConfirmClear = null"
                      >
                        {{ t('settings.profile.confirmKeepAction') }}
                      </button>
                    </div>
                  </div>
                </div>

                <!-- 签名：始终独立一行（有内容用引用块，无内容用添加占位） -->
                <div
                  v-if="showSignatureQuote || isEditingSignature || showSignatureAddSlot"
                  class="gf-profile-inline gf-profile-inline--quote mt-1.5 sm:mt-2"
                  :class="isEditingSignature ? 'gf-profile-inline--editing' : 'gf-profile-inline--idle'"
                >
                  <div v-if="!isEditingSignature" class="flex items-start gap-1">
                    <button
                      type="button"
                      class="min-w-0 flex-1 text-left"
                      :title="t('settings.profile.editSignatureHint')"
                      :aria-label="t('settings.profile.editSignatureAria')"
                      @dblclick="beginInlineEdit('signature')"
                    >
                      <aside v-if="showSignatureQuote" class="gf-profile-signature !mt-0" :aria-label="t('user.signatureLabel')">
                        <div class="gf-profile-signature__row">
                          <Feather class="gf-profile-signature__icon" aria-hidden="true" />
                          <p class="gf-profile-signature__text">{{ profileForm.signature }}</p>
                        </div>
                        <svg class="gf-profile-signature__squiggle" viewBox="0 0 100 8" preserveAspectRatio="none" aria-hidden="true">
                          <path
                            d="M2 5 C 10 0, 18 8, 26 5 S 42 8, 50 5 S 66 8, 74 5 S 90 8, 98 5"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="1.8"
                            stroke-linecap="round"
                          />
                        </svg>
                      </aside>
                      <p
                        v-else
                        class="gf-profile-bio gf-profile-bio--empty px-1.5 py-1 sm:px-2"
                      >
                        {{ t('settings.profile.addSignature') }}
                      </p>
                    </button>
                    <button
                      type="button"
                      class="gf-profile-inline__edit-btn mt-1"
                      :aria-label="t('settings.profile.editSignatureAria')"
                      :title="t('settings.profile.editSignatureHint')"
                      @click="beginInlineEdit('signature')"
                    >
                      <Pencil class="h-3.5 w-3.5" />
                    </button>
                  </div>
                  <div v-else class="gf-profile-inline__editor">
                    <textarea
                      ref="inlineFieldRef"
                      :value="inlineDraft"
                      class="gf-profile-inline__textarea gf-profile-inline__textarea--quote"
                      rows="2"
                      :maxlength="SIGNATURE_MAX_LENGTH"
                      :disabled="savingInlineProfile"
                      :aria-label="t('settings.profile.signature')"
                      @input="onInlineInput"
                      @keydown="onInlineKeydown"
                    />
                    <div v-if="inlineConfirmClear !== 'signature'" class="gf-profile-inline__toolbar">
                      <span class="gf-profile-inline__hint">{{ t('settings.profile.inlineSaveHint') }}</span>
                      <span class="gf-profile-inline__count">{{ inlineDraft.length }}/{{ SIGNATURE_MAX_LENGTH }}</span>
                      <button
                        type="button"
                        class="gf-profile-inline__action gf-profile-inline__action--save"
                        :disabled="savingInlineProfile"
                        :aria-label="t('common.save')"
                        :title="t('common.save')"
                        @click="saveInlineEdit"
                      >
                        <Loader2 v-if="savingInlineProfile" class="h-4 w-4 animate-spin" />
                        <Check v-else class="h-4 w-4" />
                      </button>
                      <button
                        type="button"
                        class="gf-profile-inline__action"
                        :disabled="savingInlineProfile"
                        :aria-label="t('common.cancel')"
                        :title="t('common.cancel')"
                        @click="cancelInlineEdit"
                      >
                        <X class="h-4 w-4" />
                      </button>
                    </div>
                    <div v-else class="gf-profile-inline__confirm" role="alert">
                      <span class="gf-profile-inline__confirm-text">{{ t('settings.profile.confirmClearSignature') }}</span>
                      <button
                        type="button"
                        class="gf-profile-inline__confirm-btn gf-profile-inline__confirm-btn--clear"
                        :disabled="savingInlineProfile"
                        @click="saveInlineEdit"
                      >
                        {{ t('settings.profile.confirmClearAction') }}
                      </button>
                      <button
                        type="button"
                        class="gf-profile-inline__confirm-btn"
                        :disabled="savingInlineProfile"
                        @click="inlineConfirmClear = null"
                      >
                        {{ t('settings.profile.confirmKeepAction') }}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 右栏操作区：导航出口置顶/靠前，编辑动作同组 gap-2（better-layout）。
                 信息栏编辑进行中时隐藏，让编辑框展开到整行宽度（40 字签名一行放下） -->
            <div v-if="!isEditingBio && !isEditingSignature" class="flex shrink-0 flex-col items-stretch gap-2 sm:items-end">
              <div class="flex flex-wrap items-center gap-2 sm:justify-end">
                <a
                  :href="profileUrl"
                  class="gf-button gf-button-md gf-button-secondary"
                  :aria-label="t('settings.backToProfile')"
                >
                  <ArrowLeft class="h-4 w-4" />
                  {{ t('settings.backToProfile') }}
                </a>
                <button
                  type="button"
                  class="gf-button gf-button-md gf-button-secondary"
                  @click="chooseCustomAvatar"
                >
                  <Camera class="h-4 w-4" />
                  {{ t('settings.avatar.change') }}
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 grid grid-cols-4 border-y border-line py-3 lg:grid-cols-7 lg:py-4">
            <div v-for="item in statsItems" :key="item.label" class="px-1 py-1 text-center lg:px-0 lg:text-left">
              <div class="text-lg font-bold tabular-nums text-base-content lg:text-xl">{{ formatNumber(item.value) }}</div>
              <div class="mt-0.5 text-[11px] font-medium text-base-content/55 lg:text-xs">{{ item.label }}</div>
            </div>
          </div>

          <div class="mt-4 flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-base-content/55">
            <span class="inline-flex items-center gap-1.5"><CalendarDays class="h-3.5 w-3.5" /> {{ t('user.joinedAt', { date: formatDate(props.stats.createdAt) }) }}</span>
            <span v-if="profileForm.website" class="inline-flex min-w-0 items-center gap-1.5">
              <LinkIcon class="h-3.5 w-3.5 shrink-0" />
              <span class="truncate">{{ profileForm.websiteName || profileForm.website }}</span>
            </span>
          </div>

          <section class="mt-4 border-t border-line pt-3">
            <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
              <div>
                <h3 class="text-sm font-semibold text-base-content/75">{{ t('settings.avatar.presetsTitle') }}</h3>
                <p class="mt-0.5 text-xs text-base-content/50">{{ t('settings.avatar.presetsDescription') }}</p>
              </div>
              <div class="flex items-center gap-1.5">
                <button
                  v-if="presetAvatarChanged"
                  type="button"
                  class="gf-button gf-button-sm gf-button-primary h-8 text-sm"
                  :disabled="Boolean(savingPresetAvatar) || uploadingAvatar"
                  @click="applyPresetAvatar"
                >
                  <Loader2 v-if="savingPresetAvatar" class="h-3.5 w-3.5 animate-spin" />
                  <Check v-else class="h-3.5 w-3.5" />
                  {{ t('settings.avatar.applyPreset') }}
                </button>
                <button
                  type="button"
                  class="gf-button gf-button-sm gf-button-secondary h-8 text-sm"
                  :disabled="uploadingAvatar"
                  @click="chooseCustomAvatar"
                >
                  <Camera class="h-3.5 w-3.5" />
                  {{ t('settings.avatar.uploadCustom') }}
                </button>
              </div>
            </div>
            <div class="flex gap-1.5 overflow-x-auto pb-1">
              <button
                v-for="url in presetAvatars"
                :key="url"
                type="button"
                class="relative h-11 w-11 shrink-0 rounded-md border bg-base-100 p-0.5 transition hover:border-primary/50 hover:bg-base-200 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/20 disabled:cursor-wait disabled:opacity-70"
                :class="presetAvatarDraft === url ? 'border-primary ring-2 ring-primary/15' : 'border-line'"
                :disabled="Boolean(savingPresetAvatar) || uploadingAvatar"
                :aria-label="t('settings.avatar.selectPreset')"
                @click="selectPresetAvatar(url)"
              >
                <UserAvatar :src="url" :alt="t('settings.avatar.selectPreset')" class="h-full w-full rounded object-cover" />
                <span v-if="avatarUrl === url" class="absolute right-0.5 top-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-primary text-primary-content ring-2 ring-base-100">
                  <Check class="h-3 w-3" />
                </span>
                <span v-else-if="savingPresetAvatar === url" class="absolute inset-0 flex items-center justify-center rounded-md bg-base-100/70">
                  <Loader2 class="h-3.5 w-3.5 animate-spin text-primary" />
                </span>
              </button>
            </div>
          </section>

          <section v-if="userBadges.length" class="mt-4 border-t border-line pt-3">
            <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
              <div>
                <h3 class="text-sm font-semibold text-base-content/75">{{ t('settings.avatar.wornBadgeTitle') }}</h3>
                <p class="mt-0.5 text-xs text-base-content/50">{{ t('settings.avatar.wornBadgeDescription') }}</p>
              </div>
              <button
                type="button"
                class="gf-button gf-button-sm gf-button-primary h-8 text-sm"
                :disabled="savingWornBadge"
                @click="applyWornBadge"
              >
                <Loader2 v-if="savingWornBadge" class="h-3.5 w-3.5 animate-spin" />
                <Check v-else class="h-3.5 w-3.5" />
                {{ savingWornBadge ? t('settings.savingShort') : t('settings.avatar.applyWornBadge') }}
              </button>
            </div>
            <div class="flex gap-2 overflow-x-auto pb-1">
              <button
                type="button"
                class="relative flex h-16 w-16 shrink-0 flex-col items-center justify-center gap-1 rounded-md bg-base-100 px-1 py-1.5 text-xs font-semibold transition hover:bg-base-200 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/20"
                :disabled="savingWornBadge"
                @click="selectWornBadge('')"
              >
                <span class="flex h-8 w-8 items-center justify-center rounded-full bg-base-200 text-base-content/55 ring-1 ring-inset ring-line">
                  <Ban class="h-4 w-4" />
                </span>
                <span class="max-w-full truncate text-[10px] leading-4">{{ t('settings.avatar.noWornBadge') }}</span>
                <span v-if="wornBadgeCode === ''" class="absolute right-0.5 top-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-primary text-primary-content ring-2 ring-base-100">
                  <Check class="h-3 w-3" />
                </span>
              </button>
              <button
                v-for="badge in userBadges"
                :key="badge.code"
                type="button"
                class="relative flex h-16 w-16 shrink-0 flex-col items-center justify-center gap-1 rounded-md bg-base-100 px-1 py-1.5 transition focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/20 disabled:cursor-wait"
                :class="[
                  badge.isWearable ? 'hover:bg-base-200' : 'cursor-not-allowed opacity-45 grayscale',
                ]"
                :aria-disabled="!badge.isWearable"
                :disabled="savingWornBadge"
                :title="badge.isWearable ? badgeTooltip(badge) : `${badgeTooltip(badge)} · ${t('settings.avatar.badgeNotWearable')}`"
                @click="selectWornBadge(badge.code)"
              >
                <span
                  class="flex h-8 w-8 items-center justify-center ring-1 ring-inset"
                  :class="badgeClass(badge.color, badge.level)"
                  style="clip-path: polygon(25% 5%, 75% 5%, 100% 50%, 75% 95%, 25% 95%, 0 50%)"
                >
                  <img :src="badgeIconURL(badge)" :alt="badge.name" class="h-4 w-4 object-contain" />
                </span>
                <span class="max-w-full truncate text-[10px] leading-4">{{ badge.name }}</span>
                <span v-if="wornBadgeCode === badge.code" class="absolute right-0.5 top-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-primary text-primary-content ring-2 ring-base-100">
                  <Check class="h-3 w-3" />
                </span>
              </button>
            </div>
          </section>
        </div>

      <p
        v-if="hasStatus"
        class="mx-4 mb-3 rounded-md px-3 py-2 text-sm font-medium sm:mx-5"
        :class="error ? 'bg-error/10 text-error' : 'bg-success/10 text-success'"
      >
        {{ error || status }}
      </p>

        <nav class="flex overflow-x-auto border-t border-line bg-base-200/35 px-3">
            <button
              v-for="tab in props.tabs"
              :key="tab.key"
              type="button"
              class="inline-flex h-11 shrink-0 items-center border-b-2 px-4 text-sm font-semibold"
              :class="activeTab === tab.key ? 'border-primary text-primary' : 'border-transparent text-base-content/55 hover:text-base-content'"
              @click="setActiveTab(tab.key as TabKey)"
            >
              {{ settingsTabLabel(tab.key, tab.label) }}
            </button>
          </nav>

        <div class="space-y-3">
          <section v-show="activeTab === 'profile'">
            <SectionHeader :icon="UserRound" :title="t('settings.profile.title')" :description="t('settings.profile.description')" />
            <div class="space-y-6 p-4">
              <div class="grid gap-4 sm:grid-cols-2">
                <label class="block min-w-0">
                  <span class="text-sm font-medium text-base-content/75">{{ t('auth.username') }}</span>
                  <div v-if="!editingUsername" class="mt-1 flex min-w-0 items-center gap-2">
                    <div class="flex h-10 min-w-0 flex-1 items-center rounded-md border border-line bg-base-200/70 px-3 text-sm font-medium text-base-content">
                      <span class="truncate">{{ usernameForm.username }}</span>
                    </div>
                    <button
                      type="button"
                      class="inline-flex h-10 shrink-0 items-center gap-1.5 rounded-md border border-primary/20 bg-info/10 px-3 text-sm font-semibold text-primary hover:border-primary/20 hover:bg-info/10"
                      @click="editingUsername = true"
                    >
                      <Pencil class="h-4 w-4" />
                      {{ t('common.edit') }}
                    </button>
                  </div>
                  <div v-else class="mt-1 flex min-w-0 gap-2">
                    <input v-model="usernameForm.username" class="gf-input min-w-0 flex-1 border-primary/40 ring-4 ring-primary/20" />
                    <button type="button" class="gf-button gf-button-lg gf-button-primary shrink-0" :disabled="savingUsername" @click="saveUsername">
                      {{ savingUsername ? t('settings.savingShort') : t('common.save') }}
                    </button>
                    <button type="button" class="gf-button gf-button-lg gf-button-muted shrink-0 px-2.5 font-medium" @click="cancelUsernameEdit">{{ t('common.cancel') }}</button>
                  </div>
                </label>
                <div class="block min-w-0">
                  <span class="text-sm font-medium text-base-content/75">{{ t('auth.email') }}</span>
                  <div v-if="!editingEmail" class="mt-1 flex min-w-0 items-center gap-2">
                    <div class="flex h-10 min-w-0 flex-1 items-center rounded-md border border-line bg-base-200/70 px-3 text-sm font-medium text-base-content">
                      <span class="truncate">{{ emailForm.email }}</span>
                    </div>
                    <button
                      type="button"
                      class="inline-flex h-10 shrink-0 items-center gap-1.5 rounded-md border border-primary/20 bg-info/10 px-3 text-sm font-semibold text-primary hover:border-primary/20 hover:bg-info/10"
                      @click="editingEmail = true"
                    >
                      <Pencil class="h-4 w-4" />
                      {{ t('common.edit') }}
                    </button>
                  </div>
                  <div v-else class="mt-1 flex min-w-0 flex-col gap-2">
                    <input v-model="emailForm.email" type="email" class="gf-input min-w-0 border-primary/40 ring-4 ring-primary/20" />
                    <div class="flex min-w-0 gap-2">
                      <input v-model="emailForm.password" type="password" class="gf-input min-w-0 flex-1" :placeholder="t('settings.account.currentPassword')" autocomplete="current-password" />
                      <button type="button" class="gf-button gf-button-lg gf-button-primary shrink-0" :disabled="savingEmail" @click="saveEmail">
                        {{ savingEmail ? t('settings.savingShort') : t('common.save') }}
                      </button>
                      <button type="button" class="gf-button gf-button-lg gf-button-muted shrink-0 px-2.5 font-medium" @click="cancelEmailEdit">{{ t('common.cancel') }}</button>
                    </div>
                  </div>
                  <div v-if="layout.viewer.requiresEmailVerification" class="mt-2 flex flex-col gap-2 border-l-2 border-warning bg-warning/10 px-3 py-2 sm:flex-row sm:items-center sm:justify-between">
                    <span class="min-w-0 text-sm text-warning">
                      <span class="font-semibold">{{ t('settings.emailVerification.title') }}</span>
                      <span class="ml-1 text-warning">{{ t('settings.emailVerification.description') }}</span>
                    </span>
                    <button
                      type="button"
                      class="inline-flex h-8 shrink-0 items-center justify-center gap-1.5 rounded-md border border-warning/30 bg-base-100 px-3 text-sm font-semibold text-warning hover:bg-warning/15 disabled:cursor-wait disabled:opacity-70"
                      :disabled="sendingActivationEmail"
                      @click="sendActivationEmail"
                    >
                      <Loader2 v-if="sendingActivationEmail" class="h-4 w-4 animate-spin" />
                      <Mail v-else class="h-4 w-4" />
                      {{ sendingActivationEmail ? t('settings.emailVerification.sending') : t('settings.emailVerification.action') }}
                    </button>
                  </div>
                </div>
                <label class="block min-w-0">
                  <span class="text-sm font-medium text-base-content/75">{{ t('settings.profile.displayName') }}</span>
                  <input v-model="profileForm.nickname" class="gf-input mt-1" />
                </label>
                <label class="block min-w-0">
                  <span class="text-sm font-medium text-base-content/75">{{ t('settings.profile.language') }}</span>
                  <SiteSelect v-model="profileForm.locale" class="mt-1" :options="localeOptions" />
                </label>
                <label class="block">
                  <span class="text-sm font-medium text-base-content/75">{{ t('settings.profile.websiteName') }}</span>
                  <input v-model="profileForm.websiteName" class="gf-input mt-1" />
                </label>
                <label class="block">
                  <span class="text-sm font-medium text-base-content/75">{{ t('settings.profile.website') }}</span>
                  <input v-model="profileForm.website" class="gf-input mt-1" placeholder="https://example.com" />
                </label>
              </div>

              <div class="border-t border-line pt-5">
                <div class="mb-1 flex items-center gap-2">
                  <LinkIcon class="h-4 w-4 text-base-content/55" />
                  <h3 class="text-sm font-semibold text-base-content">{{ t('settings.profile.social') }}</h3>
                </div>
                <p class="mb-3 text-xs text-base-content/55">{{ t('settings.social.hint') }}</p>
                <div class="grid gap-3 sm:grid-cols-2">
                  <label v-for="item in socialItems" :key="item.key" class="block">
                    <span class="inline-flex items-center gap-2 text-sm font-medium text-base-content/75">
                      <span class="inline-flex h-5 w-5 items-center justify-center text-base-content/55">
                        <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                          <path :d="item.icon.path" />
                        </svg>
                      </span>
                      {{ item.label }}
                    </span>
                    <input v-model="profileForm.externalInformation[item.key].link" class="gf-input mt-1" :placeholder="t('settings.social.placeholder')" />
                  </label>
                </div>
              </div>

              <div class="border-t border-line pt-5">
                <button
                  type="button"
                  class="gf-button gf-button-lg gf-button-primary min-w-28 disabled:cursor-wait"
                  :disabled="savingProfile"
                  @click="saveProfile"
                >
                  <Loader2 v-if="savingProfile" class="h-4 w-4 animate-spin" />
                  <span>{{ savingProfile ? t('settings.savingShort') : t('settings.profile.save') }}</span>
                </button>
              </div>
            </div>
          </section>

          <section v-show="activeTab === 'account'">
            <SectionHeader :icon="KeyRound" :title="t('settings.account.title')" />
            <form class="max-w-xl space-y-4 p-4" @submit.prevent="submitPassword">
              <label class="block">
                <span class="text-sm font-medium text-base-content/75">{{ t('settings.account.currentPassword') }}</span>
                <input v-model="passwordForm.oldPassword" required type="password" class="gf-input mt-1" />
              </label>
              <label class="block">
                <span class="text-sm font-medium text-base-content/75">{{ t('auth.newPassword') }}</span>
                <input v-model="passwordForm.newPassword" required type="password" class="gf-input mt-1" />
                <span class="mt-1 block text-xs text-base-content/55">{{ t('settings.account.passwordHint') }}</span>
              </label>
              <label class="block">
                <span class="text-sm font-medium text-base-content/75">{{ t('auth.confirmPassword') }}</span>
                <input v-model="passwordForm.confirmPassword" required type="password" class="gf-input mt-1" />
              </label>
              <button type="submit" class="gf-button gf-button-lg gf-button-primary disabled:cursor-wait" :disabled="savingPassword">
                <Loader2 v-if="savingPassword" class="h-4 w-4 animate-spin" />
                {{ t('settings.account.changePassword') }}
              </button>
            </form>

            <div class="mx-4 mb-4 max-w-xl rounded-lg border border-error/20 bg-error/5 p-4">
              <h3 class="text-sm font-semibold text-error">{{ t('settings.account.closeTitle') }}</h3>
              <p class="mt-1 text-sm leading-6 text-base-content/60">{{ t('settings.account.closeDescription') }}</p>
              <button type="button" class="gf-button gf-button-sm gf-button-danger mt-3" @click="openAccountCloseDialog">
                <UserRound class="h-4 w-4" />
                {{ t('settings.account.closeAction') }}
              </button>
            </div>
          </section>

          <Teleport to="body">
            <div v-if="accountCloseOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="closeAccountCloseDialog">
              <div class="w-full max-w-md rounded-xl border border-line bg-base-100 p-5 shadow-xl">
                <h3 class="text-base font-semibold text-base-content">{{ t('settings.account.closeTitle') }}</h3>

                <div class="mt-3 space-y-2">
                  <label class="flex cursor-pointer items-start gap-3 rounded-lg border border-line p-3 transition hover:bg-base-200/60">
                    <input
                      v-model="accountCloseMode"
                      type="radio"
                      value="anonymize"
                      class="mt-1 h-4 w-4 accent-primary"
                    />
                    <span>
                      <span class="block text-sm font-semibold text-base-content">{{ t('settings.account.closeModeAnonymize') }}</span>
                      <span class="mt-0.5 block text-[13px] leading-5 text-base-content/55">{{ t('settings.account.closeModeAnonymizeDescription') }}</span>
                    </span>
                  </label>
                  <label class="flex cursor-pointer items-start gap-3 rounded-lg border border-line p-3 transition hover:bg-base-200/60">
                    <input
                      v-model="accountCloseMode"
                      type="radio"
                      value="delete"
                      class="mt-1 h-4 w-4 accent-primary"
                    />
                    <span>
                      <span class="block text-sm font-semibold text-base-content">{{ t('settings.account.closeModeDelete') }}</span>
                      <span class="mt-0.5 block text-[13px] leading-5 text-base-content/55">{{ t('settings.account.closeModeDeleteDescription') }}</span>
                    </span>
                  </label>
                </div>

                <label class="mt-4 block">
                  <span class="text-sm font-medium text-base-content/75">{{ t('settings.account.closeConfirmLabel') }}</span>
                  <input v-model="accountCloseConfirmText" type="text" class="gf-input mt-1" placeholder="注销" />
                </label>

                <label class="mt-4 block">
                  <span class="text-sm font-medium text-base-content/75">{{ t('settings.account.closePasswordLabel') }}</span>
                  <input
                    v-model="accountClosePassword"
                    type="password"
                    autocomplete="current-password"
                    class="gf-input mt-1"
                    :placeholder="t('settings.account.closePasswordPlaceholder')"
                    :disabled="accountCloseSubmitting"
                    @keydown.enter="submitAccountClose"
                  />
                </label>
                <p class="mt-1.5 text-xs leading-5 text-base-content/50">{{ t('settings.account.closePasswordHint') }}</p>

                <p v-if="accountCloseError" class="mt-2 text-sm text-error">{{ accountCloseError }}</p>

                <div class="mt-4 flex justify-end gap-2">
                  <button type="button" class="gf-button gf-button-sm gf-button-muted" :disabled="accountCloseSubmitting" @click="closeAccountCloseDialog">
                    {{ t('common.cancel') }}
                  </button>
                  <button type="button" class="gf-button gf-button-sm gf-button-danger" :disabled="accountCloseSubmitting || !accountClosePassword || !accountCloseConfirmText" @click="submitAccountClose">
                    <Loader2 v-if="accountCloseSubmitting" class="h-4 w-4 animate-spin" />
                    {{ t('settings.account.closeConfirmButton') }}
                  </button>
                </div>
              </div>
            </div>
          </Teleport>

          <section v-show="activeTab === 'privacy'">
            <SectionHeader :icon="Shield" :title="t('settings.privacy.title')" />
            <div class="max-w-2xl divide-y divide-line p-4">
              <label class="flex items-center justify-between gap-4 py-4">
                <span>
                  <span class="block text-sm font-semibold text-base-content">{{ t('settings.privacy.showTopics') }}</span>
                  <span class="text-sm text-base-content/55">{{ t('settings.privacy.showTopicsDescription') }}</span>
                </span>
                <input v-model="privacy.showTopics" type="checkbox" class="h-5 w-5 rounded border-line text-primary" @change="savePrivacy" />
              </label>
              <label class="flex items-center justify-between gap-4 py-4">
                <span>
                  <span class="block text-sm font-semibold text-base-content">{{ t('settings.privacy.showFollowing') }}</span>
                  <span class="text-sm text-base-content/55">{{ t('settings.privacy.showFollowingDescription') }}</span>
                </span>
                <input v-model="privacy.showFollowing" type="checkbox" class="h-5 w-5 rounded border-line text-primary" @change="savePrivacy" />
              </label>
              <label class="flex items-center justify-between gap-4 py-4">
                <span>
                  <span class="block text-sm font-semibold text-base-content">{{ t('settings.privacy.emailNotifications') }}</span>
                  <span class="text-sm text-base-content/55">{{ t('settings.privacy.emailNotificationsDescription') }}</span>
                </span>
                <input v-model="privacy.emailNotifications" type="checkbox" class="h-5 w-5 rounded border-line text-primary" @change="savePrivacy" />
              </label>
            </div>
          </section>

          <section v-show="activeTab === 'binding'">
            <SectionHeader :icon="Mail" :title="t('settings.binding.title')">
              <template #actions>
                <button type="button" class="text-xs font-medium text-primary hover:text-primary" @click="loadBindings">{{ t('settings.binding.refresh') }}</button>
              </template>
            </SectionHeader>
            <div v-if="loadingBindings" class="p-4 py-8 text-center text-sm text-base-content/55">
              <Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />
              {{ t('settings.binding.loading') }}
            </div>
            <div v-else class="space-y-3 p-4">
              <div
                v-for="provider in providers"
                :key="provider.key"
                class="flex items-center justify-between gap-4 rounded-lg border p-4"
                :class="provider.supported ? 'border-line bg-base-100' : 'border-line bg-base-200/70'"
              >
                <div class="flex min-w-0 items-center gap-3">
                  <div
                    class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border"
                    :class="provider.supported ? 'border-line bg-base-100 shadow-sm' : 'border-line bg-base-300 opacity-60'"
                  >
                    <svg v-if="provider.key === 'github'" class="h-6 w-6 text-base-content" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.44 9.8 8.21 11.39.6.11.82-.26.82-.58v-2.04c-3.34.73-4.04-1.61-4.04-1.61-.55-1.39-1.34-1.76-1.34-1.76-1.09-.75.08-.73.08-.73 1.21.08 1.84 1.24 1.84 1.24 1.07 1.83 2.81 1.3 3.49.99.11-.78.42-1.3.76-1.6-2.67-.3-5.47-1.33-5.47-5.93 0-1.31.47-2.38 1.24-3.22-.12-.3-.54-1.52.12-3.18 0 0 1.01-.32 3.3 1.23A11.5 11.5 0 0 1 12 5.8c1.02.01 2.05.14 3.01.4 2.29-1.55 3.3-1.23 3.3-1.23.65 1.66.24 2.88.12 3.18.77.84 1.23 1.91 1.23 3.22 0 4.61-2.81 5.62-5.48 5.92.43.37.82 1.1.82 2.22v3.29c0 .32.22.7.82.58A12.01 12.01 0 0 0 24 12c0-6.63-5.37-12-12-12Z" />
                    </svg>
                    <svg v-else class="h-6 w-6" viewBox="0 0 24 24" aria-hidden="true">
                      <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.31v2.77h3.56c2.08-1.92 3.28-4.74 3.28-8.09Z" />
                      <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.56-2.77c-.99.66-2.24 1.06-3.72 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84A11 11 0 0 0 12 23Z" />
                      <path fill="#FBBC05" d="M5.84 14.1A6.61 6.61 0 0 1 5.5 12c0-.73.12-1.44.34-2.1V7.07H2.18A11 11 0 0 0 1 12c0 1.78.43 3.45 1.18 4.93l3.66-2.83Z" />
                      <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15A10.6 10.6 0 0 0 12 1 11 11 0 0 0 2.18 7.07L5.84 9.9C6.71 7.31 9.14 5.38 12 5.38Z" />
                    </svg>
                  </div>
                  <div>
                    <h3 class="font-semibold text-base-content">{{ provider.label }}</h3>
                    <p class="text-sm" :class="provider.supported ? 'text-base-content/55' : 'text-base-content/55'">
                      {{ provider.supported ? (isBound(provider.key) ? t('settings.binding.connected') : t('settings.binding.disconnected')) : t('settings.binding.siteUnsupported') }}
                    </p>
                  </div>
                </div>
                <button
                  type="button"
                  class="inline-flex h-9 min-w-24 items-center justify-center gap-2 rounded-md border px-3 text-sm font-semibold disabled:cursor-not-allowed"
                  :class="[
                    !provider.supported
                      ? 'border-line bg-base-300 text-base-content/55'
                      : isBound(provider.key)
                        ? 'border-error/30 bg-error/10 text-error hover:bg-error/10'
                        : 'border-neutral bg-neutral text-neutral-content hover:bg-neutral/90',
                  ]"
                  :disabled="bindingAction === provider.key || !provider.supported"
                  @click="toggleBinding(provider.key)"
                >
                  <Loader2 v-if="bindingAction === provider.key" class="h-4 w-4 animate-spin" />
                  <Check v-else-if="isBound(provider.key)" class="h-4 w-4" />
                  {{ providerActionLabel(provider) }}
                </button>
              </div>
            </div>
          </section>
          <section v-show="activeTab === 'security'">
            <SectionHeader :icon="Shield" :title="t('settings.security.title')" :description="t('settings.security.description')">
              <template #actions>
                <button type="button" class="text-xs font-medium text-primary hover:text-primary" @click="loadSessions">{{ t('settings.security.refresh') }}</button>
              </template>
            </SectionHeader>
            <div class="space-y-3 border-b border-line p-4">
              <div class="flex items-center justify-between gap-4 rounded-lg border border-line bg-base-100 p-4">
                <div class="min-w-0">
                  <h3 class="font-semibold text-base-content">{{ t('settings.security.totpTitle') }}</h3>
                  <p class="mt-1 text-sm text-base-content/55">{{ t('settings.security.totpDescription') }}</p>
                  <p v-if="totpEnabled" class="mt-1 text-sm font-medium text-success">{{ t('settings.security.totpEnabled') }}</p>
                </div>
                <div class="flex shrink-0 gap-2">
                  <button
                    v-if="!totpEnabled"
                    type="button"
                    class="gf-button gf-button-md gf-button-primary disabled:cursor-wait"
                    :disabled="totpLoading"
                    @click="totpSetupPending = true"
                  >
                    <Loader2 v-if="totpLoading" class="h-4 w-4 animate-spin" />
                    {{ t('settings.security.totpEnable') }}
                  </button>
                  <button
                    v-else
                    type="button"
                    class="gf-button gf-button-md gf-button-muted disabled:cursor-wait"
                    :disabled="totpLoading"
                    @click="confirmTotpDisable"
                  >
                    {{ t('settings.security.totpDisable') }}
                  </button>
                </div>
              </div>

              <div v-if="!totpEnabled && totpSetupPending && !totpSetup" class="rounded-lg border border-line bg-base-100 p-4">
                <label class="block">
                  <span class="text-sm font-medium text-base-content/75">{{ t('settings.security.totpPasswordLabel') }}</span>
                  <input
                    v-model="totpSetupPassword"
                    type="password"
                    class="gf-input mt-1"
                    :placeholder="t('settings.security.totpPasswordPlaceholder')"
                    autocomplete="current-password"
                    @keyup.enter="startTotpSetup"
                  />
                </label>
                <div class="mt-3 flex justify-end gap-2">
                  <button type="button" class="gf-button gf-button-md gf-button-muted shrink-0" @click="totpSetupPending = false; totpSetupPassword = ''">
                    {{ t('common.cancel') }}
                  </button>
                  <button type="button" class="gf-button gf-button-md gf-button-primary shrink-0 disabled:cursor-wait" :disabled="totpLoading" @click="startTotpSetup">
                    <Loader2 v-if="totpLoading" class="h-4 w-4 animate-spin" />
                    {{ t('common.save') }}
                  </button>
                </div>
              </div>

              <div v-if="totpSetup" class="rounded-lg border border-line bg-base-100 p-4">
                <h4 class="text-sm font-semibold text-base-content">{{ t('settings.security.totpSetupTitle') }}</h4>
                <p class="mt-1 text-sm text-base-content/55">{{ t('settings.security.totpSetupHint') }}</p>
                <div class="mt-3 flex flex-col items-center gap-3">
                  <img v-if="totpQrUrl" :src="totpQrUrl" :alt="t('settings.security.totpQrAlt')" class="h-44 w-44 rounded-lg border border-line bg-base-200 object-contain" />
                  <code class="rounded bg-base-200 px-2 py-1 font-mono text-sm text-base-content">{{ totpSetup.secret }}</code>
                </div>
                <div class="mt-3 flex gap-2">
                  <input
                    v-model="totpSetupCode"
                    class="gf-input min-w-0 flex-1"
                    :placeholder="t('settings.security.totpCodePlaceholder')"
                    inputmode="numeric"
                    maxlength="6"
                  />
                  <button type="button" class="gf-button gf-button-md gf-button-primary shrink-0 disabled:cursor-wait" :disabled="totpLoading" @click="confirmTotpEnable">
                    <Loader2 v-if="totpLoading" class="h-4 w-4 animate-spin" />
                    {{ t('common.save') }}
                  </button>
                  <button type="button" class="gf-button gf-button-md gf-button-muted shrink-0" @click="closeTotpSetup">
                    {{ t('common.cancel') }}
                  </button>
                </div>
                <div v-if="totpRecoveryCodes.length > 0" class="mt-3 rounded-lg border border-warning/30 bg-warning/10 p-3">
                  <p class="text-sm font-semibold text-warning">{{ t('settings.security.totpRecoveryTitle') }}</p>
                  <p class="mt-1 text-xs text-base-content/55">{{ t('settings.security.totpRecoveryHint') }}</p>
                  <div class="mt-2 grid grid-cols-2 gap-1 font-mono text-sm text-base-content">
                    <span v-for="code in totpRecoveryCodes" :key="code">{{ code }}</span>
                  </div>
                </div>
              </div>

              <div v-if="totpEnabled && !totpSetup" class="flex gap-2">
                <input
                  v-model="totpDisableCode"
                  class="gf-input min-w-0 flex-1"
                  :placeholder="t('settings.security.totpCodePlaceholder')"
                  inputmode="numeric"
                  maxlength="6"
                />
                <button type="button" class="gf-button gf-button-md gf-button-muted shrink-0 disabled:cursor-wait" :disabled="totpLoading" @click="confirmTotpDisable">
                  {{ t('settings.security.totpDisable') }}
                </button>
              </div>
            </div>
            <div v-if="loadingSessions" class="p-4 py-8 text-center text-sm text-base-content/55">
              <Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />
              {{ t('settings.security.loading') }}
            </div>
            <div v-else class="space-y-3 p-4">
              <div
                v-for="session in sessions"
                :key="session.id"
                class="flex items-center justify-between gap-4 rounded-lg border border-line bg-base-100 p-4"
              >
                <div class="flex min-w-0 items-center gap-3">
                  <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-line bg-base-100 shadow-sm">
                    <CalendarDays class="h-5 w-5 text-base-content/60" />
                  </div>
                  <div class="min-w-0">
                    <h3 class="flex items-center gap-2 font-semibold text-base-content">
                      {{ sessionDeviceLabel(session) }}
                      <span v-if="session.isCurrent" class="gf-badge gf-badge-info rounded text-[11px]">{{ t('settings.security.current') }}</span>
                    </h3>
                    <p class="truncate text-sm text-base-content/55">
                      {{ sessionBrowserLabel(session) }} · {{ session.ipMasked || '—' }}
                    </p>
                    <p class="text-xs text-base-content/45">
                      {{ formatDate(new Date(session.createdAt).toISOString()) }}
                    </p>
                  </div>
                </div>
                <button
                  type="button"
                  class="inline-flex h-9 min-w-24 shrink-0 items-center justify-center gap-2 rounded-md border border-error/30 bg-error/10 px-3 text-sm font-semibold text-error hover:bg-error/10 disabled:cursor-not-allowed disabled:opacity-60"
                  :disabled="session.isCurrent || revokingSessionId === session.id"
                  @click="handleRevokeSession(session.id)"
                >
                  <Loader2 v-if="revokingSessionId === session.id" class="h-4 w-4 animate-spin" />
                  {{ t('settings.security.revoke') }}
                </button>
              </div>
              <div v-if="sessions.length === 0" class="rounded-lg border border-line bg-base-200/70 p-6 text-center text-sm text-base-content/55">
                {{ t('settings.security.empty') }}
              </div>
              <div class="border-t border-line pt-4">
                <button
                  type="button"
                  class="gf-button gf-button-lg gf-button-error min-w-28 disabled:cursor-wait"
                  :disabled="revokingAll"
                  @click="revokeAllConfirmOpen = true"
                >
                  <Loader2 v-if="revokingAll" class="h-4 w-4 animate-spin" />
                  {{ t('settings.security.revokeAll') }}
                </button>
                <p class="mt-2 text-xs text-base-content/45">{{ t('settings.security.revokeAllHint') }}</p>
              </div>
            </div>
          </section>

          <section v-show="activeTab === 'content'">
            <SectionHeader :icon="Settings2" :title="t('settings.content.title')" :description="t('settings.content.description')" />

            <div class="gf-card overflow-hidden">
              <div class="flex items-center gap-1 border-b border-line bg-base-200/60 p-2">
                <button
                  type="button"
                  class="gf-tab"
                  :class="myContentType === 'topic' ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
                  @click="switchMyContentType('topic')"
                >
                  {{ t('settings.content.topicTab') }}
                </button>
                <button
                  type="button"
                  class="gf-tab"
                  :class="myContentType === 'post' ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
                  @click="switchMyContentType('post')"
                >
                  {{ t('settings.content.replyTab') }}
                </button>
                <div class="ml-auto flex items-center gap-2">
                  <span v-if="selectedMyContentIds.length" class="text-xs text-base-content/55">
                    {{ t('settings.content.selectedCount', { count: selectedMyContentIds.length }) }}
                  </span>
                  <button
                    type="button"
                    class="gf-button gf-button-sm gf-button-danger"
                    :disabled="selectedMyContentIds.length === 0 || batchDeleting"
                    @click="requestBatchDelete"
                  >
                    <Loader2 v-if="batchDeleting" class="h-3.5 w-3.5 animate-spin" />
                    <Trash2 v-else class="h-3.5 w-3.5" />
                    {{ t('settings.content.batchDelete') }}
                  </button>
                </div>
              </div>

              <p v-if="batchDeleteError" class="border-b border-line bg-error/10 px-3 py-2 text-sm text-error">{{ batchDeleteError }}</p>

              <div v-if="loadingMyContent && myContentItems.length === 0" class="p-4 py-10 text-center text-sm text-base-content/55">
                <Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />
                {{ t('settings.deleted.loading') }}
              </div>
              <div v-else-if="myContentItems.length === 0" class="p-6 text-center">
                <FileText class="mx-auto h-8 w-8 text-base-content/30" />
                <h3 class="mt-3 text-sm font-semibold text-base-content/75">{{ t('settings.content.emptyTitle') }}</h3>
                <p class="mx-auto mt-1 max-w-md text-sm leading-6 text-base-content/50">{{ t('settings.content.emptyDescription') }}</p>
              </div>
              <div v-else class="divide-y divide-line">
                <label
                  v-for="item in myContentItems"
                  :key="myContentItemKey(item)"
                  class="flex cursor-pointer items-start gap-3 px-4 py-3 transition hover:bg-base-200/60"
                >
                  <input
                    type="checkbox"
                    class="mt-1 h-4 w-4 shrink-0 rounded border-line accent-primary"
                    :checked="isMyContentSelected(item)"
                    @change="toggleMyContent(item)"
                  />
                  <span class="min-w-0">
                    <span class="block truncate text-sm font-semibold text-base-content">{{ item.title }}</span>
                    <span v-if="item.excerpt" class="mt-0.5 block truncate text-[13px] leading-5 text-base-content/55">{{ item.excerpt }}</span>
                    <span class="mt-0.5 block text-xs text-base-content/45">{{ formatDate(item.createdAt) }}</span>
                  </span>
                </label>
              </div>
            </div>
          </section>

          <!-- 批量删除频率二次确认（PRD R9） -->
          <Teleport to="body">
            <div v-if="batchDeleteConfirmOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="cancelBatchDeleteConfirm">
              <div class="w-full max-w-md rounded-xl border border-line bg-base-100 p-5 shadow-xl">
                <h3 class="text-base font-semibold text-base-content">{{ t('settings.content.batchConfirmTitle') }}</h3>
                <p class="mt-1 text-sm leading-6 text-base-content/60">{{ t('settings.content.batchConfirmDescription', { count: selectedMyContentIds.length }) }}</p>
                <label class="mt-4 block">
                  <span class="mb-1 block text-xs font-semibold text-base-content/70">{{ t('settings.content.batchPasswordLabel') }}</span>
                  <input
                    v-model="batchDeletePassword"
                    type="password"
                    autocomplete="current-password"
                    class="gf-input h-9 w-full text-sm"
                    :placeholder="t('settings.content.batchPasswordPlaceholder')"
                    :disabled="batchDeleting"
                    @keydown.enter="confirmForceBatchDelete"
                  />
                </label>
                <p class="mt-2 text-xs leading-5 text-base-content/50">{{ t('settings.content.batchPasswordHint') }}</p>
                <div class="mt-4 flex justify-end gap-2">
                  <button type="button" class="gf-button gf-button-sm gf-button-muted" :disabled="batchDeleting" @click="cancelBatchDeleteConfirm">
                    {{ t('common.cancel') }}
                  </button>
                  <button type="button" class="gf-button gf-button-sm gf-button-danger" :disabled="batchDeleting || !batchDeletePassword" @click="confirmForceBatchDelete">
                    <Loader2 v-if="batchDeleting" class="h-4 w-4 animate-spin" />
                    {{ t('settings.content.batchConfirmContinue') }}
                  </button>
                </div>
              </div>
            </div>
          </Teleport>

          <section v-show="activeTab === 'deleted'">
            <SectionHeader :icon="Archive" :title="t('settings.deleted.title')" :description="t('settings.deleted.description')">
              <template #actions>
                <button
                  type="button"
                  class="text-xs font-medium text-primary hover:text-primary disabled:cursor-wait disabled:opacity-60"
                  :disabled="loadingDeletedContent"
                  @click="() => loadDeletedContent(true)"
                >
                  {{ t('settings.deleted.refresh') }}
                </button>
              </template>
            </SectionHeader>

            <div v-if="loadingDeletedContent" class="p-4 py-10 text-center text-sm text-base-content/55">
              <Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />
              {{ t('settings.deleted.loading') }}
            </div>
            <div v-else-if="deletedContentItems.length === 0" class="p-6 text-center">
              <Archive class="mx-auto h-8 w-8 text-base-content/30" />
              <h3 class="mt-3 text-sm font-semibold text-base-content/75">{{ t('settings.deleted.emptyTitle') }}</h3>
              <p class="mx-auto mt-1 max-w-md text-sm leading-6 text-base-content/50">{{ t('settings.deleted.emptyDescription') }}</p>
            </div>
            <div v-else class="space-y-3 p-4">
              <article
                v-for="item in deletedContentItems"
                :key="`${item.contentType}:${item.id}`"
                class="rounded-lg border border-line bg-base-100 p-4"
              >
                <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                  <div class="min-w-0">
                    <div class="flex flex-wrap items-center gap-2 text-xs font-semibold text-base-content/50">
                      <span class="gf-badge gf-badge-info rounded">{{ deletedContentLabel(item) }}</span>
                      <span>{{ t('settings.deleted.deletedAt', { time: formatDate(item.deletedAt) }) }}</span>
                    </div>
                    <h3 class="mt-2 break-words text-sm font-semibold text-base-content">{{ deletedContentTitle(item) }}</h3>
                    <p v-if="item.contentType === 'topic'" class="mt-1 text-sm leading-6 text-base-content/55">
                      {{ item.excerpt || t('settings.deleted.topicFallback') }}
                    </p>
                    <p v-else class="mt-1 line-clamp-3 whitespace-pre-wrap text-sm leading-6 text-base-content/55">
                      {{ item.excerpt || t('settings.deleted.replyFallback') }}
                    </p>
                    <p class="mt-2 text-xs text-base-content/45">{{ t('settings.deleted.retentionHint') }}</p>
                  </div>
                  <div class="flex shrink-0 flex-wrap gap-2 sm:justify-end">
                    <button
                      v-if="item.canRestore"
                      type="button"
                      class="gf-tip gf-button gf-button-sm gf-button-primary"
                      :data-tip="t('settings.deleted.restoreHint')"
                      :disabled="Boolean(deletedContentAction)"
                      @click="restoreDeletedItem(item)"
                    >
                      <Loader2 v-if="deletedContentAction === deletedContentActionKey(item, 'restore')" class="h-3.5 w-3.5 animate-spin" />
                      <RotateCcw v-else class="h-3.5 w-3.5" />
                      {{ t('settings.deleted.restore') }}
                    </button>
                    <button
                      v-if="item.canPermanent"
                      type="button"
                      class="gf-tip gf-button gf-button-sm gf-button-danger"
                      :data-tip="t('settings.deleted.purgeHint')"
                      :disabled="Boolean(deletedContentAction)"
                      @click="purgeDeletedItem(item)"
                    >
                      <Loader2 v-if="deletedContentAction === deletedContentActionKey(item, 'purge')" class="h-3.5 w-3.5 animate-spin" />
                      <Trash2 v-else class="h-3.5 w-3.5" />
                      {{ t('settings.deleted.purge') }}
                    </button>
                    <button
                      type="button"
                      class="gf-tip gf-button gf-button-sm gf-button-secondary"
                      :data-tip="t('settings.deleted.privacyEraseHint')"
                      :disabled="Boolean(deletedContentAction)"
                      @click="privacyEraseDeletedItem(item)"
                    >
                      <Loader2 v-if="deletedContentAction === deletedContentActionKey(item, 'privacy')" class="h-3.5 w-3.5 animate-spin" />
                      <Shield v-else class="h-3.5 w-3.5" />
                      {{ t('settings.deleted.privacyErase') }}
                    </button>
                  </div>
                </div>
              </article>
              <button
                v-if="hasMoreDeletedTopics || hasMoreDeletedPosts"
                type="button"
                class="gf-button gf-button-sm gf-button-secondary mx-auto"
                :disabled="loadingDeletedContent"
                @click="loadMoreDeletedContent"
              >
                <Loader2 v-if="loadingDeletedContent" class="h-3.5 w-3.5 animate-spin" />
                {{ t('common.loadMore') }}
              </button>
            </div>
          </section>

          <section v-show="activeTab === 'general'">
            <SectionHeader :icon="Settings2" :title="t('settings.general.title')" :description="t('settings.general.description')" />
            <div class="max-w-2xl divide-y divide-line p-4">
              <div v-for="zone in fontZones" :key="zone.key" class="py-4">
                <div class="flex items-center justify-between gap-4">
                  <span class="min-w-0">
                    <span class="block text-sm font-semibold text-base-content">{{ zone.label }}</span>
                    <span class="text-sm text-base-content/55">{{ zone.description }}</span>
                  </span>
                  <div class="flex shrink-0 items-center gap-2">
                    <input
                      type="number"
                      min="12"
                      max="24"
                      step="1"
                      class="gf-input h-9 w-20 text-center text-sm"
                      v-model.number="appearance.zones[zone.key].size"
                      :aria-label="zone.label + ' ' + t('settings.general.size')"
                      @input="previewAppearance"
                      @change="saveAppearance"
                    />
                    <SiteSelect
                      class="w-44 shrink-0"
                      :options="fontFamilyOptions"
                      v-model="appearance.zones[zone.key].familyPreset"
                      @update:model-value="saveAppearance"
                    />
                  </div>
                </div>
                <div v-if="appearance.zones[zone.key].familyPreset === 'custom'" class="mt-3">
                  <div
                    v-if="appearance.zones[zone.key].customFamily"
                    class="mb-2 flex items-center justify-between gap-2 rounded-md border border-line bg-base-200/50 px-3 py-2"
                  >
                    <span
                      class="min-w-0 truncate text-sm font-medium text-base-content"
                      :style="{ fontFamily: quoteFontFamily(appearance.zones[zone.key].customFamily) }"
                    >
                      {{ appearance.zones[zone.key].customFamily }}
                    </span>
                    <button
                      type="button"
                      class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded text-base-content/45 transition hover:bg-base-300 hover:text-base-content"
                      :aria-label="t('settings.general.clearCustomFont')"
                      :title="t('settings.general.clearCustomFont')"
                      @click="clearCustomFont(zone.key)"
                    >
                      <X class="h-3.5 w-3.5" />
                    </button>
                  </div>

                  <button
                    v-if="localFontsStatus[zone.key] === 'idle' || localFontsStatus[zone.key] === 'loading' || localFontsStatus[zone.key] === 'error'"
                    type="button"
                    class="gf-button gf-button-md gf-button-secondary w-full"
                    :disabled="localFontsStatus[zone.key] === 'loading'"
                    @click="handleLoadLocalFonts(zone.key)"
                  >
                    <Loader2 v-if="localFontsStatus[zone.key] === 'loading'" class="h-4 w-4 animate-spin" />
                    <Check v-else class="h-4 w-4" />
                    {{ localFontsStatus[zone.key] === 'error' ? t('settings.general.retryLoadFonts') : t('settings.general.loadLocalFonts') }}
                  </button>

                  <p v-else-if="localFontsStatus[zone.key] === 'unsupported'" class="text-xs text-base-content/55">
                    {{ t('settings.general.fontsUnsupported') }}
                  </p>

                  <template v-else-if="localFontsStatus[zone.key] === 'loaded'">
                    <div v-if="localFonts.length" class="mt-2 max-h-56 overflow-y-auto rounded-md border border-line p-1">
                      <button
                        v-for="font in localFonts"
                        :key="font.family"
                        type="button"
                        class="flex h-9 w-full items-center gap-2 rounded-md px-2.5 text-left text-sm font-medium text-base-content hover:bg-base-200"
                        :class="appearance.zones[zone.key].customFamily === font.family ? 'bg-primary/10 text-primary' : ''"
                        :style="{ fontFamily: quoteFontFamily(font.family) }"
                        @click="selectLocalFont(zone.key, font)"
                      >
                        <span class="min-w-0 flex-1 truncate">{{ font.fullName }}</span>
                        <Check v-if="appearance.zones[zone.key].customFamily === font.family" class="h-4 w-4 shrink-0" />
                      </button>
                    </div>
                    <p v-else class="text-xs text-base-content/55">
                      {{ t('settings.general.fontsEmpty') }}
                      <button type="button" class="ml-1 text-primary underline" @click="handleLoadLocalFonts(zone.key)">
                        {{ t('settings.general.retryLoadFonts') }}
                      </button>
                    </p>
                  </template>
                </div>
              </div>

              <div class="py-4">
                <div class="flex items-center justify-between gap-4">
                  <span>
                    <span class="block text-sm font-semibold text-base-content">{{ t('settings.general.customCss') }}</span>
                    <span class="text-sm text-base-content/55">{{ t('settings.general.customCssDescription') }}</span>
                  </span>
                  <div class="flex shrink-0 items-center gap-2">
                    <button type="button" class="gf-button gf-button-sm gf-button-muted" @click="triggerCssFileImport">{{ t('settings.general.importCss') }}</button>
                    <button type="button" class="gf-button gf-button-sm gf-button-muted" :disabled="!appearance.customCss" @click="clearCustomCss">{{ t('settings.general.clearCss') }}</button>
                    <input ref="cssFileInput" type="file" accept=".css,text/css" class="hidden" @change="onCssFileChange" />
                  </div>
                </div>
                <textarea
                  v-model="appearance.customCss"
                  class="gf-textarea mt-3 h-44 w-full font-mono text-xs"
                  :placeholder="t('settings.general.customCssPlaceholder')"
                  :aria-label="t('settings.general.customCss')"
                  @input="previewCssDebounced"
                  @blur="saveAppearance"
                ></textarea>
              </div>

              <label class="flex items-center justify-between gap-4 py-4">
                <span>
                  <span class="block text-sm font-semibold text-base-content">{{ t('settings.general.clickAnimation') }}</span>
                  <span class="text-sm text-base-content/55">{{ t('settings.general.clickAnimationDescription') }}</span>
                </span>
                <input v-model="appearance.clickAnimation" type="checkbox" class="h-5 w-5 rounded border-line text-primary" @change="saveAppearance" />
              </label>

              <div class="flex items-center justify-between gap-4 py-4">
                <span class="text-sm text-base-content/55">{{ t('settings.general.resetDescription') }}</span>
                <button type="button" class="gf-button gf-button-md gf-button-muted shrink-0" @click="confirmResetAppearance">
                  {{ t('settings.general.reset') }}
                </button>
              </div>
            </div>
          </section>

          <div v-if="revokeAllConfirmOpen" class="fixed inset-0 z-[100] overflow-y-auto bg-neutral/50 px-3 py-4 backdrop-blur-sm sm:px-4" role="dialog" aria-modal="true">
            <div class="mx-auto flex min-h-full max-w-md items-center justify-center">
              <div class="gf-menu-surface w-full p-5">
                <h2 class="text-base font-semibold text-base-content">{{ t('settings.security.revokeAllConfirmTitle') }}</h2>
                <p class="mt-2 text-sm leading-relaxed text-base-content/70">{{ t('settings.security.revokeAllConfirmDescription') }}</p>
                <div class="mt-5 flex justify-end gap-2">
                  <button type="button" class="gf-button gf-button-lg gf-button-muted font-medium" @click="revokeAllConfirmOpen = false">
                    {{ t('common.cancel') }}
                  </button>
                  <button type="button" class="gf-button gf-button-lg gf-button-error font-medium" :disabled="revokingAll" @click="handleRevokeAll">
                    <Loader2 v-if="revokingAll" class="h-4 w-4 animate-spin" />
                    {{ t('settings.security.revokeAllConfirm') }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- 头像编辑模态框（知乎式：居中 1:1 预览 + 滑条缩放 + 保存） -->
      <div
        v-if="avatarCropOpen"
        class="fixed inset-0 z-[100] overflow-y-auto bg-neutral/50 px-3 py-4 backdrop-blur-sm sm:px-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby="avatar-crop-dialog-title"
      >
        <div class="mx-auto flex min-h-full items-center justify-center">
          <div class="gf-menu-surface relative flex w-full max-w-[400px] flex-col overflow-hidden">
            <button
              type="button"
              class="absolute right-3 top-3 z-10 inline-flex h-8 w-8 items-center justify-center rounded-md text-base-content/45 transition hover:bg-base-300 hover:text-base-content"
              :aria-label="t('common.close')"
              @click="closeAvatarCrop"
            >
              <X class="h-4 w-4" aria-hidden="true" />
            </button>

            <div class="px-6 pb-2 pt-8 text-center">
              <h2 id="avatar-crop-dialog-title" class="text-lg font-semibold text-base-content">
                {{ t('settings.avatar.cropTitle') }}
              </h2>
              <p class="mt-1 text-sm text-base-content/55">
                {{ t('settings.avatar.cropDescription') }}
              </p>
            </div>

            <div class="flex flex-col items-center px-6 py-4">
              <AvatarImageEditor
                v-if="avatarImageUrl"
                ref="avatarEditorRef"
                :image-url="avatarImageUrl"
                :stage-size="256"
                :output-size="300"
                :saving="uploadingAvatar"
                @save="onAvatarSave"
                @cancel="closeAvatarCrop"
              />
            </div>


            <div class="px-6 pb-6 pt-2">
              <button
                type="button"
                class="gf-button gf-button-lg gf-button-primary w-full font-semibold disabled:cursor-wait"
                :disabled="uploadingAvatar || !avatarImageUrl"
                :aria-busy="uploadingAvatar"
                @click="saveAvatarViaEditor"
              >
                <Loader2 v-if="uploadingAvatar" class="h-4 w-4 animate-spin" />
                {{ uploadingAvatar ? t('settings.avatar.uploading') : t('common.save') }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <input
        ref="avatarInput"
        type="file"
        class="hidden"
        accept="image/png,image/jpeg,image/webp,image/gif,image/bmp,image/avif"
        @change="handleAvatarChange"
      />

    </main>
</template>
