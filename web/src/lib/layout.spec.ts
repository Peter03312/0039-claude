import { describe, expect, it } from 'vitest'

import type { PrunedEdge, StateInfo, TraversedEdge } from '../types'
import { layoutGraph, stateId } from './layout'

function st(keyboard: string, key: number, edges: number, couplers: string[] = [], pressed = false): StateInfo {
  return { keyboard, key, edges, couplers, startKeyboard: 'I', startKey: 60, pressed }
}

describe('layoutGraph', () => {
  it('按边数分列，列内按（键盘, 键号）排序且确定', () => {
    const states = [
      st('III', 79, 2, ['C1', 'C2']),
      st('II', 72, 1, ['C1']),
      st('I', 60, 0, [], true),
      st('II', 65, 1, ['C2']),
    ]
    const g = layoutGraph(states, [], [])
    const node = (kb: string, k: number) => g.nodes.find((n) => n.id === stateId(kb, k))!
    expect(node('I', 60).x).toBeLessThan(node('II', 72).x)
    expect(node('II', 72).x).toBeLessThan(node('III', 79).x)
    // 同列同 x，不同 y；65 排在 72 之前
    expect(node('II', 65).x).toBe(node('II', 72).x)
    expect(node('II', 65).y).toBeLessThan(node('II', 72).y)
    // 与输入顺序无关：重排输入结果一致
    const g2 = layoutGraph([...states].reverse(), [], [])
    expect(g2).toEqual(g)
  })

  it('遍历边连接已布局节点并携带联动标签', () => {
    const states = [st('I', 60, 0, [], true), st('II', 72, 1, ['C1'])]
    const traversed: TraversedEdge[] = [
      { fromKeyboard: 'I', fromKey: 60, toKeyboard: 'II', toKey: 72, coupler: 'C1' },
    ]
    const g = layoutGraph(states, traversed, [])
    expect(g.edges).toHaveLength(1)
    expect(g.edges[0]).toMatchObject({ fromId: 'I:60', toId: 'II:72', label: 'C1', dashed: false })
  })

  it('越界裁剪目标放在“将达到的边数”那一列并携带原因', () => {
    const states = [st('I', 65, 0, [], true)]
    const pruned: PrunedEdge[] = [
      {
        keyboard: 'I',
        key: 65,
        coupler: 'C-up',
        targetKeyboard: 'II',
        targetKey: 77,
        reason: '目标键号 77 超出有效音域',
        startKeyboard: 'I',
        startKey: 65,
        edges: 1,
        couplers: ['C-up'],
      },
    ]
    const g = layoutGraph(states, [], pruned)
    const src = g.nodes.find((n) => n.id === stateId('I', 65))!
    const dst = g.nodes.find((n) => n.prunedTarget)!
    expect(dst.x).toBeGreaterThan(src.x)
    expect(dst.reason).toBe('目标键号 77 超出有效音域')
    expect(g.edges.some((e) => e.dashed && e.fromId === src.id && e.toId === dst.id)).toBe(true)
  })

  it('空闭包不产生节点，画布尺寸仍合法', () => {
    const g = layoutGraph([], [], [])
    expect(g.nodes).toHaveLength(0)
    expect(g.edges).toHaveLength(0)
    expect(g.width).toBeGreaterThan(0)
    expect(g.height).toBeGreaterThan(0)
  })
})
