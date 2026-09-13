import { describe, expect, it } from 'vitest'

import { blackKeyBox } from './piano'

describe('blackKeyBox', () => {
  it('居中跨在左侧白键边界上', () => {
    // 4 个白键：每个宽 25%，黑键宽 15.5%，第 2 条边界 → 50% − 7.75%
    const box = blackKeyBox(2, 4)
    expect(box.width).toBeCloseTo(25 * 0.62)
    expect(box.left).toBeCloseTo(50 - box.width / 2)
  })

  it('音域以黑键开头时向左钳制，不伸出容器', () => {
    const box = blackKeyBox(0, 3)
    expect(box.left).toBe(0)
    expect(box.left + box.width).toBeLessThanOrEqual(100)
  })

  it('音域以黑键收尾时向右钳制，不盖住右侧内容', () => {
    const box = blackKeyBox(5, 5)
    expect(box.left + box.width).toBeCloseTo(100)
    expect(box.left).toBeGreaterThanOrEqual(0)
  })

  it('极窄音域（单白键）仍保持在界内', () => {
    const box = blackKeyBox(1, 1)
    expect(box.width).toBeCloseTo(62)
    expect(box.left).toBeCloseTo(38)
    expect(box.left + box.width).toBeLessThanOrEqual(100)
  })

  it('没有白键时退化为零宽，不产生 NaN', () => {
    const box = blackKeyBox(0, 0)
    expect(box.left).toBe(0)
    expect(box.width).toBe(0)
    expect(Number.isNaN(box.left)).toBe(false)
  })
})
