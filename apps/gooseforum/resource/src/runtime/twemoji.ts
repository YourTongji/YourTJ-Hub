import {
  Fragment,
  defineComponent,
  h,
  ref,
  toDisplayString,
  type App,
  type Directive,
} from 'vue'
import {
  isContentEditableValue,
  TWEMOJI_TEXT_FREE_TAGS,
} from './twemoji-editability'
import manifestData from '../../../../mobile/packages/ui_kit/assets/twemoji-manifest.json'

type TwemojiManifest = {
  version: string
  entries: Array<{ sequence: string; asset: string }>
}

export type TwemojiSegment =
  | { type: 'text'; value: string }
  | { type: 'emoji'; value: string; asset: string }

const manifest = manifestData as TwemojiManifest
const assetBySequence = new Map(
  manifest.entries.map(({ sequence, asset }) => [sequence, asset]),
)
const maximumCodepoints = Math.max(
  ...manifest.entries.map(({ sequence }) => Array.from(sequence).length),
)
const variationSelector = '\uFE0F'
const zeroWidthJoiner = '\u200D'
const isEmojiExtension = (codepoint: string) => {
  const value = codepoint.codePointAt(0) ?? 0
  return (
    value === 0xfe0e ||
    value === 0xfe0f ||
    value === 0x20e3 ||
    (value >= 0x1f3fb && value <= 0x1f3ff) ||
    (value >= 0xe0020 && value <= 0xe007f)
  )
}

function consumeExtensions(codepoints: string[], start: number): number {
  let end = start
  while (end < codepoints.length && isEmojiExtension(codepoints[end])) end += 1
  return end
}

function consumeJoinedSequence(codepoints: string[], start: number): number {
  let end = consumeExtensions(codepoints, start + 1)
  let hasJoiner = false
  while (end < codepoints.length && codepoints[end] === zeroWidthJoiner) {
    hasJoiner = true
    end += 1
    if (end < codepoints.length) end += 1
    end = consumeExtensions(codepoints, end)
  }
  return hasJoiner ? end : start + 1
}

export const TWEMOJI_VERSION = manifest.version

/** Splits by the longest complete sequence in the pinned Twemoji asset manifest. */
export function parseTwemoji(text: string): TwemojiSegment[] {
  const codepoints = Array.from(text)
  const segments: TwemojiSegment[] = []
  let plainText = ''
  let index = 0

  while (index < codepoints.length) {
    let bestEnd = -1
    let bestAsset: string | undefined
    let candidate = ''
    let hasJoiner = false

    for (
      let end = index;
      end < codepoints.length && end - index < maximumCodepoints + 4;
      end += 1
    ) {
      candidate += codepoints[end]
      hasJoiner ||= codepoints[end] === zeroWidthJoiner
      const key = hasJoiner
        ? candidate
        : candidate.replaceAll(variationSelector, '')
      const asset = assetBySequence.get(key)
      if (asset) {
        bestEnd = end + 1
        bestAsset = asset
      }
    }

    if (bestAsset && bestEnd > index) {
      const extendedEnd = consumeExtensions(codepoints, bestEnd)
      if (codepoints[extendedEnd] === zeroWidthJoiner) {
        const unsupportedEnd = consumeJoinedSequence(codepoints, index)
        plainText += codepoints.slice(index, unsupportedEnd).join('')
        index = unsupportedEnd
        continue
      }
      if (extendedEnd > bestEnd) {
        const extended = codepoints.slice(index, extendedEnd).join('')
        const extendedKey = extended.includes(zeroWidthJoiner)
          ? extended
          : extended.replaceAll(variationSelector, '')
        const extendedAsset = assetBySequence.get(extendedKey)
        if (!extendedAsset) {
          plainText += codepoints.slice(index, extendedEnd).join('')
          index = extendedEnd
          continue
        }
        bestEnd = extendedEnd
        bestAsset = extendedAsset
      }
    } else {
      const joinedEnd = consumeJoinedSequence(codepoints, index)
      if (joinedEnd > index + 1) {
        plainText += codepoints.slice(index, joinedEnd).join('')
        index = joinedEnd
        continue
      }
    }

    if (bestAsset && bestEnd > index) {
      if (plainText) segments.push({ type: 'text', value: plainText })
      const value = codepoints.slice(index, bestEnd).join('')
      segments.push({ type: 'emoji', value, asset: bestAsset })
      plainText = ''
      index = bestEnd
    } else {
      plainText += codepoints[index]
      index += 1
    }
  }

  if (plainText) segments.push({ type: 'text', value: plainText })
  return segments
}

