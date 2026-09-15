<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Activity, ArrowUpRight, SunMoon } from '@lucide/vue'
import StatusPage from './StatusPage.vue'
const { t, locale } = useI18n()
function theme() { document.documentElement.dataset.theme = document.documentElement.dataset.theme === 'gf-dark' ? 'gf-light' : 'gf-dark' }
function language() { document.documentElement.lang = locale.value }
</script>
<template>
  <div class="site-shell">
    <nav class="site-nav" aria-label="YourTJ"><a href="/" class="brand"><Activity :size="20" /><span>yourtj<span class="brand-note"> / status</span></span></a><div class="site-tools"><select v-model="locale" aria-label="Language" @change="language"><option value="zh">中文</option><option value="en">EN</option><option value="ja">日本語</option><option value="de">DE</option></select><button :aria-label="t('status.theme')" @click="theme"><SunMoon :size="18" /></button><a class="community-link" href="https://f.yourtj.de">{{ t('status.community') }}<ArrowUpRight :size="14" /></a></div></nav>
    <main><StatusPage /></main>
  </div>
</template>
