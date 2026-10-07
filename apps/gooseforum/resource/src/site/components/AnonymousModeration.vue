<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { governIdentity, revealIdentity } from '@/runtime/anonymous-identity'
const props=defineProps<{postId:number;publicUid:string;canReveal:boolean}>()
const {t}=useI18n()
const open=ref(false),busy=ref(false),error=ref(''),reason=ref('')
const owner=ref<{userId:number;username:string}>()
function close(){open.value=false;owner.value=undefined;reason.value='';error.value=''}
async function act(action:'ban'|'restore'|'reveal'){
 if(!reason.value.trim()||busy.value)return
 busy.value=true;error.value=''
 try {if(action==='reveal'){owner.value=await revealIdentity(props.publicUid,reason.value)}else{await governIdentity(props.postId,action==='ban',reason.value);close()}}catch(e){error.value=e instanceof Error?e.message:String(e)}finally{busy.value=false}
}
</script>
<template>
 <button class="text-xs text-primary underline" @click="open=true">{{t('anonymous.manage')}}</button>
 <div v-if="open" class="my-3 rounded-box border border-line p-3" :aria-busy="busy">
  <label class="block">{{t('anonymous.reason')}}<textarea v-model="reason" maxlength="512" class="textarea w-full" /></label>
  <p v-if="error" role="alert" class="text-error">{{error}}</p>
  <p v-if="owner" class="break-all">{{owner.username}} · {{owner.userId}}</p>
  <div class="flex flex-wrap gap-2">
   <button :disabled="busy||!reason.trim()" class="gf-button gf-button-secondary" @click="act('ban')">{{t('anonymous.ban')}}</button>
   <button :disabled="busy||!reason.trim()" class="gf-button gf-button-secondary" @click="act('restore')">{{t('anonymous.restore')}}</button>
   <button v-if="canReveal" :disabled="busy||!reason.trim()" class="gf-button gf-button-secondary" @click="act('reveal')">{{t('anonymous.reveal')}}</button>
   <button :disabled="busy" class="gf-button gf-button-secondary" @click="close">{{t('anonymous.cancel')}}</button>
  </div>
 </div>
</template>
