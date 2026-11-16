<script setup lang="ts">
/**
 * 选中文字时浮在选区上方的快捷格式条（类飞书文档）：正文/标题切换、加粗、斜体、删除线、
 * 行内代码、链接、引用。命令交给 Vditor 自己的工具栏执行（同一套快捷键、撤销与序列化），
 * 这里只负责呈现与定位。按钮按下不夺焦点，选区保持不变。
 */
import { computed, nextTick, ref, watch } from 'vue'
import { Bold, Check, ChevronDown, Code, Italic, Link2, Quote, Strikethrough } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { placeFloating, type FloatingRect } from '@/runtime/floating-position'

export type SelectionCommand = 'bold' | 'italic' | 'strike' | 'inline-code' | 'link' | 'quote'
export type HeadingTag = 'h1' | 'h2' | 'h3'

const props = defineProps<{
  open: boolean
  host: HTMLElement | null
  anchor: FloatingRect | null
  boundary: FloatingRect | null
  /** 当前编辑器工具栏里存在的命令（移动端精简工具栏没有的不显示） */
  available: readonly SelectionCommand[]
  active: readonly SelectionCommand[]
  /** 当前块的标题级别；0 为正文，null 表示不可切换（如表格内） */
  heading: number | null
  /** 触控设备：系统选区菜单在上方，工具条放到选区下方 */
  preferBelow?: boolean
}>()
const emit = defineEmits<{ command: [command: SelectionCommand]; heading: [tag: HeadingTag | null] }>()

const { t } = useI18n()
const panel = ref<HTMLElement | null>(null)
const headingMenuOpen = ref(false)

const ITEMS: Record<SelectionCommand, { icon: typeof Bold; label: string }> = {
  bold: { icon: Bold, label: 'editor.menu.bold' },
  italic: { icon: Italic, label: 'editor.menu.italic' },
  strike: { icon: Strikethrough, label: 'editor.menu.strike' },
  'inline-code': { icon: Code, label: 'editor.menu.inlineCode' },
  link: { icon: Link2, label: 'editor.menu.link' },
  quote: { icon: Quote, label: 'editor.menu.quote' },
}
const INLINE: SelectionCommand[] = ['bold', 'italic', 'strike', 'inline-code', 'link']
const inlineItems = computed(() => INLINE.filter((type) => props.available.includes(type)))
const showQuote = computed(() => props.available.includes('quote'))
const headingOptions = computed(() => [
  { tag: null, level: 0, label: t('editor.floating.paragraph') },
  { tag: 'h1' as const, level: 1, label: t('editor.floating.heading', { level: 1 }) },
  { tag: 'h2' as const, level: 2, label: t('editor.floating.heading', { level: 2 }) },
  { tag: 'h3' as const, level: 3, label: t('editor.floating.heading', { level: 3 }) },
])
const headingLabel = computed(() => (props.heading ? `H${props.heading}` : t('editor.floating.paragraphShort')))

function reposition() {
  if (!panel.value || !props.anchor || !props.boundary) return
  placeFloating(panel.value, props.anchor, props.boundary, { side: props.preferBelow ? 'bottom' : 'top', gap: 10 })
}

watch(
  () => [props.open, props.anchor] as const,
  ([open]) => {
    if (!open) headingMenuOpen.value = false
    void nextTick(reposition)
  },
)

function run(command: SelectionCommand) {
  headingMenuOpen.value = false
  emit('command', command)
}

function chooseHeading(tag: HeadingTag | null) {
  headingMenuOpen.value = false
  emit('heading', tag)
}

defineExpose({ reposition, contains: (node: Node | null) => Boolean(node && panel.value?.contains(node)) })
</script>

<template>
  <Teleport v-if="host" :to="host">
    <Transition name="gf-editor-float">
      <div
        v-if="open && anchor"
        ref="panel"
        role="toolbar"
        :aria-label="t('editor.floating.selectionToolbar')"
        class="gf-editor-float gf-floating-surface"
        data-test="editor-selection-toolbar"
        @pointerdown.prevent
        @mousedown.prevent
      >
        <div v-if="heading !== null" class="relative">
          <button
            type="button"
            class="gf-editor-float-button gf-editor-float-button--text"
            :class="{ 'is-active': headingMenuOpen }"
            :aria-label="t('editor.floating.textStyle')"
            aria-haspopup="menu"
            :aria-expanded="headingMenuOpen"
            @click="headingMenuOpen = !headingMenuOpen"
          >
            <span class="min-w-[1.75rem] text-center tabular-nums">{{ headingLabel }}</span>
            <ChevronDown class="h-3.5 w-3.5 opacity-60 transition-transform duration-150 ease-out" :class="{ 'rotate-180': headingMenuOpen }" />
          </button>
          <Transition name="gf-editor-float">
            <div v-if="headingMenuOpen" role="menu" class="gf-editor-float-menu gf-floating-surface">
              <button
                v-for="option in headingOptions"
                :key="option.level"
                type="button"
                role="menuitemradio"
                :aria-checked="heading === option.level"
                class="gf-editor-float-menu-item"
                :class="`gf-editor-float-menu-item--h${option.level}`"
                @click="chooseHeading(option.tag)"
              >
                <span class="flex-1 truncate">{{ option.label }}</span>
                <Check v-if="heading === option.level" class="h-3.5 w-3.5 text-primary" />
              </button>
            </div>
          </Transition>
        </div>
        <span v-if="heading !== null && inlineItems.length" class="gf-editor-float-divider" aria-hidden="true" />
        <button
          v-for="type in inlineItems"
          :key="type"
          type="button"
          class="gf-editor-float-button"
          :class="{ 'is-active': active.includes(type) }"
          :aria-label="t(ITEMS[type].label)"
          :aria-pressed="active.includes(type)"
          :title="t(ITEMS[type].label)"
          :data-command="type"
          @click="run(type)"
        >
          <component :is="ITEMS[type].icon" class="h-4 w-4" :stroke-width="2.25" />
        </button>
        <template v-if="showQuote">
          <span class="gf-editor-float-divider" aria-hidden="true" />
          <button
            type="button"
            class="gf-editor-float-button"
            :class="{ 'is-active': active.includes('quote') }"
            :aria-label="t(ITEMS.quote.label)"
            :aria-pressed="active.includes('quote')"
            :title="t(ITEMS.quote.label)"
            data-command="quote"
            @click="run('quote')"
          >
            <Quote class="h-4 w-4" :stroke-width="2.25" />
          </button>
        </template>
      </div>
    </Transition>
  </Teleport>
</template>
