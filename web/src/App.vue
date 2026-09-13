<script setup lang="ts">
import { computed, reactive, ref } from 'vue'

import { api, ApiError } from './api'
import ConfigEditor from './components/ConfigEditor.vue'
import PianoKeyboard from './components/PianoKeyboard.vue'
import PipeChecklist from './components/PipeChecklist.vue'
import PropagationGraph from './components/PropagationGraph.vue'
import SnapshotPanel from './components/SnapshotPanel.vue'
import type {
  Config,
  FieldError,
  PressedKey,
  Snapshot,
  VerifyRequest,
  VerifyResult,
} from './types'

const configText = ref('')
const config = ref<Config | null>(null)
const configParseError = ref('')
const serverErrors = ref<FieldError[]>([])

// 键盘 ID 是任意字符串，按键集合用 Map<键盘, Set<键号>> 存放，不做字符串拼接。
const pressedMap = reactive(new Map<string, Set<number>>())
const enabledStops = reactive(new Set<string>())

const activeKeyboard = ref('')
const result = ref<VerifyResult | null>(null)
const lastRequest = ref<VerifyRequest | null>(null)
const running = ref(false)

const snapshotName = ref('')
const adoptMsg = ref('')
const adoptErr = ref('')
const snapshotPanel = ref<InstanceType<typeof SnapshotPanel>>()

function applyConfig() {
  configParseError.value = ''
  serverErrors.value = []
  try {
    const raw: unknown = JSON.parse(configText.value)
    if (raw === null || typeof raw !== 'object' || Array.isArray(raw)) {
      throw new Error('配置必须是一个 JSON 对象')
    }
    const cfg = raw as Partial<Config>
    if (!Array.isArray(cfg.keyboards)) cfg.keyboards = []
    if (!Array.isArray(cfg.stops)) cfg.stops = []
    if (!Array.isArray(cfg.couplers)) cfg.couplers = []
    config.value = cfg as Config
    pressedMap.clear()
    enabledStops.clear()
    activeKeyboard.value = cfg.keyboards[0]?.id ?? ''
    result.value = null
    lastRequest.value = null
    adoptMsg.value = ''
  } catch (e) {
    config.value = null
    configParseError.value = e instanceof Error ? e.message : String(e)
  }
}

const keyboards = computed(() => config.value?.keyboards ?? [])
const activeKeyboardCfg = computed(() => keyboards.value.find((k) => k.id === activeKeyboard.value))

const pressedOfActive = computed(() => pressedMap.get(activeKeyboard.value) ?? new Set<number>())

const pressedCount = computed(() => {
  let n = 0
  for (const s of pressedMap.values()) n += s.size
  return n
})

const pressedChips = computed(() => {
  const out: { keyboard: string; key: number }[] = []
  for (const [kb, keys] of pressedMap) {
    for (const key of [...keys].sort((a, b) => a - b)) out.push({ keyboard: kb, key })
  }
  return out.sort((a, b) => (a.keyboard < b.keyboard ? -1 : a.keyboard > b.keyboard ? 1 : a.key - b.key))
})

const stopsByKeyboard = computed(() => {
  const m = new Map<string, { id: string; pipeCount: number }[]>()
  for (const st of config.value?.stops ?? []) {
    const arr = m.get(st.keyboard) ?? []
    arr.push({ id: st.id, pipeCount: Object.keys(st.pipes ?? {}).length })
    m.set(st.keyboard, arr)
  }
  for (const arr of m.values()) arr.sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0))
  return m
})

const couplerCount = computed(() => (config.value?.couplers ?? []).filter((c) => c.enabled !== false).length)

function toggleKey(kb: string, key: number) {
  let s = pressedMap.get(kb)
  if (!s) {
    s = new Set<number>()
    pressedMap.set(kb, s)
  }
  if (s.has(key)) s.delete(key)
  else s.add(key)
}

function toggleStop(id: string) {
  if (enabledStops.has(id)) enabledStops.delete(id)
  else enabledStops.add(id)
}

function collectPressed(): PressedKey[] {
  const out: PressedKey[] = []
  for (const [kb, keys] of pressedMap) {
    for (const key of keys) out.push({ keyboard: kb, key })
  }
  return out.sort((a, b) => (a.keyboard < b.keyboard ? -1 : a.keyboard > b.keyboard ? 1 : a.key - b.key))
}

async function run() {
  if (!config.value) return
  running.value = true
  serverErrors.value = []
  adoptMsg.value = ''
  adoptErr.value = ''
  try {
    const req: VerifyRequest = {
      config: config.value,
      pressed: collectPressed(),
      stops: [...enabledStops].sort(),
    }
    result.value = await api.verify(req)
    lastRequest.value = req
  } catch (e) {
    result.value = null
    lastRequest.value = null
    serverErrors.value = e instanceof ApiError ? e.errors : [{ path: '', message: String(e) }]
  } finally {
    running.value = false
  }
}

async function adopt() {
  if (!lastRequest.value || !result.value) return
  adoptMsg.value = ''
  adoptErr.value = ''
  try {
    const snap = await api.saveSnapshot(snapshotName.value, lastRequest.value, result.value)
    adoptMsg.value = `已保存快照「${snap.name}」`
    snapshotName.value = ''
    await snapshotPanel.value?.refresh()
  } catch (e) {
    adoptErr.value = e instanceof ApiError ? e.errors.map((x) => x.message).join('；') : String(e)
  }
}

function loadSnapshot(snap: Snapshot) {
  configText.value = JSON.stringify(snap.input.config, null, 2)
  applyConfig()
  for (const pk of snap.input.pressed) {
    let s = pressedMap.get(pk.keyboard)
    if (!s) {
      s = new Set<number>()
      pressedMap.set(pk.keyboard, s)
    }
    s.add(pk.key)
  }
  for (const sid of snap.input.stops) enabledStops.add(sid)
  result.value = snap.result
  lastRequest.value = snap.input
  serverErrors.value = []
  adoptMsg.value = ''
}
</script>

