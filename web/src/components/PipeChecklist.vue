<script setup lang="ts">
import { computed, reactive } from 'vue'

import type { PipeEntry } from '../types'

const props = defineProps<{
  pipes: PipeEntry[]
}>()

// 核查勾选状态：设计师逐根确认音管，纯前端复核辅助。
const checked = reactive(new Set<string>())

function toggle(pipeId: string) {
  if (checked.has(pipeId)) checked.delete(pipeId)
  else checked.add(pipeId)
}

const doneCount = computed(() => props.pipes.filter((p) => checked.has(p.pipeId)).length)

function pathOf(p: PipeEntry): string {
  const s = p.source
  const head = `${s.startKeyboard} · 键 ${s.startKey}`
  return s.edges === 0 ? `${head}（直按）` : `${head} → ${s.couplers.join(' → ')}`
}
</script>

<template>
  <div>
    <p class="hint">已确认 {{ doneCount }} / {{ props.pipes.length }} 根物理音管（同一音管只计一次，来源为最优路径）。</p>
    <table class="checklist">
      <thead>
        <tr>
          <th>核查</th>
          <th>物理音管</th>
          <th>音栓</th>
          <th>发声位置</th>
          <th>边数</th>
          <th>来源路径</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="p in props.pipes" :key="p.pipeId" :class="{ done: checked.has(p.pipeId) }">
          <td>
            <input type="checkbox" :checked="checked.has(p.pipeId)" @change="toggle(p.pipeId)" />
          </td>
          <td class="mono">{{ p.pipeId }}</td>
          <td class="mono">{{ p.source.stop }}</td>
          <td class="mono">{{ p.source.keyboard }} · 键 {{ p.source.key }}</td>
          <td class="mono">{{ p.source.edges }}</td>
          <td class="mono">{{ pathOf(p) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
