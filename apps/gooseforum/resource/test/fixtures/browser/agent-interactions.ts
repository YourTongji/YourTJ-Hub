import {createApp,h} from 'vue'
import AgentDialog from '../../../src/admin/pages/management/AgentWebhookManagementDialog.vue'
import MentionCandidates from '../../../src/site/components/MentionCandidates.vue'
import {i18n,setLocale} from '../../../src/runtime/i18n'
const params=new URLSearchParams(location.search)
if(params.has('mention')) await import('../../../src/styles/resource.css')
else await import('../../../src/admin/styles/admin.css')
await setLocale((params.get('lang')||'en') as 'en')
const agent={agentId:8,username:'helper-agent',nickname:'Forum assistant',avatarUrl:'',tokenPrefix:'agt_',enabled:1,createdAt:0,updatedAt:0,configVersion:5,eventsEnabled:true,eventTypes:['agent.mentioned'],subscriptionGeneration:2,webhookEnabled:true,webhookEndpoint:'https://receiver.example/events',endpointGeneration:3,secretConfigured:true,secretVersion:1,pendingCount:2,latestAcceptedAt:'2026-10-04T12:00:00Z'}
createApp({render:()=>params.has('mention')?h('main',{style:'margin:24px 12px;max-width:480px'},[h('h1','Forum mention candidates'),h(MentionCandidates,{open:true,query:'helper',candidates:[{id:8,username:'helper-agent',nickname:'Forum assistant',avatarUrl:'data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 width=%2232%22 height=%2232%22%3E%3Crect width=%2232%22 height=%2232%22 fill=%22%239db4d8%22/%3E%3C/svg%3E',actorType:'bot'},{id:9,username:'classmate',nickname:'Classmate',avatarUrl:'data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 width=%2232%22 height=%2232%22%3E%3Crect width=%2232%22 height=%2232%22 fill=%22%239db4d8%22/%3E%3C/svg%3E',actorType:'human'}],activeIndex:0,loading:false,failed:false,docked:true,panelStyle:{position:'relative',width:'100%'},onSelect:()=>{document.documentElement.dataset.selected='true'}})]):h(AgentDialog,{open:true,agent})}).use(i18n).mount('#app')
document.documentElement.dataset.ready='true'
