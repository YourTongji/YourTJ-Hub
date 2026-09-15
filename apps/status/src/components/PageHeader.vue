<script setup lang="ts">
defineProps<{ title: string; description?: string; compact?: boolean }>()
</script>
<template><header class="gf-page-header"><div><h1>{{ title }}</h1><p v-if="description">{{ description }}</p></div><div><slot name="actions" /></div></header></template>
<style scoped>
.gf-page-header { display:flex; align-items:center; justify-content:space-between; gap:16px; margin-bottom:12px; }
h1 { font-size:24px; line-height:1.3; margin:0; font-weight:700; } p { font-size:14px; line-height:1.7; color:var(--gf-color-icon-muted); margin:4px 0 0; }
@media(max-width:640px) { .gf-page-header { align-items:flex-start; flex-wrap:wrap; } h1 { font-size:20px; } }
</style>
