<template>
  <div>
    <p class="ordinary">ordinary 中文 🐦‍🔥 mixed 😀 text</p>
    <div class="editable-boolean" contenteditable>
      editable 🐦‍🔥
      <span>nested 🧑🏽‍💻</span>
      <span contenteditable="false">non-editable island 🔥</span>
    </div>
    <div class="editable-empty" contenteditable="">
      empty 🐦‍🔥 <span>nested 🇨🇳</span>
    </div>
    <div class="editable-true" contenteditable="true">
      true 🐦‍🔥 <span>nested ❤️</span>
    </div>
    <div class="editable-false" contenteditable="false">false 🐦‍🔥</div>
    <div class="editable-dynamic" :contenteditable="editable">
      dynamic 🐦‍🔥 <span>nested ✨</span>
    </div>
    <div class="editable-dynamic-argument" :[editableAttribute]="editableValue">
      dynamic argument 🐦‍🔥 <span>nested 🌟</span>
    </div>
    <div class="editable-object-binding" v-bind="editableAttrs">
      object binding 🐦‍🔥 <span>nested 🎉</span>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  editable: boolean
  editableAttribute: string
  editableValue: boolean
  editableAttrs: Record<string, unknown>
}>()
</script>
