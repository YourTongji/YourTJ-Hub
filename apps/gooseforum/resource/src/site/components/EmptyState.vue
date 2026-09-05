<script setup lang="ts">
import type { Component } from 'vue'
import LogoLoader from '@/site/components/LogoLoader.vue'

defineProps<{
  icon?: Component
  title: string
  description?: string
  loading?: boolean
}>()
</script>

<template>
  <div class="flex min-h-56 flex-col items-center justify-center px-6 text-center">
    <!-- 加载态：品牌 logo 动画替代图标旋转；空态/错误态仍用传入图标 -->
    <LogoLoader v-if="loading" :size="40" />
    <component
      :is="icon"
      v-else-if="icon"
      class="h-8 w-8 text-base-content/35"
    />
    <h2 class="mt-2 text-base font-semibold text-base-content">{{ title }}</h2>
    <p v-if="description" class="mt-1 text-sm text-base-content/55">{{ description }}</p>
    <div v-if="$slots.default" class="mt-4">
      <slot />
    </div>
  </div>
</template>
