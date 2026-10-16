import { createApp, h, onMounted, ref } from 'vue'
import { renderTwemojiHtml, twemojiVuePlugin } from '../../../src/runtime/twemoji'

const copyFixture = `
  <div><p>段落一 🐦‍🔥</p><blockquote><p>引用 ❤️</p></blockquote></div>
  <ul><li>列表一 🔥</li><li><strong>列表二 🇨🇳</strong></li></ul>
  <table><tbody><tr><td>表格甲</td><td>表格乙 👩🏽‍💻</td></tr></tbody></table>
`

createApp({
  setup() {
    const root = ref<HTMLElement>()
    onMounted(() => {
      if (root.value) renderTwemojiHtml(root.value)
    })
    return () => h('main', { id: 'copy-root', ref: root, innerHTML: copyFixture })
  },
})
  .use(twemojiVuePlugin)
  .mount('#app')
