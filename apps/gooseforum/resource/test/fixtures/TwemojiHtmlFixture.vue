<template>
  <div v-html="html" />
</template>

<script setup lang="ts">
defineProps<{ html: string }>()
</script>
