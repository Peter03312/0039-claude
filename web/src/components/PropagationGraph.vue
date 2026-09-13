<script setup lang="ts">
import { computed } from 'vue'

import { layoutGraph, NODE_H, NODE_W, type GraphEdge } from '../lib/layout'
import type { PrunedEdge, StateInfo, TraversedEdge } from '../types'

const props = defineProps<{
  states: StateInfo[]
  traversed: TraversedEdge[]
  pruned: PrunedEdge[]
}>()

const graph = computed(() => layoutGraph(props.states, props.traversed, props.pruned))

const nodeById = computed(() => new Map(graph.value.nodes.map((n) => [n.id, n])))

interface EdgeView {
  key: string
  x1: number
  y1: number
  x2: number
  y2: number
  mx: number
  my: number
  label: string
  dashed: boolean
}

const edgeViews = computed<EdgeView[]>(() => {
  const out: EdgeView[] = []
  graph.value.edges.forEach((e: GraphEdge, i: number) => {
    const a = nodeById.value.get(e.fromId)
    const b = nodeById.value.get(e.toId)
    if (!a || !b) return
    const x1 = a.x + NODE_W
    const y1 = a.y + NODE_H / 2
    const x2 = b.x
    const y2 = b.y + NODE_H / 2
    out.push({
      key: `${e.fromId}->${e.toId}#${i}`,
      x1,
      y1,
      x2,
      y2,
      mx: (x1 + x2) / 2,
      my: (y1 + y2) / 2 - 6,
      label: e.label,
      dashed: e.dashed,
    })
  })
  return out
})
</script>

<template>
  <div class="graph-scroll">
    <svg :width="graph.width" :height="graph.height" :viewBox="`0 0 ${graph.width} ${graph.height}`">
      <defs>
        <marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
          <path d="M 0 0 L 10 5 L 0 10 z" fill="#7c3f16" />
        </marker>
        <marker id="arrow-pruned" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
          <path d="M 0 0 L 10 5 L 0 10 z" fill="#a33b20" />
        </marker>
      </defs>

      <g v-for="e in edgeViews" :key="e.key">
        <line
          :x1="e.x1"
          :y1="e.y1"
          :x2="e.x2"
          :y2="e.y2"
          :stroke="e.dashed ? '#a33b20' : '#7c3f16'"
          :stroke-width="e.dashed ? 1.4 : 1.8"
          :stroke-dasharray="e.dashed ? '5 4' : undefined"
          :marker-end="e.dashed ? 'url(#arrow-pruned)' : 'url(#arrow)'"
        />
        <text :x="e.mx" :y="e.my" text-anchor="middle" font-size="11" :fill="e.dashed ? '#a33b20' : '#7c3f16'">
          {{ e.label }}
        </text>
      </g>

      <g v-for="n in graph.nodes" :key="n.id">
        <rect
          :x="n.x"
          :y="n.y"
          :width="NODE_W"
          :height="NODE_H"
          rx="7"
          :fill="n.prunedTarget ? '#fbeae4' : n.pressed ? '#7c3f16' : '#fffdf8'"
          :stroke="n.prunedTarget ? '#a33b20' : '#7c3f16'"
          :stroke-dasharray="n.prunedTarget ? '5 4' : undefined"
          stroke-width="1.4"
        />
        <text
          :x="n.x + 10"
          :y="n.y + 18"
          font-size="12.5"
          font-weight="600"
          :fill="n.prunedTarget ? '#a33b20' : n.pressed ? '#ffe9d6' : '#2b2620'"
        >
          {{ n.label }}
        </text>
        <text :x="n.x + 10" :y="n.y + 35" font-size="10.5" :fill="n.prunedTarget ? '#a33b20' : n.pressed ? '#f0dfcc' : '#7a7264'">
          {{ n.sub }}
        </text>
      </g>
    </svg>
  </div>
</template>
