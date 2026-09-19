<script setup lang="ts">
defineProps<{ title: string; description?: string; compact?: boolean }>()
</script>
<template><header class="gf-page-header"><div class="gf-page-header-copy"><h1>{{ title }}</h1><p v-if="description">{{ description }}</p></div><div class="gf-page-header-actions"><slot name="actions" /></div></header></template>
<style scoped>
.gf-page-header { display:flex; align-items:center; justify-content:space-between; gap:16px; margin-bottom:12px; }
.gf-page-header-copy { min-width:0; }
.gf-page-header-actions { display:flex; align-items:center; flex-shrink:0; }
h1 { font-size:24px; line-height:1.3; margin:0; font-weight:700; } p { font-size:14px; line-height:1.7; color:var(--gf-color-icon-muted); margin:4px 0 0; }
@media(max-width:640px) { .gf-page-header { display:grid; grid-template-columns:minmax(0, 1fr) auto; align-items:center; column-gap:12px; row-gap:4px; } .gf-page-header-copy { display:contents; } .gf-page-header h1 { grid-column:1; grid-row:1; font-size:20px; } .gf-page-header p { grid-column:1 / -1; grid-row:2; } .gf-page-header-actions { grid-column:2; grid-row:1; } }
</style>
