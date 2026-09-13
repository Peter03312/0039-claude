// 与 Go 端 internal/core 的类型一一对应。

export interface KeyboardConfig {
  id: string
  midiMin: number
  midiMax: number
}

export interface StopConfig {
  id: string
  keyboard: string
  pipes: Record<string, string>
}

export interface CouplerConfig {
  id: string
  from: string
  to: string
  offset: number
  enabled?: boolean | null
}

export interface Config {
  keyboards: KeyboardConfig[]
  stops: StopConfig[]
  couplers: CouplerConfig[]
}

export interface PressedKey {
  keyboard: string
  key: number
}

export interface VerifyRequest {
  config: Config
  pressed: PressedKey[]
  stops: string[]
}

export interface FieldError {
  path: string
  message: string
}

export interface Source {
  stop: string
  keyboard: string
  key: number
  startKeyboard: string
  startKey: number
  edges: number
  couplers: string[]
}

export interface PipeEntry {
  pipeId: string
  source: Source
}

export interface PrunedEdge {
  keyboard: string
  key: number
  coupler: string
  targetKeyboard: string
  targetKey: number
  reason: string
  startKeyboard: string
  startKey: number
  edges: number
  couplers: string[]
}

export interface StateInfo {
  keyboard: string
  key: number
  edges: number
  couplers: string[]
  startKeyboard: string
  startKey: number
  pressed: boolean
}

export interface TraversedEdge {
  fromKeyboard: string
  fromKey: number
  toKeyboard: string
  toKey: number
  coupler: string
}

export interface Unmapped {
  keyboard: string
  key: number
  stop: string
  edges: number
  couplers: string[]
  startKeyboard: string
  startKey: number
}

export interface Summary {
  states: number
  pipes: number
  pruned: number
  unmapped: number
  traversed: number
}

export interface VerifyResult {
  pipes: PipeEntry[]
  states: StateInfo[]
  traversed: TraversedEdge[]
  pruned: PrunedEdge[]
  unmapped: Unmapped[]
  summary: Summary
}

export interface SnapshotSummary {
  id: string
  name: string
  createdAt: string
  pipes: number
  states: number
  pruned: number
}

export interface Snapshot {
  id: string
  name: string
  createdAt: string
  input: VerifyRequest
  result: VerifyResult
}
