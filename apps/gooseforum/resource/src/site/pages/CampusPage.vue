<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import type { CampusStatus, CampusDataset, CampusDatasetKey, CampusMessageSummary, CampusMessageDetail, LayoutPayload } from '@gooseforum/client'
import { ArrowUpRight, Bell, BookOpen, CalendarDays, ChartNoAxesCombined, Check, ChevronLeft, ChevronRight, GraduationCap, Link2, Loader2, RefreshCw, ShieldCheck, Shuffle, Unplug } from '@lucide/vue'
import { campusAPI, CampusError } from '@/runtime/campus-api'
import PageHeader from '@/site/components/PageHeader.vue'
import SectionHeader from '@/site/components/SectionHeader.vue'
import EmptyState from '@/site/components/EmptyState.vue'
import CampusTimetable from '@/site/components/CampusTimetable.vue'
import CampusMessageDialog from '@/site/components/CampusMessageDialog.vue'
import { timetableCourseStyle } from '@/site/utils/timetableCourseStyle'
import { campusClock, randomCampusWish } from '@/site/utils/campusGreeting'

const page = defineProps<{ layout: LayoutPayload; props: Record<string, never> }>()
const status = ref<CampusStatus | null>(null)
const loading = ref(false), busy = ref(false), error = ref(''), unbindOpen = ref(false)
const tab = ref('overview'), week = ref(1), filter = ref('')
const data = ref<Partial<Record<CampusDatasetKey, CampusDataset>>>({})
const failures = ref<Partial<Record<CampusDatasetKey, string>>>({})
const now = ref(new Date())
const clock = computed(() => campusClock(now.value))
const wish = ref(randomCampusWish())
const dateLabel = computed(() => `${clock.value.date} ${clock.value.weekday}`)
const displayName = computed(() => data.value.profile?.metrics.find(m => m.label === '姓名')?.value || '同学')
let clockTimer: ReturnType<typeof setInterval> | undefined
let controller = new AbortController()
let generation = 0
const pendingKeys = new Set<CampusDatasetKey>()
const tabs = [{ key: 'overview', label: '校园概览' }, { key: 'timetable', label: '我的课表' }, { key: 'grades', label: '学业记录' }, { key: 'cet', label: '四六级' }, { key: 'messages', label: '校园消息' }, { key: 'terms', label: '校历' }, { key: 'services', label: '连接状态' }]
const names: Record<CampusDatasetKey, string> = { profile: '个人姓名', calendar: '当前校历', timetable: '个人课表', grades: '本科成绩', summary: '学业汇总', cet: '四六级成绩', terms: '历年校历', messages: '学校消息', sports: '体测数据', health: '体测健康', arrangements: '调课安排' }
const keys = Object.keys(names) as CampusDatasetKey[]
const current = computed(() => data.value[tab.value as CampusDatasetKey])
const rows = computed(() => (current.value?.rows || []).filter(r => !filter.value || r.some(c => c.toLowerCase().includes(filter.value.toLowerCase()))))
const summary = computed(() => data.value.summary?.metrics || [])
const calendar = computed(() => data.value.calendar?.metrics || [])
const termWeeks = computed(() => Number(calendar.value.find(m => m.label === '学期周数')?.value) || 20)
const currentWeek = computed(() => {
  const value = Number(calendar.value.find(m => m.label === '教学周')?.value)
  return Number.isFinite(value) && value > 0 ? value : null
})
const courses = computed(() => data.value.timetable?.events || [])
const weekCourses = computed(() => courses.value.filter(c => !c.weeks.length || c.weeks.includes(week.value)))
const today = computed(() => clock.value.day)
const todayCourses = computed(() => currentWeek.value === null ? [] : courses.value.filter(c => c.day === today.value && (!c.weeks.length || c.weeks.includes(currentWeek.value!))).sort((a,b) => a.start - b.start))
const todayTitle = computed(() => failures.value.timetable || data.value.timetable?.status === 'unavailable' ? '课表暂时无法读取' : !data.value.timetable ? '正在读取今日课表…' : currentWeek.value === null ? '暂时无法确定教学周' : '今天没有安排课程')
const totalCredits = computed(() => Number(summary.value.find(m => m.label === '要求学分')?.value) || 0)
const completedCredits = computed(() => Number(summary.value.find(m => m.label === '已修学分')?.value) || 0)
const creditProgress = computed(() => totalCredits.value ? Math.max(0, Math.min(100, completedCredits.value / totalCredits.value * 100)) : 0)
const messages = computed(() => data.value.messages?.messages || [])
const recentMessages = computed(() => messages.value.slice(0, 5))
const filteredMessages = computed(() => messages.value.filter(m => `${m.title} ${m.publisher}`.toLowerCase().includes(filter.value.toLowerCase())))

