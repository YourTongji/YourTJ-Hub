<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { BookOpen, ChevronDown, HelpCircle, PenSquare, Plus, Sparkles } from '@lucide/vue'
import { PopoverArrow, PopoverContent, PopoverPortal, PopoverRoot, PopoverTrigger } from 'reka-ui'
import { useQuickPublish } from '@/site/composables/useQuickPublish'

const props = withDefaults(
  defineProps<{
    variant?: 'navbar' | 'fab'
  }>(),
  {
    variant: 'navbar',
  },
)

const { t } = useI18n()
const { openQuickPublish } = useQuickPublish()
const open = ref(false)
const itemRefs = ref<HTMLAnchorElement[]>([])

interface PublishOption {
  type: 0 | 1 | 2 | 3
  href: string
  label: string
  icon: any
  badgeClass: string
}

const publishOptions = computed<PublishOption[]>(() => [
  {
    type: 2,
    href: '/publish?type=thought',
    label: t('publish.contentTypesAction.thought'),
    icon: Sparkles,
    badgeClass:
      'flex h-8 w-8 shrink-0 items-center justify-center rounded-[9px] bg-gradient-to-b from-[#8b5cf6] to-[#6366f1] text-white shadow-[0_2px_6px_rgba(99,102,241,0.35)] transition-transform duration-150 group-hover:scale-105',
  },
  {
    type: 1,
    href: '/publish?type=question',
    label: t('publish.contentTypesAction.question'),
    icon: HelpCircle,
    badgeClass:
      'flex h-8 w-8 shrink-0 items-center justify-center rounded-[9px] bg-gradient-to-b from-[#1cd2a3] to-[#0ea883] text-white shadow-[0_2px_6px_rgba(14,168,131,0.35)] transition-transform duration-150 group-hover:scale-105',
  },
  {
    type: 3,
    href: '/publish?type=article',
    label: t('publish.contentTypesAction.article'),
    icon: BookOpen,
    badgeClass:
      'flex h-8 w-8 shrink-0 items-center justify-center rounded-[9px] bg-gradient-to-b from-[#f59e0b] to-[#ea580c] text-white shadow-[0_2px_6px_rgba(234,88,12,0.35)] transition-transform duration-150 group-hover:scale-105',
  },
])

function onOpenChange(nextOpen: boolean) {
  open.value = nextOpen
  if (nextOpen) {
    void nextTick(() => {
      itemRefs.value[0]?.focus()
    })
  }
}

function handleItemClick(event: MouseEvent, item: PublishOption) {
  open.value = false
  // 非文章类型（瞬间、提问）直接弹出弹层发布；文章跳转发布页；保留 Cmd/Ctrl/新标签页快捷操作
  if (item.type !== 3) {
    if (!event.ctrlKey && !event.metaKey && !event.shiftKey && event.button === 0) {
      event.preventDefault()
      openQuickPublish(item.type)
    }
  }
}

function handleItemKeydown(event: KeyboardEvent, index: number) {
  const count = publishOptions.value.length
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    itemRefs.value[(index + 1) % count]?.focus()
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    itemRefs.value[(index - 1 + count) % count]?.focus()
  } else if (event.key === 'Home') {
    event.preventDefault()
    itemRefs.value[0]?.focus()
  } else if (event.key === 'End') {
    event.preventDefault()
    itemRefs.value[count - 1]?.focus()
  }
}
</script>

<template>
  <PopoverRoot :open="open" @update:open="onOpenChange">
    <!-- 桌面端 (navbar) 触发按钮：恢复原版 gf-button 风格（图标 + 发布 + 下拉指示） -->
    <PopoverTrigger
      v-if="variant === 'navbar'"
      as-child
    >
      <button
        type="button"
        class="gf-button gf-button-lg gf-button-primary hidden shrink-0 whitespace-nowrap active:scale-[0.96] motion-reduce:active:scale-100 sm:inline-flex items-center gap-1.5 shadow-sm"
        :aria-label="t('publish.chooseContentType')"
        :aria-expanded="open"
        aria-haspopup="true"
      >
        <PenSquare class="h-4 w-4" />
        <span>{{ t('shell.publish') }}</span>
        <ChevronDown
          class="h-3.5 w-3.5 opacity-80 transition-transform duration-200"
          :class="{ 'rotate-180': open }"
          aria-hidden="true"
        />
      </button>
    </PopoverTrigger>

    <!-- 移动端 (<sm) FAB 触发按钮 -->
    <PopoverTrigger
      v-else
      as-child
    >
      <button
        type="button"
        class="fixed bottom-[calc(1.25rem+env(safe-area-inset-bottom))] right-5 z-40 inline-flex h-14 w-14 items-center justify-center rounded-full bg-primary text-primary-content shadow-lg transition-[transform,box-shadow] duration-150 hover:scale-105 hover:shadow-xl active:scale-[0.94] motion-reduce:transition-none motion-reduce:hover:scale-100 sm:hidden focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
        :aria-label="t('publish.chooseContentType')"
        :aria-expanded="open"
        aria-haspopup="true"
      >
        <Plus
          class="h-6 w-6 stroke-[2.6] transition-transform duration-200"
          :class="{ 'rotate-45': open }"
          aria-hidden="true"
        />
      </button>
    </PopoverTrigger>

    <PopoverPortal>
      <PopoverContent
        :side="variant === 'fab' ? 'top' : 'bottom'"
        align="center"
        :side-offset="variant === 'fab' ? 12 : 8"
        :collision-padding="16"
        class="gf-menu-surface z-[70] min-w-[148px] max-w-[176px] rounded-2xl border border-line/60 bg-base-100/98 p-1.5 shadow-[0_12px_36px_-6px_rgba(0,0,0,0.14),0_4px_12px_-2px_rgba(0,0,0,0.06)] backdrop-blur-md outline-none transition-[opacity,transform] duration-150"
      >
        <!-- 指向按钮的小三角形（PopoverArrow） -->
        <PopoverArrow class="fill-base-100 drop-shadow-[0_-1px_1px_rgba(0,0,0,0.05)]" :width="13" :height="7" />

        <nav class="flex flex-col gap-0.5" role="menu" :aria-label="t('publish.chooseContentType')">
          <a
            v-for="(item, index) in publishOptions"
            :key="item.type"
            ref="itemRefs"
            :href="item.href"
            role="menuitem"
            class="group flex items-center gap-3 rounded-xl px-2.5 py-2 text-left transition-colors duration-150 hover:bg-base-200/80 active:scale-[0.97] motion-reduce:active:scale-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/70 cursor-pointer"
            @click="handleItemClick($event, item)"
            @keydown="handleItemKeydown($event, index)"
          >
            <div :class="item.badgeClass">
              <component :is="item.icon" class="h-4 w-4 stroke-[2.4]" aria-hidden="true" />
            </div>
            <span class="text-[15px] font-medium text-base-content/90 transition-colors group-hover:text-primary tracking-wide whitespace-nowrap">
              {{ item.label }}
            </span>
          </a>
        </nav>
      </PopoverContent>
    </PopoverPortal>
  </PopoverRoot>
</template>
