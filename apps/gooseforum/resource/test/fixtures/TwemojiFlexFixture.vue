<template>
  <!-- 复刻 AppShell 页脚版权行：插值的直接父元素是 inline-flex 容器。
       emoji 被替换成 img 后，img 与文本不得成为两个独立 flex item。 -->
  <footer>
    <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-0.5">
      <span class="footer-primary inline-flex min-h-5 items-center rounded">{{ primary }}</span>
    </div>
  </footer>
</template>

<script setup lang="ts">
defineProps<{ primary: string }>()
</script>
