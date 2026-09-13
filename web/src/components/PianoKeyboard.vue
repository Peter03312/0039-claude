<script setup lang="ts">
import { computed } from 'vue'

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

// 黑键定位于其左侧白键边界：left = 之前白键数 × 白键宽 − 半个黑键宽。
const blacks = computed(() => {
  let w = 0
  const out: { key: number; leftUnits: number }[] = []
  for (const k of keys.value) {
    if (isBlack(k)) out.push({ key: k, leftUnits: w })
    else w++
  }
  return out
})
</script>

<template>
  <div class="piano" :style="{ '--white-count': whites.length }">
    <div class="whites">
      <button
        v-for="k in whites"
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
      :style="{
        width: 'calc(100% / var(--white-count) * 0.62)',
        left: `calc(${b.leftUnits} * 100% / var(--white-count) - 100% / var(--white-count) * 0.31)`,
      }"
      :title="`${noteName(b.key)} · MIDI ${b.key}`"
      @click="emit('toggle', b.key)"
    />
  </div>
</template>
