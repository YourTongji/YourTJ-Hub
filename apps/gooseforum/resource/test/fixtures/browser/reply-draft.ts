import { h, onMounted, ref } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import PostStream from '../../../src/site/components/PostStream.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { ViewerPayload } from '@gooseforum/client'

await setLocale('zh')
const app = createApp({
  setup() {
    const stream = ref<InstanceType<typeof PostStream>>()
    onMounted(() => stream.value?.openFloatingPostComposer())
    return () => h(PostStream, {
      ref: stream, topicId: 900, topicTitle: 'Reply draft recovery', contentType: 0,
      viewer: { id: 7, username: 'writer', isAuthenticated: true } as ViewerPayload,
      canPost: true, initialPostStream: { posts: [], hasBefore: false, hasAfter: false },
    })
  },
}).use(i18n)
for (const name of ['code-copy', 'code-highlight', 'math-render', 'content-enhancements']) app.directive(name, {})
app.mount('#app')
