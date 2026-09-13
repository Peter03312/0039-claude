<script setup lang="ts">
import { onMounted, ref } from 'vue'

import { api } from '../api'
import type { Snapshot, SnapshotSummary } from '../types'

const emit = defineEmits<{
  load: [snap: Snapshot]
}>()

const list = ref<SnapshotSummary[]>([])
const error = ref('')
const busy = ref(false)

async function refresh() {
  error.value = ''
  try {
    list.value = await api.listSnapshots()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  }
}

async function load(id: string) {
  busy.value = true
  error.value = ''
  try {
    emit('load', await api.getSnapshot(id))
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}

async function remove(id: string) {
  busy.value = true
  error.value = ''
  try {
    await api.deleteSnapshot(id)
    await refresh()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}

function fmtTime(iso: string): string {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

onMounted(refresh)
defineExpose({ refresh })
</script>

<template>
  <div>
    <h2>④ 已采纳快照</h2>
    <ul v-if="error" class="error-list"><li>{{ error }}</li></ul>
    <p v-if="!list.length && !error" class="hint">暂无快照。核查通过后在结果区点击“采纳并保存快照”。</p>
    <div v-for="s in list" :key="s.id" class="snapshot-item">
      <div>
        <div>{{ s.name }}</div>
        <div class="meta">{{ fmtTime(s.createdAt) }} · {{ s.pipes }} 音管 / {{ s.states }} 状态 / {{ s.pruned }} 裁剪</div>
      </div>
      <div class="actions">
        <button type="button" class="small" :disabled="busy" @click="load(s.id)">载入</button>
        <button type="button" class="small" :disabled="busy" @click="remove(s.id)">删除</button>
      </div>
    </div>
  </div>
</template>
