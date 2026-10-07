<script setup lang="ts">
import {useI18n} from 'vue-i18n'
import TopicList from '@/site/components/TopicList.vue'
import type {AnonymousProfileProps} from '@gooseforum/client'
const page=defineProps<{props:AnonymousProfileProps}>()
const {t}=useI18n()
</script>
<template>
 <main class="mx-auto max-w-4xl space-y-5 p-4">
  <header class="flex min-w-0 items-center gap-4"><img :src="page.props.persona.avatarUrl" :alt="page.props.persona.name" class="h-20 w-20 rounded-full" /><div class="min-w-0"><h1 class="break-all text-2xl font-bold">{{page.props.persona.name}}</h1><p>{{t('anonymous.identity')}}</p></div></header>
  <p>{{t('anonymous.profileCounts',{topics:page.props.topicCount,replies:page.props.replyCount})}}</p>
  <TopicList :topics="page.props.topics" />
  <ul class="divide-y divide-line"><li v-for="reply in page.props.replies" :key="reply.id" class="py-3"><a :href="reply.url">{{reply.excerpt}}</a></li></ul>
  <a v-if="page.props.hasNext" :href="`${page.props.persona.profileUrl}?page=${page.props.page+1}`" class="gf-button gf-button-secondary">{{t('anonymous.next')}}</a>
 </main>
</template>