const assetBase = `${import.meta.env.BASE_URL}twemoji/72x72/`

export const TwemojiText = defineComponent({
  name: 'TwemojiText',
  props: {
    text: { type: null, default: '' },
  },
  setup(props) {
    const failedAssets = ref<Set<string>>(new Set())

    return () => {
      const text = toDisplayString(props.text)
      const segments = parseTwemoji(text)

      return h(
        Fragment,
        null,
        segments.map((segment, index) => {
          if (segment.type === 'text' || failedAssets.value.has(segment.asset)) {
            return segment.type === 'text' ? segment.value : segment.value
          }

          return h('img', {
            key: `${segment.asset}-${index}`,
            class: 'twemoji',
            style: {
              display: 'inline-block',
              width: '1em',
              height: '1em',
              margin: '0 0.05em 0 0.1em',
              verticalAlign: '-0.1em',
              objectFit: 'contain',
            },
            src: `${assetBase}${segment.asset}`,
            alt: segment.value,
            draggable: false,
            loading: 'lazy',
            decoding: 'async',
            onError: () => {
              failedAssets.value = new Set(failedAssets.value).add(segment.asset)
            },
          })
        }),
      )
    }
  },
})

function nodeIsTwemojiExcluded(node: Node): boolean {
  let current: Element | null =
    node instanceof Element ? node : node.parentElement
  while (current) {
    if (
      TWEMOJI_TEXT_FREE_TAGS.has(current.localName.toLowerCase()) ||
      current.hasAttribute('data-twemoji-ignore') ||
      isContentEditableValue(current.getAttribute('contenteditable'))
    ) {
      return true
    }
    current = current.parentElement
  }
  return false
}

function replaceTextNodeWithTwemoji(node: Text): void {
  const segments = parseTwemoji(node.data)
  if (!segments.some((segment) => segment.type === 'emoji')) return

  const fragment = document.createDocumentFragment()
  for (const segment of segments) {
    if (segment.type === 'text') {
      fragment.append(document.createTextNode(segment.value))
      continue
    }

    const image = document.createElement('img')
    image.className = 'twemoji'
    image.style.cssText =
      'display:inline-block;width:1em;height:1em;margin:0 0.05em 0 0.1em;vertical-align:-0.1em;object-fit:contain'
    image.src = `${assetBase}${segment.asset}`
    image.alt = segment.value
    image.dataset.emoji = segment.value
    image.draggable = false
    image.loading = 'lazy'
    image.decoding = 'async'
    image.dataset.twemoji = 'true'
    image.addEventListener(
      'error',
      () => image.replaceWith(document.createTextNode(segment.value)),
      { once: true },
    )
    fragment.append(image)
  }
  node.replaceWith(fragment)
}

const twemojiHtmlDirective: Directive<HTMLElement> = {
  mounted(element) {
    renderTwemojiHtml(element)
  },
  updated(element) {
    renderTwemojiHtml(element)
  },
}

const lastRenderedHtml = new WeakMap<
  HTMLElement,
  { html: string; isSuppressed: boolean }
>()
const copyListeners = new WeakSet<Window>()

function htmlWithTwemojiFromFragment(fragment: DocumentFragment): string {
  const copy = fragment.cloneNode(true) as DocumentFragment
  for (const image of copy.querySelectorAll<HTMLImageElement>('img.twemoji')) {
    image.replaceWith(document.createTextNode(image.dataset.emoji ?? image.alt))
  }
  const container = document.createElement('div')
  container.append(copy)
  return container.innerHTML
}

