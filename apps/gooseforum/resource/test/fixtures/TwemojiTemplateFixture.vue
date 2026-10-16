<template>
  <div>
    <button>😀</button>
    <span>Mixed {{ text }}</span>
    <code>🐦‍🔥</code>
  </div>
</template>

<script setup lang="ts">
defineProps<{ text: string }>()
</script>
