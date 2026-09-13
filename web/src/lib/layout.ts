import type { PrunedEdge, StateInfo, TraversedEdge } from '../types'

// 传播图布局：按“边数”分列，列内按（键盘, 键号）纵向排序。
// 纯函数，不依赖 DOM，便于单元测试。

export interface GraphNode {
  id: string
  label: string
  sub: string
  depth: number
  x: number
  y: number
  pressed: boolean
  prunedTarget: boolean
  reason?: string
}

export interface GraphEdge {
  fromId: string
  toId: string
  label: string
  dashed: boolean
}

export interface GraphLayout {
  nodes: GraphNode[]
  edges: GraphEdge[]
  width: number
  height: number
}

export const NODE_W = 150
export const NODE_H = 46
export const COL_W = 220
export const ROW_H = 66
export const PAD = 24

export function stateId(keyboard: string, key: number): string {
  return `${keyboard}:${key}`
}

function cmpStr(a: string, b: string): number {
  return a < b ? -1 : a > b ? 1 : 0
}

export function layoutGraph(
  states: StateInfo[],
  traversed: TraversedEdge[],
  pruned: PrunedEdge[],
): GraphLayout {
  const rowsUsed = new Map<number, number>()
  const takeRow = (depth: number): number => {
    const r = rowsUsed.get(depth) ?? 0
    rowsUsed.set(depth, r + 1)
    return r
  }

  const byDepth = new Map<number, StateInfo[]>()
  for (const s of states) {
    const arr = byDepth.get(s.edges) ?? []
    arr.push(s)
    byDepth.set(s.edges, arr)
  }

  const nodes: GraphNode[] = []
  for (const depth of [...byDepth.keys()].sort((a, b) => a - b)) {
    const list = byDepth
      .get(depth)!
      .slice()
      .sort((a, b) => cmpStr(a.keyboard, b.keyboard) || a.key - b.key)
    for (const s of list) {
      nodes.push({
        id: stateId(s.keyboard, s.key),
        label: `${s.keyboard} · 键 ${s.key}`,
        sub: s.edges === 0 ? '直按' : `${s.edges} 边 · ${s.couplers.join(' → ')}`,
        depth,
        x: PAD + depth * COL_W,
        y: PAD + takeRow(depth) * ROW_H,
        pressed: s.pressed,
        prunedTarget: false,
      })
    }
  }

  const edges: GraphEdge[] = traversed.map((t) => ({
    fromId: stateId(t.fromKeyboard, t.fromKey),
    toId: stateId(t.toKeyboard, t.toKey),
    label: t.coupler,
    dashed: false,
  }))

  // 越界裁剪的目标画成红色虚线终点，列位置为“若未越界将达到的边数”。
  pruned.forEach((p, i) => {
    const id = `pruned#${i}`
    nodes.push({
      id,
      label: `✕ ${p.targetKeyboard} · 键 ${p.targetKey}`,
      sub: p.reason,
      depth: p.edges,
      x: PAD + p.edges * COL_W,
      y: PAD + takeRow(p.edges) * ROW_H,
      pressed: false,
      prunedTarget: true,
      reason: p.reason,
    })
    edges.push({ fromId: stateId(p.keyboard, p.key), toId: id, label: p.coupler, dashed: true })
  })

  let maxDepth = 0
  let maxRows = 1
  for (const n of nodes) {
    if (n.depth > maxDepth) maxDepth = n.depth
  }
  for (const rows of rowsUsed.values()) {
    if (rows > maxRows) maxRows = rows
  }

  return {
    nodes,
    edges,
    width: PAD * 2 + (maxDepth + 1) * COL_W,
    height: PAD * 2 + maxRows * ROW_H,
  }
}
