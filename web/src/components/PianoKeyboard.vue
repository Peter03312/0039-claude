<script setup lang="ts">
import { computed } from 'vue'

import { blackKeyBox } from '../lib/piano'
import type { KeyboardConfig } from '../types'

const props = defineProps<{
  keyboard: KeyboardConfig
  pressed: Set<number>
}>()

const emit = defineEmits<{
  toggle: [key: number]
}>()

const BLACK = new Set([1, 3, 6, 8, 10])
const NAMES = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B']

function pitchClass(m: number): number {
  return ((m % 12) + 12) % 12
}

function isBlack(m: number): boolean {
  return BLACK.has(pitchClass(m))
}

function noteName(m: number): string {
  return `${NAMES[pitchClass(m)]}${Math.floor(m / 12) - 1}`
}

const keys = computed(() => {
  const out: number[] = []
  for (let m = props.keyboard.midiMin; m <= props.keyboard.midiMax; m++) out.push(m)
  return out
})

const whites = computed(() => keys.value.filter((k) => !isBlack(k)))

// 退化音域（整段没有白键，如单黑键音域）：全部按键按白键槽位渲染，保证可点按。
const displayWhites = computed(() => (whites.value.length > 0 ? whites.value : keys.value))

// 黑键定位于其左侧白键边界；音域以黑键开头/收尾时钳制在键盘内部，
// 避免窄音域下黑键伸出卡片、压住相邻面板。
const blacks = computed(() => {
  if (whites.value.length === 0) return []
  let w = 0
  const out: { key: number; left: number; width: number }[] = []
  for (const k of keys.value) {
    if (isBlack(k)) {
      const box = blackKeyBox(w, whites.value.length)
      out.push({ key: k, left: box.left, width: box.width })
    } else {
      w++
    }
  }
  return out
})
</script>

<template>
  <div class="piano">
    <div class="whites">
      <button
        v-for="k in displayWhites"
        :key="k"
        type="button"
        class="white"
        :class="{ on: props.pressed.has(k) }"
        :title="`${noteName(k)} · MIDI ${k}`"
        @click="emit('toggle', k)"
      >
        <span v-if="pitchClass(k) === 0" class="c-label">{{ noteName(k) }}</span>
      </button>
    </div>
    <button
      v-for="b in blacks"
      :key="b.key"
      type="button"
      class="black"
      :class="{ on: props.pressed.has(b.key) }"
      :style="{ left: `${b.left}%`, width: `${b.width}%` }"
      :title="`${noteName(b.key)} · MIDI ${b.key}`"
      @click="emit('toggle', b.key)"
    />
  </div>
</template>
