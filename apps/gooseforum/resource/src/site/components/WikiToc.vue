<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { WikiTocItem } from '@gooseforum/client'
import { resolveActiveHeading } from '@/site/utils/wiki-toc'

const props = defineProps<{
  items: WikiTocItem[]
}>()

const { t } = useI18n()
const activeId = ref('')
const rootEl = ref<HTMLElement | null>(null)
// 滚动侦测：高亮"最后一个顶边仍在阅读线之上的标题"（文档序）。
// 略低于 scrollToHeading 的 88px 粘性头偏移，保证点击跳转后标题贴到阅读线即命中。
const HEADING_OFFSET = 96
const headingEls: { id: string; el: HTMLElement }[] = []
let scrollRaf: number | undefined

function updateActiveId() {
  scrollRaf = undefined
  const id = resolveActiveHeading(
    headingEls.map(({ id, el }) => ({ id, top: el.getBoundingClientRect().top })),
    HEADING_OFFSET,
  )
  if (id !== activeId.value) activeId.value = id
}

function onScroll() {
  if (scrollRaf !== undefined) return // rAF 节流：一帧内最多结算一次
  scrollRaf = requestAnimationFrame(updateActiveId)
}

onMounted(() => {
  if (!props.items.length) return
  for (const item of props.items) {
    const el = document.getElementById(item.id)
    if (el) headingEls.push({ id: item.id, el }) // TOC 序 == 文档序
  }
  if (!headingEls.length) return
  updateActiveId()
  window.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('resize', onScroll, { passive: true })
})

onBeforeUnmount(() => {
  if (scrollRaf !== undefined) cancelAnimationFrame(scrollRaf)
  window.removeEventListener('scroll', onScroll)
  window.removeEventListener('resize', onScroll)
})

async function scrollToHeading(id: string) {
  activeId.value = id
  const element = document.getElementById(id)
  if (!element) return
  await nextTick()
  const top = element.getBoundingClientRect().top + window.scrollY - 88
  window.scrollTo({ top: Math.max(0, top), behavior: 'smooth' })
}
</script>

<template>
  <nav v-if="items.length" ref="rootEl" class="px-4 py-4" aria-label="Wiki table of contents">
    <h2 class="text-sm font-semibold text-base-content/55">{{ t('wiki.tocTitle') }}</h2>
    <ul class="mt-3 space-y-0.5 border-l border-line">
      <li v-for="item in items" :key="item.id">
        <button
          type="button"
          class="block w-full truncate rounded-r-md py-1 pr-2 text-left text-[13px] leading-5 transition-colors hover:bg-base-200 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          :class="[
            item.level === 1 ? 'pl-3 font-semibold' : item.level === 2 ? 'pl-5' : 'pl-7',
            activeId === item.id ? 'bg-info/10 text-primary' : 'text-base-content/65',
          ]"
          :title="item.text"
          @click="scrollToHeading(item.id)"
        >
          {{ item.text }}
        </button>
      </li>
    </ul>
  </nav>
</template>