<template>
  <header class="topbar">
    <h1>管风琴联动验算台</h1>
    <p>状态图可达闭包 · 逐边累加移调 · 越界分支裁剪 · 物理音管去重 · 稳定路径裁决</p>
  </header>

  <main class="layout">
    <div class="col">
      <section class="card">
        <h2>① 配置（导入或编辑 JSON）</h2>
        <ConfigEditor
          v-model="configText"
          :errors="serverErrors"
          :parse-error="configParseError"
          @apply="applyConfig"
        />
      </section>

      <section v-if="config" class="card">
        <h2>② 按键与启用音栓</h2>
        <template v-if="keyboards.length">
          <div class="tabs">
            <button
              v-for="kb in keyboards"
              :key="kb.id"
              type="button"
              :class="{ active: kb.id === activeKeyboard }"
              @click="activeKeyboard = kb.id"
            >
              {{ kb.id }}（{{ kb.midiMin }}–{{ kb.midiMax }}）
            </button>
          </div>
          <PianoKeyboard
            v-if="activeKeyboardCfg"
            :keyboard="activeKeyboardCfg"
            :pressed="pressedOfActive"
            @toggle="(k) => toggleKey(activeKeyboard, k)"
          />
        </template>
        <p v-else class="hint">配置中还没有键盘。请在 JSON 的 keyboards 数组中添加。</p>

        <h3>启用音栓</h3>
        <div v-if="stopsByKeyboard.size" class="stop-groups">
          <div v-for="[kb, stops] in stopsByKeyboard" :key="kb" class="stop-group">
            <div class="kb">{{ kb }}</div>
            <label v-for="s in stops" :key="s.id" class="stop-item">
              <input type="checkbox" :checked="enabledStops.has(s.id)" @change="toggleStop(s.id)" />
              <span class="mono">{{ s.id }}</span>
              <span class="hint">（{{ s.pipeCount }} 根映射）</span>
            </label>
          </div>
        </div>
        <p v-else class="hint">配置中还没有音栓。</p>

        <h3>已按下的键（{{ pressedCount }}）</h3>
        <div v-if="pressedChips.length" class="chips">
          <span v-for="c in pressedChips" :key="`${c.keyboard}#${c.key}`" class="chip">
            {{ c.keyboard }} · 键 {{ c.key }}
            <button type="button" title="松开" @click="toggleKey(c.keyboard, c.key)">×</button>
          </span>
        </div>
        <p v-else class="hint">在上方虚拟键盘点击琴键。</p>

        <div class="toolbar">
          <button type="button" class="primary" :disabled="running" @click="run">
            {{ running ? '核查中…' : `运行核查（${couplerCount} 条启用联动）` }}
          </button>
        </div>
      </section>
    </div>

    <div class="col">
      <section v-if="result" class="card">
        <h2>③ 核查结果</h2>
        <div class="summary">
          <div class="stat"><div class="n">{{ result.summary.pipes }}</div><div class="t">物理音管</div></div>
          <div class="stat"><div class="n">{{ result.summary.states }}</div><div class="t">闭包状态</div></div>
          <div class="stat"><div class="n">{{ result.summary.traversed }}</div><div class="t">采纳联动边</div></div>
          <div class="stat"><div class="n">{{ result.summary.pruned }}</div><div class="t">越界裁剪</div></div>
          <div class="stat"><div class="n">{{ result.summary.unmapped }}</div><div class="t">缺映射点</div></div>
        </div>

        <h3>传播图</h3>
        <PropagationGraph :states="result.states" :traversed="result.traversed" :pruned="result.pruned" />

        <template v-if="result.pruned.length">
          <h3>裁剪原因</h3>
          <div v-for="(p, i) in result.pruned" :key="i" class="pruned-item">
            <span class="mono">{{ p.keyboard }} · 键 {{ p.key }} —{{ p.coupler }}→ {{ p.targetKeyboard }} · 键 {{ p.targetKey }}</span>
            <br />{{ p.reason }}
          </div>
        </template>

        <template v-if="result.unmapped.length">
          <h3>潜在漏音（启用音栓在该键位无映射）</h3>
          <div v-for="(u, i) in result.unmapped" :key="i" class="unmapped-item">
            <span class="mono">{{ u.keyboard }} · 键 {{ u.key }}</span> 经音栓 <span class="mono">{{ u.stop }}</span> 无音管映射
            （{{ u.edges === 0 ? '直按' : `${u.edges} 边 · ${u.couplers.join(' → ')}` }}）
          </div>
        </template>

        <h3>音管核查单</h3>
        <PipeChecklist v-if="result.pipes.length" :pipes="result.pipes" />
        <p v-else class="hint">本次核查没有音管发声（未按键或未启用音栓）。</p>

        <div class="adopt-bar">
          <input v-model="snapshotName" type="text" placeholder="快照名称（可留空）" />
          <button type="button" class="primary" @click="adopt">采纳并保存快照</button>
          <span v-if="adoptMsg" class="ok-msg">{{ adoptMsg }}</span>
          <span v-if="adoptErr" class="hint">{{ adoptErr }}</span>
        </div>
      </section>

      <section v-else class="card">
        <h2>③ 核查结果</h2>
        <p class="hint">应用配置、按下琴键并启用音栓后，点击“运行核查”。</p>
      </section>

      <section class="card">
        <SnapshotPanel ref="snapshotPanel" @load="loadSnapshot" />
      </section>
    </div>
  </main>
</template>
