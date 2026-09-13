<script setup lang="ts">
import { ref } from 'vue'

import type { FieldError } from '../types'

const props = defineProps<{
  modelValue: string
  errors: FieldError[]
  parseError: string
}>()

const emit = defineEmits<{
  'update:modelValue': [text: string]
  apply: []
}>()

const fileInput = ref<HTMLInputElement>()

const EMPTY_TEMPLATE = `{
  "keyboards": [],
  "stops": [],
  "couplers": []
}`

function onInput(e: Event) {
  emit('update:modelValue', (e.target as HTMLTextAreaElement).value)
}

function pickFile() {
  fileInput.value?.click()
}

async function onFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  const text = await file.text()
  emit('update:modelValue', text)
  emit('apply')
  input.value = ''
}

function insertTemplate() {
  emit('update:modelValue', EMPTY_TEMPLATE)
  emit('apply')
}
</script>

<template>
  <div>
    <textarea
      :value="props.modelValue"
      spellcheck="false"
      placeholder='在此粘贴配置 JSON，或点击“导入 JSON 文件”。结构：{"keyboards":[…],"stops":[…],"couplers":[…]}'
      @input="onInput"
    />
    <div class="toolbar">
      <button type="button" @click="pickFile">导入 JSON 文件</button>
      <button type="button" @click="insertTemplate">插入空白模板</button>
      <button type="button" class="primary" @click="emit('apply')">应用配置</button>
      <input ref="fileInput" type="file" accept=".json,application/json" hidden @change="onFile" />
    </div>
    <ul v-if="props.parseError" class="error-list">
      <li><code>JSON</code>{{ props.parseError }}</li>
    </ul>
    <ul v-if="props.errors.length" class="error-list">
      <li v-for="(err, i) in props.errors" :key="i">
        <code>{{ err.path || '(请求)' }}</code>{{ err.message }}
      </li>
    </ul>
  </div>
</template>
