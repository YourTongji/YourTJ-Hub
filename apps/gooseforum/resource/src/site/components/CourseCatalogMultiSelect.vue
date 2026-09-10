<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { ChevronDown, Search } from '@lucide/vue'
import { useI18n } from 'vue-i18n'

type Option = { value: string; label: string }

const props = defineProps<{
  /** 表单字段名（checkbox 提交用的 name，如 department）。 */
  modelName: string
  /** 按钮上的标题（如「全部院系」）。 */
  label: string
  /** 选项列表。 */
  options: Option[]
  /** 已选中的值（数组）。 */
  selected: string[]
  /** 搜索框占位文案。 */
  placeholder?: string
  /** 左侧图标组件。 */
  icon: unknown
}>()

const { t } = useI18n()

const open = ref(false)
const panelUp = ref(false)
const search = ref('')
const triggerRef = ref<HTMLElement | null>(null)
const rootRef = ref<HTMLElement | null>(null)
const panelId = `course-filter-${Math.random().toString(36).slice(2, 9)}`

const count = computed(() => props.selected.length)

// 选项全部渲染（v-show 隐藏），避免搜索过滤时把已勾选项移出 DOM 导致 GET 提交丢失。
const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return props.options
  return props.options.filter((o) => o.label.toLowerCase().includes(q))
})

function toggle() {
  if (open.value) {
    open.value = false
    return
  }
  open.value = true
  search.value = ''
  nextTick(computePlacement)
}

// 关闭并回焦到触发按钮（键盘一致性）。
function close() {
  open.value = false
  nextTick(() => triggerRef.value?.focus())
}

// 键盘：Esc 关闭并回焦；面板打开时 ↑/↓ 在选项间移动焦点。
function onRootKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    close()
    return
  }
  if (!open.value || (e.key !== 'ArrowDown' && e.key !== 'ArrowUp')) return
  e.preventDefault()
  const boxes = Array.from(rootRef.value?.querySelectorAll<HTMLElement>('input[type="checkbox"]') ?? [])
  if (!boxes.length) return
  const idx = boxes.indexOf(document.activeElement as HTMLElement)
  const next = e.key === 'ArrowDown' ? (idx + 1) % boxes.length : (idx - 1 + boxes.length) % boxes.length
  boxes[next].focus()
}

// 依据按钮位置决定向下或向上展开，避免超出视口底部。
function computePlacement() {
  const el = triggerRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const spaceBelow = window.innerHeight - rect.bottom
  panelUp.value = spaceBelow < 320
}

function onDocMouseDown(e: MouseEvent) {
  if (!open.value) return
  if (rootRef.value && !rootRef.value.contains(e.target as Node)) open.value = false
}

onMounted(() => document.addEventListener('mousedown', onDocMouseDown))
onBeforeUnmount(() => document.removeEventListener('mousedown', onDocMouseDown))
</script>

<template>
  <div ref="rootRef" class="relative" @keydown="onRootKeydown">
    <button
      type="button"
      ref="triggerRef"
      class="gf-input gf-input-md flex w-full cursor-pointer items-center gap-2 pl-9 text-left"
      :aria-expanded="open"
      :aria-controls="open ? panelId : undefined"
      @click="toggle"
    >
      <component
        :is="icon"
        class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/40"
      />
      <span class="flex-1 truncate">{{ label }}</span>
      <span v-if="count" class="rounded-full bg-primary px-1.5 py-0.5 text-xs font-medium leading-none tabular-nums text-primary-content">
        {{ count }}
      </span>
      <ChevronDown class="h-4 w-4 shrink-0 text-base-content/50 transition-transform" :class="{ 'rotate-180': open }" />
    </button>

    <div
      v-show="open"
      :id="panelId"
      role="group"
      class="absolute left-0 z-30 w-full min-w-[15rem] rounded-md border border-line bg-base-100 shadow-md"
      :class="panelUp ? 'bottom-full mb-1' : 'top-full mt-1'"
    >
      <div class="border-b border-line/60 p-1.5">
        <div class="relative">
          <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-base-content/40" />
          <input
            v-model="search"
            type="text"
            :placeholder="placeholder ?? t('common.search')"
            :aria-label="t('common.searchOptions')"
            class="gf-input gf-input-sm w-full pl-8"
            @keydown.enter.prevent
            @click.stop
          />
        </div>
      </div>
      <div class="max-h-60 overflow-y-auto p-1.5">
        <label
          v-for="opt in options"
          v-show="search ? opt.label.toLowerCase().includes(search.trim().toLowerCase()) : true"
          :key="opt.value"
          class="flex cursor-pointer items-center gap-2 rounded px-2 py-1 text-sm text-base-content/75 transition hover:bg-base-200/60"
        >
          <input
            type="checkbox"
            :name="modelName"
            :value="opt.value"
            :checked="selected.includes(opt.value)"
            class="h-3.5 w-3.5 shrink-0 accent-primary"
          />
          <span class="truncate">{{ opt.label }}</span>
        </label>
        <p v-if="filtered.length === 0" class="px-2 py-1 text-sm text-base-content/45">{{ t('common.noMatch') }}</p>
      </div>
    </div>
  </div>
</template>
