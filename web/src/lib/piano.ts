// 虚拟钢琴键盘的几何计算：纯函数，便于单元测试。

export const BLACK_WIDTH_RATIO = 0.62

export interface BlackKeyBox {
  /** 左边界，键盘宽度的百分比。 */
  left: number
  /** 宽度，键盘宽度的百分比。 */
  width: number
}

/**
 * 计算黑键的水平位置（百分比）。
 * 黑键跨在其左侧第 leftUnits 条白键边界上；当音域以黑键开头或收尾时，
 * 位置会被钳制在键盘内部，避免窄音域下黑键伸出容器、盖住相邻内容。
 */
export function blackKeyBox(leftUnits: number, whiteCount: number, ratio = BLACK_WIDTH_RATIO): BlackKeyBox {
  if (whiteCount <= 0) return { left: 0, width: 0 }
  const whiteW = 100 / whiteCount
  const width = whiteW * ratio
  const raw = leftUnits * whiteW - width / 2
  const left = Math.max(0, Math.min(raw, 100 - width))
  return { left, width }
}
