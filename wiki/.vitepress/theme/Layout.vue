<script setup lang="ts">
import { nextTick, onMounted, watch } from 'vue'
import { useRoute } from 'vitepress'
import DefaultTheme from 'vitepress/theme'
import { init } from '@waline/client'
import '@waline/client/style'

const route = useRoute()

// 评论服务地址通过构建环境变量注入；未配置（默认）时整站不渲染评论。
// 部署示例：
//   VITE_WALINE_SERVER_URL=https://comment.example.com
const serverURL = (import.meta.env.VITE_WALINE_SERVER_URL as string | undefined)?.trim()

let waline: ReturnType<typeof init> | null = null

function mountWaline() {
  if (!serverURL) return
  const el = document.querySelector('#waline-comments')
  if (!el) return
  waline = init({
    el,
    serverURL,
    path: route.path, // 按路由分 key，每页独立评论
    lang: 'zh-CN',
    locale: {
      placeholder: '登录后发表评论……',
    },
  })
}

function destroyWaline() {
  if (waline) {
    waline.destroy()
    waline = null
  }
}

onMounted(() => {
  mountWaline()
})

// 路由切换后重建 Waline（按新路径加载对应评论）
watch(() => route.path, () => {
  destroyWaline()
  nextTick(mountWaline)
})
</script>

<template>
  <DefaultTheme.Layout>
    <template #doc-after>
      <div
        v-if="serverURL"
        id="waline-comments"
        class="waline-wrapper"
      />
    </template>
  </DefaultTheme.Layout>
</template>

<style>
.waline-wrapper {
  margin-top: 3rem;
  padding-top: 2rem;
  border-top: 1px solid var(--vp-c-divider);
}
</style>
