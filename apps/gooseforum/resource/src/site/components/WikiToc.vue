<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { WikiTocItem } from '@gooseforum/client'

const props = defineProps<{
  items: WikiTocItem[]
}>()

const { t } = useI18n()
const activeId = ref('')
const rootEl = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | undefined

onMounted(() => {
  if (!('IntersectionObserver' in window) || !props.items.length) return

  const headingIds = props.items
    .map((item) => item.id)
    .filter((id) => document.getElementById(id))

  if (!headingIds.length) return

  observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          activeId.value = entry.target.id
        }
      }
    },
    { rootMargin: '-80px 0px -65% 0px', threshold: 0 },
  )

  headingIds.forEach((id) => {
    const element = document.getElementById(id)
    if (element) observer?.observe(element)
  })
})

onBeforeUnmount(() => {
  observer?.disconnect()
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