const messageOpen = ref(false), messageLoading = ref(false), messageError = ref(''), messageNeedsAuthorization = ref(false)
const selectedMessage = ref<CampusMessageSummary | null>(null)
const messageDetail = ref<CampusMessageDetail | null>(null)
let messageController: AbortController | null = null
let messageRequest = 0
let messageTrigger: HTMLElement | null = null
function closeMessage() {
  messageRequest++
  messageController?.abort()
  messageOpen.value = false
  selectedMessage.value = null
  messageDetail.value = null
  messageLoading.value = false
  messageError.value = ''
  messageNeedsAuthorization.value = false
  const trigger = messageTrigger
  messageTrigger = null
  void nextTick(() => { if (trigger?.isConnected) trigger.focus() })
}
async function loadMessage() {
  const selected = selectedMessage.value
  if (!selected) return
  messageController?.abort()
  messageController = new AbortController()
  const request = ++messageRequest, epoch = generation
  messageLoading.value = true
  messageDetail.value = null
  messageError.value = ''
  messageNeedsAuthorization.value = false
  try {
    const value = await campusAPI.message(selected.id, messageController.signal)
    if (epoch === generation && request === messageRequest) messageDetail.value = value
  } catch (e) {
    if (epoch !== generation || request !== messageRequest) return
    messageError.value = e instanceof Error ? e.message : '消息暂时无法读取。'
    messageNeedsAuthorization.value = e instanceof CampusError && ['campus.messageAuthorizationRequired', 'campus.authorizationRequired'].includes(e.code)
  } finally { if (request === messageRequest) messageLoading.value = false }
}
function openMessage(message: CampusMessageSummary, event: Event) {
  if (!message.id) return
  messageTrigger = event.currentTarget as HTMLElement
  selectedMessage.value = message
  messageOpen.value = true
  void loadMessage()
}
function messageDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('zh-CN', { timeZone: 'Asia/Shanghai', month: 'numeric', day: 'numeric' }).format(date)
}
function resetData() {
  generation++
  closeMessage()
  loading.value = false
  pendingKeys.clear()
  controller.abort()
  controller = new AbortController()
  data.value = {}
  failures.value = {}
  filter.value = ''
}
async function loadStatus() {
  const next = await campusAPI.status(controller.signal)
  if (next.binding?.revision !== status.value?.binding?.revision) resetData()
  status.value = next
}
function activeKeys(): CampusDatasetKey[] {
  if (tab.value === 'overview') return ['profile', 'calendar', 'messages', 'timetable']
  if (tab.value === 'grades') return ['summary', 'grades']
  if (tab.value === 'timetable') return ['calendar', 'timetable']
  if (tab.value === 'services') return keys
  return [tab.value as CampusDatasetKey]
}
async function loadData(targets = activeKeys(), force = false) {
  if (!status.value?.binding || status.value.binding.needsAuthorization) return
  const requested = targets.filter(key => !pendingKeys.has(key) && (force || !data.value[key]))
  if (!requested.length) return
  const epoch = generation
  requested.forEach(key => pendingKeys.add(key))
  loading.value = true
  await Promise.all(requested.map(async key => {
    try {
      const value = await campusAPI.dataset(key, controller.signal)
      if (epoch !== generation) return
      data.value[key] = value
      delete failures.value[key]
      if (key === 'calendar' && currentWeek.value !== null) week.value = currentWeek.value
    } catch (e) {
      if (epoch !== generation || controller.signal.aborted) return
      failures.value[key] = e instanceof Error ? e.message : '读取失败'
      if (e instanceof CampusError && e.code === 'campus.authorizationRequired' && status.value?.binding) status.value.binding.needsAuthorization = true
    } finally {
      if (epoch === generation) { pendingKeys.delete(key); loading.value = pendingKeys.size > 0 }
    }
  }))
}
async function refresh() {
  error.value = ''
  try { await loadStatus(); await loadData(activeKeys(), true) }
  catch (e) { error.value = e instanceof Error ? e.message : '读取失败' }
}
async function authorize(mode: 'bind' | 'replace' | 'reauthorize') {
  busy.value = true
  error.value = ''
  try { const { url } = await campusAPI.start(mode); window.location.assign(url) }
  catch (e) { error.value = (e as Error).message; busy.value = false }
}
async function confirm(resumeMessage = false) {
  const selected = resumeMessage ? selectedMessage.value : null
  const request = messageRequest
  const trigger = messageTrigger
  busy.value = true
  error.value = ''
  try {
    await campusAPI.confirm()
    const reopen = selected && messageOpen.value && request === messageRequest
    resetData()
    await refresh()
    // Recheck the refreshed list, rather than carrying private metadata across
    // a credential change. A close while confirmation was pending stays closed.
    const message = reopen && messages.value.find(m => m.id === selected.id)
    if (message) {
      messageTrigger = trigger
      selectedMessage.value = message
      messageOpen.value = true
      void loadMessage()
    }
  } catch (e) {
    error.value = (e as Error).message
    if (messageOpen.value) { messageError.value = error.value; messageNeedsAuthorization.value = false }
    await loadStatus()
  } finally { busy.value = false }
}
async function unbind() {
  const binding = status.value?.binding
  if (!binding) return
  busy.value = true
  error.value = ''
  try { await campusAPI.unbind(binding.revision); resetData(); unbindOpen.value = false; await loadStatus() }
  catch (e) { error.value = (e as Error).message }
  finally { busy.value = false }
}
function setTab(key: string) { tab.value = key; filter.value = ''; void loadData() }
function displayMetric(value: string) {
  return /^-?\d+\.\d+$/.test(value) ? value.replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '') : value
}
function courseStyle(name: string) {
  return timetableCourseStyle(name)
}
function serviceState(key: CampusDatasetKey) {
  if (failures.value[key]) return '读取失败'
  const state = data.value[key]?.status
  return state === 'ready' ? '已连接' : state === 'empty' ? '暂无记录' : state === 'unavailable' ? '学校服务暂不可用' : loading.value ? '同步中' : '尚未同步'
}
onMounted(async () => {
  clockTimer = setInterval(() => {
    const previous = clock.value.date
    now.value = new Date()
    if (previous !== clock.value.date && tab.value === 'overview') void loadData(['calendar', 'timetable'], true)
  }, 60_000)
  if (new URLSearchParams(location.search).get('authorization') === 'failed') error.value = '学校认证未完成或已失效，请重新发起。原绑定未改变。'
  if (page.layout.viewer.isAuthenticated) {
    try { await loadStatus(); await loadData() }
    catch (e) { error.value = (e as Error).message }
  }
})
onBeforeUnmount(() => { clearInterval(clockTimer); resetData() })
</script>