function textWithTwemojiFromFragment(fragment: DocumentFragment): string {
  const copy = fragment.cloneNode(true) as DocumentFragment
  const replacements: Array<[string, string]> = []
  let tokenIndex = 0
  for (const image of copy.querySelectorAll<HTMLImageElement>('img.twemoji')) {
    let token: string
    do {
      token = `\uE000TWEMOJI_${tokenIndex++}_\uE001`
    } while (copy.textContent?.includes(token))
    replacements.push([token, image.dataset.emoji ?? image.alt])
    image.replaceWith(document.createTextNode(token))
  }

  // Let the browser apply its block, list, line-break and table-cell text
  // rules. A connected offscreen host gives innerText a layout context without
  // changing the live selection.
  const container = document.createElement('div')
  container.style.cssText =
    'position:fixed;left:-100000px;top:0;width:max-content;white-space:normal'
  container.append(copy)
  document.body.append(container)
  let text = container.innerText
  container.remove()
  for (const [token, emoji] of replacements) text = text.replaceAll(token, emoji)
  return text
}

function preserveTwemojiCopy(event: ClipboardEvent): void {
  const selection = window.getSelection()
  if (
    event.defaultPrevented ||
    !event.clipboardData ||
    !selection ||
    selection.isCollapsed ||
    selection.rangeCount === 0 ||
    (event.clipboardData.types?.length ?? 0) > 0
  ) {
    return
  }

  const fragments: DocumentFragment[] = []
  for (let index = 0; index < selection.rangeCount; index += 1) {
    const range = selection.getRangeAt(index)
    if (nodeIsTwemojiExcluded(range.commonAncestorContainer)) return
    fragments.push(range.cloneContents())
  }
  const selectedTwemoji = fragments.flatMap((fragment) =>
    Array.from(fragment.querySelectorAll<HTMLImageElement>('img.twemoji')),
  )
  if (
    selectedTwemoji.length === 0 ||
    selectedTwemoji.some(nodeIsTwemojiExcluded)
  ) {
    return
  }

  const copiedText = fragments.map(textWithTwemojiFromFragment).join('')
  const copiedHtml = fragments.map(htmlWithTwemojiFromFragment).join('')
  try {
    event.clipboardData.setData('text/plain', copiedText)
    event.clipboardData.setData('text/html', copiedHtml)
  } catch {
    return
  }
  event.preventDefault()
}

export function renderTwemojiHtml(element: HTMLElement): void {
  for (const image of element.querySelectorAll<HTMLImageElement>('img.twemoji')) {
    if (nodeIsTwemojiExcluded(image)) {
      image.replaceWith(document.createTextNode(image.dataset.emoji ?? image.alt))
    }
  }

  const currentHtml = element.innerHTML
  const isSuppressed = nodeIsTwemojiExcluded(element)
  const previous = lastRenderedHtml.get(element)
  if (previous?.html === currentHtml && previous.isSuppressed === isSuppressed) return

  const walker = document.createTreeWalker(element, NodeFilter.SHOW_TEXT)
  const textNodes: Text[] = []
  let node = walker.nextNode()
  while (node) {
    if (node instanceof Text && !nodeIsTwemojiExcluded(node)) {
      textNodes.push(node)
    }
    node = walker.nextNode()
  }

  for (const textNode of textNodes) replaceTextNodeWithTwemoji(textNode)
  lastRenderedHtml.set(element, {
    html: element.innerHTML,
    isSuppressed: nodeIsTwemojiExcluded(element),
  })
}

export const twemojiVuePlugin = {
  install(app: App) {
    app.component('TwemojiText', TwemojiText)
    app.directive('twemoji-html', twemojiHtmlDirective)
    if (typeof document !== 'undefined' && !copyListeners.has(window)) {
      // Window bubble runs after target, editor and document handlers, so an
      // existing clipboard implementation can prevent or populate the event.
      window.addEventListener('copy', preserveTwemojiCopy)
      copyListeners.add(window)
    }
  },
}
