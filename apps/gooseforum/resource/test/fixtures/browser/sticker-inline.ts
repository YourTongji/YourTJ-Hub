import { createApp, h, ref } from 'vue'
import MarkdownImageViewer from '../../../src/components/MarkdownImageViewer.vue'
import StickerCard from '../../../src/site/components/StickerCard.vue'
import { i18n } from '../../../src/runtime/i18n'
import { getMarkdownImagePreview, getMarkdownSticker } from '../../../src/runtime/markdown-image-preview'
import { twemojiVuePlugin } from '../../../src/runtime/twemoji'
import '../../../src/styles/resource.css'

document.documentElement.dataset.theme = new URLSearchParams(location.search).get('theme') ?? 'gf-light'

const contentHtml = `
  <p id="plain">正文没有贴纸的一行文字 Plain text line</p>
  <p id="mixed">今天的课真有意思<img id="sticker" src="/sticker.png" alt="sticker:smile" data-gf-sticker="smile">下课见<img id="sticker-2" src="/sticker.png" alt="sticker:smile" data-gf-sticker="smile"></p>
  <p id="alone"><img id="sticker-alone" src="/sticker.png" alt="sticker:smile" data-gf-sticker="smile"></p>
  <h2 id="heading">标题<img id="sticker-heading" src="/sticker.png" alt="sticker:smile" data-gf-sticker="smile"></h2>
  <p id="photo-line"><img id="photo" src="/photo.png" alt="Photo attachment" width="96" height="48"></p>
`

createApp({
  setup() {
    const viewer = ref<InstanceType<typeof MarkdownImageViewer> | null>(null)
    const card = ref<InstanceType<typeof StickerCard> | null>(null)

    function handleClick(event: MouseEvent) {
      const sticker = getMarkdownSticker(event.target)
      if (sticker) {
        event.preventDefault()
        card.value?.open(sticker)
        return
      }
      const preview = getMarkdownImagePreview(event.target)
      if (!preview) return
      event.preventDefault()
      viewer.value?.open(preview.images, preview.index)
    }

    return () => h('main', { class: 'p-4' }, [
      h('div', { id: 'content', class: 'gf-prose gf-prose-post', innerHTML: contentHtml, onClick: handleClick }),
      h(MarkdownImageViewer, { ref: viewer }),
      h(StickerCard, { ref: card, signedIn: false }),
    ])
  },
})
  .use(i18n)
  .use(twemojiVuePlugin)
  .mount('#app')