<template>
  <main class="campus-page min-w-0 pb-8">
    <PageHeader title="我的校园" description="查看课表、学业记录与校园消息。" compact>
      <template #actions>
        <span class="mr-1 hidden text-xs text-base-content/55 sm:inline">{{ dateLabel }}</span>
        <button
          v-if="status?.binding"
          class="gf-button gf-button-sm gf-button-secondary text-xs"
          :disabled="loading || busy"
          aria-label="刷新校园数据"
          @click="refresh"
        >
          <RefreshCw class="h-3.5 w-3.5" :class="{ spinning: loading }" />
          {{ loading ? '同步中' : '刷新' }}
        </button>
      </template>
    </PageHeader>

    <div v-if="error" class="gf-status-message gf-status-message-error mx-4 mb-4 sm:mx-0" role="alert">{{ error }}</div>

    <section v-if="!page.layout.viewer.isAuthenticated" class="gf-card overflow-hidden py-6">
      <EmptyState :icon="GraduationCap" title="连接你的同济校园" description="登录 YourTJ，绑定同济官方身份后查看自己的校园数据。">
        <a class="gf-button gf-button-md gf-button-primary" href="/login?redirect=%2Fcampus">登录 YourTJ <ArrowUpRight class="h-4 w-4" /></a>
      </EmptyState>
    </section>
    <section v-else-if="!status" class="gf-card overflow-hidden" aria-live="polite">
      <EmptyState :loading="!error" :icon="ShieldCheck" :title="error ? '暂时无法读取连接状态' : '正在读取连接状态…'">
        <button v-if="error" class="gf-button gf-button-sm gf-button-secondary" @click="refresh">重试</button>
      </EmptyState>
    </section>
    <template v-else>
      <section class="gf-card mb-4 flex flex-wrap items-center gap-3 p-4">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-box bg-info/10 text-primary">
          <ShieldCheck class="h-5 w-5" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <h2 class="text-sm font-semibold">同济官方身份</h2>
            <span v-if="status.binding" class="gf-badge gf-badge-info">已认证</span>
          </div>
          <p class="mt-1 text-xs text-base-content/55">
            <span v-if="status.binding" class="tabular-nums">{{ status.binding.maskedId }} · 校园数据仅自己可见</span>
            <span v-else>绑定一个官方身份，即可连接校园服务。</span>
          </p>
        </div>
        <div class="flex w-full flex-wrap items-center justify-end gap-2 sm:w-auto">
          <template v-if="status.binding">
            <button v-if="status.binding.needsAuthorization" class="gf-button gf-button-sm gf-button-primary text-xs" :disabled="busy" @click="authorize('reauthorize')">重新授权</button>
            <button class="gf-button gf-button-sm gf-button-secondary text-xs" :disabled="busy" @click="authorize('replace')">换绑身份</button>
            <button class="gf-button gf-button-sm gf-button-muted text-xs" :disabled="busy" @click="unbindOpen = true">解绑</button>
          </template>
          <button v-else class="gf-button gf-button-sm gf-button-primary text-xs" :disabled="busy || !status.enabled" @click="authorize('bind')">
            <Loader2 v-if="busy" class="h-3.5 w-3.5 spinning" />
            <Link2 v-else class="h-3.5 w-3.5" />
            {{ status.enabled ? '连接同济账号' : '校园连接尚未启用' }}
          </button>
        </div>
      </section>

      <p v-if="status.binding?.needsAuthorization" class="gf-status-message gf-status-message-info mx-4 mb-4 sm:mx-0">
        学校授权需要更新。身份绑定仍然保留，重新授权后即可继续同步。
      </p>
      <section v-if="status.candidate" class="gf-card mb-4 flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0">
          <h2 class="flex items-center gap-2 text-sm font-semibold"><ShieldCheck class="h-4 w-4 shrink-0 text-primary" />{{ status.candidate.mode === 'reauthorize' ? '学校认证已完成，等待确认更新' : '确认连接 ' + status.candidate.maskedId }}</h2>
          <p class="mt-1 text-xs leading-5 text-base-content/55">
            已通过学校认证。{{ status.candidate.mode === 'replace' ? '确认后将替换当前绑定。' : status.candidate.mode === 'reauthorize' ? '点击确认更新授权后，新权限才会生效，无需再次前往学校登录。' : '确认后可在 YourTJ 查看自己的校园数据。' }}
          </p>
        </div>
        <button class="gf-button gf-button-sm gf-button-primary self-end whitespace-nowrap text-xs sm:self-auto" :disabled="busy" @click="confirm()">
          <Loader2 v-if="busy" class="h-3.5 w-3.5 spinning" /><Check v-else class="h-3.5 w-3.5" />
          确认{{ status.candidate.mode === 'replace' ? '换绑' : status.candidate.mode === 'reauthorize' ? '更新授权' : '绑定' }}
        </button>
      </section>
      <div v-if="unbindOpen" class="gf-card mb-4 flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between" role="alertdialog" aria-label="确认解绑">
        <div>
          <h2 class="text-sm font-semibold">解除官方身份连接？</h2>
          <p class="mt-1 text-xs leading-5 text-base-content/55">将删除本站保存的校园凭据，停止数据同步。之后可重新绑定，论坛内容不会受影响。</p>
        </div>
        <div class="flex shrink-0 justify-end gap-2">
          <button class="gf-button gf-button-sm gf-button-secondary text-xs" :disabled="busy" @click="unbindOpen = false">取消</button>
          <button class="gf-button gf-button-sm gf-button-danger text-xs" :disabled="busy" @click="unbind"><Unplug class="h-3.5 w-3.5" />确认解绑</button>
        </div>
      </div>

      <section v-if="!status.binding" class="gf-card overflow-hidden">
        <EmptyState :icon="GraduationCap" title="在这里查看你的校园日常" description="完成同济官方认证，即可查看课表、学业进度与学校消息。" />
        <div class="grid grid-cols-3 divide-x divide-line border-y border-line bg-base-200/50 py-4 text-center text-xs text-base-content/75">
          <div><CalendarDays class="mx-auto mb-2 h-5 w-5 text-primary" />一周课表</div>
          <div><ChartNoAxesCombined class="mx-auto mb-2 h-5 w-5 text-primary" />学业记录</div>
          <div><ShieldCheck class="mx-auto mb-2 h-5 w-5 text-primary" />私密连接</div>
        </div>
        <p class="px-4 py-3 text-xs leading-5 text-base-content/55">学校密码只在学校登录页输入；授权会自动续期，失效时会提示重新授权。</p>
      </section>
      <template v-else>
        <nav class="gf-card mb-4 flex gap-1 overflow-x-auto bg-base-200/60 p-2" aria-label="校园内容">
          <button
            v-for="item in tabs"
            :key="item.key"
            class="gf-tab"
            :class="tab === item.key ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
            :aria-current="tab === item.key ? 'page' : undefined"
            @click="setTab(item.key)"
          >{{ item.label }}</button>
        </nav>

        <div v-if="tab === 'overview'" class="space-y-4">
          <section class="gf-card p-4 sm:p-5">
            <div class="flex flex-wrap items-center gap-2 text-xs text-base-content/55">
              <CalendarDays class="h-4 w-4 text-primary" />
              <span v-if="currentWeek !== null" class="font-medium text-primary">第 {{ currentWeek }} 周</span>
              <span v-else>{{ failures.calendar ? '校历暂不可用' : '教学周待更新' }}</span>
              <span aria-hidden="true">·</span><span>{{ clock.weekday }}</span>
              <span class="ml-auto">{{ clock.date }}</span>
            </div>
            <h2 class="mt-4 text-xl font-semibold sm:text-2xl">{{ clock.greeting }}，{{ displayName }}</h2>
            <div class="mt-3 flex items-start gap-2">
              <p class="min-w-0 flex-1 text-sm leading-6 text-base-content/65">{{ wish }}</p>
              <button class="gf-icon-button h-7 w-7 shrink-0" aria-label="换一句祝福" title="换一句" @click="wish = randomCampusWish(wish)"><Shuffle class="h-3.5 w-3.5" /></button>
            </div>
          </section>
          <section class="gf-card overflow-hidden">
            <SectionHeader title="校园消息" :icon="Bell">
              <template #actions><button class="gf-button gf-button-xs gf-button-ghost text-xs" @click="setTab('messages')">查看全部 <ArrowUpRight class="h-3.5 w-3.5" /></button></template>
            </SectionHeader>
            <div v-if="recentMessages.length" class="divide-y divide-line">
              <button v-for="message in recentMessages" :key="message.id" class="flex w-full items-start gap-3 px-4 py-3 text-left hover:bg-base-200/60 disabled:cursor-default" :disabled="!message.id" @click="openMessage(message, $event)">
                <div class="min-w-0 flex-1"><h3 class="line-clamp-2 text-sm font-medium leading-6">{{ message.title }}</h3><p class="mt-1 text-xs text-base-content/55">{{ message.publisher }}</p></div>
                <time class="mt-1 shrink-0 text-xs text-base-content/45" :datetime="message.publishedAt">{{ messageDate(message.publishedAt) }}</time>
                <ChevronRight class="mt-1 h-4 w-4 shrink-0 text-base-content/40" />
              </button>
            </div>
            <EmptyState v-else :icon="Bell" :loading="!data.messages && !failures.messages" :title="failures.messages || data.messages?.status === 'unavailable' ? '校园消息暂时无法读取' : data.messages ? '暂无校园消息' : '正在读取校园消息…'" />
          </section>
          <section class="gf-card overflow-hidden">
            <SectionHeader title="今日课表" :icon="CalendarDays">
              <template #actions><button class="gf-button gf-button-xs gf-button-ghost text-xs" @click="setTab('timetable')">查看一周 <ArrowUpRight class="h-3.5 w-3.5" /></button></template>
            </SectionHeader>
            <EmptyState v-if="!todayCourses.length" :icon="BookOpen" :title="todayTitle" :description="failures.timetable || failures.calendar || '可切换到一周课表查看其他课程。'" />
            <div v-else class="space-y-2 p-4">
              <article v-for="(course,index) in todayCourses" :key="index" class="flex items-start gap-3 rounded-xl border p-3" :style="courseStyle(course.name)">
                <span class="w-16 shrink-0 rounded-field bg-base-100/60 py-2 text-center text-xs font-semibold tabular-nums text-[var(--card-title)]">{{ course.start }}–{{ course.end }} 节</span>
                <div class="min-w-0"><h3 class="text-sm font-semibold">{{ course.name }}</h3><p class="mt-1 text-xs leading-5 text-base-content/55">{{ course.room || '地点待定' }}<template v-if="course.teacher"> · {{ course.teacher }}</template></p></div>
              </article>
            </div>
          </section>
        </div>

        <section v-else-if="tab === 'timetable'" class="gf-card overflow-hidden">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
            <h2 class="text-sm font-semibold">一周课表</h2>
            <div class="flex items-center gap-1 text-xs">
              <button class="gf-icon-button h-8 w-8 disabled:opacity-40" aria-label="上一周" :disabled="week <= 1" @click="week--"><ChevronLeft class="h-4 w-4" /></button>
              <span class="min-w-16 text-center font-medium tabular-nums">第 {{ week }} 周</span>
              <button class="gf-icon-button h-8 w-8 disabled:opacity-40" aria-label="下一周" :disabled="week >= termWeeks" @click="week++"><ChevronRight class="h-4 w-4" /></button>
              <button class="gf-button gf-button-xs gf-button-secondary ml-2 text-xs" :disabled="currentWeek === null" @click="week = currentWeek ?? 1">本周</button>
            </div>
          </div>
          <p v-if="failures.timetable" class="gf-status-message gf-status-message-error m-4">{{ failures.timetable }}</p>
          <EmptyState v-if="!data.timetable && !failures.timetable" loading title="正在读取课表…" />
          <EmptyState v-else-if="failures.timetable || data.timetable?.status === 'unavailable'" :icon="BookOpen" title="课表暂时无法读取" description="学校服务暂不可用，请稍后刷新。" />
          <template v-else>
            <p v-if="!weekCourses.length" class="px-4 py-3 text-sm text-base-content/55">本周暂无课程安排。</p>
            <div class="p-2 sm:p-4"><CampusTimetable :courses="courses" :week="week" :today="today" /></div>
          </template>
          <p class="border-t border-line px-4 py-3 text-xs leading-5 text-base-content/55">第 {{ week }} 周 · {{ weekCourses.length }} 次课程安排。选课计划可在“选课排课”中单独管理。</p>
        </section>

        <section v-else-if="tab === 'services'" class="gf-card overflow-hidden">
          <SectionHeader title="数据连接" description="同济大学开放平台" :icon="Link2" />
          <div class="divide-y divide-line px-4">
            <div v-for="key in keys" :key="key" class="flex items-center justify-between gap-3 py-3.5">
              <h3 class="text-sm font-medium">{{ names[key] }}</h3>
              <span class="gf-badge shrink-0" :class="data[key]?.status === 'ready' ? 'gf-badge-info' : 'gf-badge-muted'">{{ serviceState(key) }}</span>
            </div>
          </div>
          <p class="border-t border-line bg-base-200/50 px-4 py-3 text-xs leading-6 text-base-content/55">体测和调课目前未获得可用数据；考试安排未对当前应用开放。各项服务独立读取，暂不可用的项目不会显示为零分或零条记录。</p>
        </section>

        <section v-else-if="tab === 'messages'" class="gf-card overflow-hidden">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
            <h2 class="text-sm font-semibold">校园消息</h2>
            <label class="w-full sm:w-64"><span class="sr-only">搜索校园消息</span><input v-model="filter" class="gf-input h-9 text-xs" type="search" placeholder="搜索标题或发布单位" /></label>
          </div>
          <div v-if="filteredMessages.length" class="divide-y divide-line">
            <button v-for="message in filteredMessages" :key="message.id" class="flex w-full items-start gap-3 px-4 py-3 text-left hover:bg-base-200/60" :disabled="!message.id" @click="openMessage(message, $event)">
              <div class="min-w-0 flex-1"><h3 class="text-sm font-medium leading-6">{{ message.title }}</h3><p class="mt-1 text-xs text-base-content/55">{{ message.publisher }}</p></div>
              <time class="mt-1 shrink-0 text-xs text-base-content/45" :datetime="message.publishedAt">{{ messageDate(message.publishedAt) }}</time><ChevronRight class="mt-1 h-4 w-4 shrink-0 text-base-content/40" />
            </button>
          </div>
          <EmptyState v-else :icon="Bell" :loading="!data.messages && !failures.messages" :title="failures.messages || data.messages?.status === 'unavailable' ? '校园消息暂时无法读取' : filter ? '没有匹配的消息' : '暂无校园消息'" />
        </section>

        <section v-else class="gf-card overflow-hidden">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
            <h2 class="text-sm font-semibold">{{ names[tab as CampusDatasetKey] }}</h2>
            <label class="w-full sm:w-64"><span class="sr-only">搜索记录</span><input v-model="filter" class="gf-input h-9 text-xs" type="search" placeholder="搜索课程、学期或标题" /></label>
          </div>
          <div v-if="tab === 'grades' && summary.length" class="border-b border-line">
            <dl class="grid grid-cols-2 gap-px bg-line sm:grid-cols-4">
              <div v-for="metric in summary" :key="metric.label" class="bg-base-100 p-4"><dt class="text-xs text-base-content/55">{{ metric.label }}</dt><dd class="mt-2 text-2xl font-semibold tabular-nums">{{ displayMetric(metric.value) }}<span class="ml-1 text-xs font-normal text-base-content/55">{{ metric.unit }}</span></dd></div>
            </dl>
            <div v-if="totalCredits" class="border-t border-line px-4 py-3"><div class="mb-2 flex justify-between text-xs text-base-content/55"><span>学分进度</span><span>{{ completedCredits }} / {{ totalCredits }} 学分</span></div><div class="h-2 overflow-hidden rounded-full bg-base-300"><div class="h-full rounded-full bg-primary/75" :style="{ width: `${creditProgress}%` }"></div></div></div>
          </div>
          <p v-if="tab === 'grades' && failures.summary" class="gf-status-message gf-status-message-error m-4">{{ failures.summary }}</p>
          <dl v-if="current?.metrics.length" class="flex flex-wrap gap-x-8 gap-y-4 border-b border-line p-4">
            <div v-for="metric in current.metrics" :key="metric.label">
              <dt class="text-xs text-base-content/55">{{ metric.label }}</dt>
              <dd class="mt-1 text-xl font-semibold tabular-nums">{{ displayMetric(metric.value) }}<span class="ml-1 text-xs font-normal text-base-content/55">{{ metric.unit }}</span></dd>
            </div>
          </dl>
          <div v-if="current?.series.length" class="border-b border-line bg-base-200/40 p-4 sm:p-5">
            <h3 class="mb-4 text-sm font-semibold">{{ tab === 'grades' ? '各学期平均绩点' : '历次考试成绩' }}</h3>
            <div v-for="(point,index) in current.series" :key="index" class="campus-chart-row mb-3">
              <span class="text-xs leading-5 text-base-content/75">{{ point.label }}</span>
              <div class="h-2 overflow-hidden rounded-full bg-base-300" aria-hidden="true"><div class="h-full rounded-full bg-primary/75" :style="{ width: `${Math.max(0, Math.min(100, point.value / (tab === 'cet' ? 710 : 5) * 100))}%` }"></div></div>
              <strong class="text-right text-xs font-semibold tabular-nums">{{ point.value }}</strong>
            </div>
            <p class="mt-4 text-xs text-base-content/55">{{ tab === 'grades' ? '按学校返回学期顺序展示，坐标上限 5。' : '笔试分数，上限 710。' }}</p>
          </div>
          <EmptyState v-if="failures[tab as CampusDatasetKey] || current?.status === 'unavailable'" :icon="BookOpen" title="暂时无法读取" :description="failures[tab as CampusDatasetKey] || '学校服务暂不可用，请稍后重试。'" />
          <EmptyState v-else-if="!current" loading title="正在读取学校记录…" />
          <EmptyState v-else-if="!rows.length" :icon="BookOpen" :title="filter ? '没有匹配的记录' : '学校暂无相关记录'" />
          <div v-else class="overflow-x-auto">
            <table class="campus-records w-full text-left text-sm">
              <thead class="bg-base-200/60 text-xs text-base-content/55"><tr><th v-for="column in current.columns" :key="column" scope="col">{{ column }}</th></tr></thead>
              <tbody class="divide-y divide-line"><tr v-for="(row,index) in rows" :key="index" class="hover:bg-base-200/60"><td v-for="(cell,i) in row" :key="i">{{ cell || '—' }}</td></tr></tbody>
            </table>
          </div>
          <p v-if="current" class="border-t border-line px-4 py-3 text-xs text-base-content/55">{{ rows.length }} 条{{ tab === 'messages' ? '已读取消息（列表）' : '记录' }} · {{ new Date(current.updatedAt).toLocaleTimeString('zh-CN') }} 同步</p>
        </section>
      </template>
    </template>

    <CampusMessageDialog :open="messageOpen" :summary="selectedMessage" :detail="messageDetail" :loading="messageLoading" :error="messageError" :needs-authorization="messageNeedsAuthorization" :pending-authorization="status?.candidate?.mode === 'reauthorize'" :busy="busy" @close="closeMessage" @retry="loadMessage" @confirm-authorization="confirm(true)" @authorize="authorize('reauthorize')" />

    <footer class="mt-4 flex flex-wrap items-center gap-x-2 gap-y-2 px-4 py-3 text-xs text-base-content/45 sm:px-0">
      <ShieldCheck class="h-3.5 w-3.5 shrink-0" /><span>校园数据仅自己可见 · 不会发布到论坛</span>
      <a class="ml-auto inline-flex items-center gap-1 hover:text-primary" href="https://github.com/oierxjn/OneTJ" target="_blank" rel="noopener noreferrer">感谢 OneTJ <ArrowUpRight class="h-3 w-3" /></a>
    </footer>
  </main>
</template>

<style scoped>
/* Shared forum components own surfaces, controls and typography. Only data
   visualization geometry is local to the campus page. Colors follow tokens. */
.campus-chart-row {
  display: grid;
  grid-template-columns: minmax(7rem, 12rem) minmax(2rem, 1fr) 2.5rem;
  align-items: center;
  gap: 0.75rem;
}

.campus-records th,
.campus-records td {
  padding: 0.75rem 1rem;
  line-height: 1.6;
}

.campus-records th {
  font-weight: 600;
  white-space: nowrap;
}

.campus-records td {
  min-width: 6rem;
  vertical-align: top;
}

.campus-records td:first-child {
  min-width: 10rem;
}

.campus-records td:nth-child(2) {
  min-width: 11rem;
}

.campus-page :is(button, a):focus-visible {
  outline: 2px solid var(--gf-color-primary);
  outline-offset: 2px;
}

.spinning {
  animation: spin 1.5s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 639.98px) {
  .campus-chart-row {
    grid-template-columns: minmax(0, 1fr) 2.5rem;
    gap: 0.375rem 0.75rem;
  }

  .campus-chart-row > span {
    grid-column: 1 / -1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .spinning { animation: none; }
}
</style>
