// Package core 实现管风琴联动核查的领域逻辑：
// 在“键盘＋键号”状态图上求可达闭包、逐边累加移调、
// 越界裁剪、物理音管去重与稳定路径裁决。
package core

// Keyboard 描述一层手键盘（或脚键盘）及其有效 MIDI 键域。
type Keyboard struct {
	ID      string `json:"id"`
	MIDIMin int    `json:"midiMin"`
	MIDIMax int    `json:"midiMax"`
}

// Stop 描述一个音栓：所属键盘与“键号 -> 物理音管 ID”映射。
// JSON 对象键为 MIDI 键号的十进制字符串。
type Stop struct {
	ID       string            `json:"id"`
	Keyboard string            `json:"keyboard"`
	Pipes    map[string]string `json:"pipes"`
}

// Coupler 描述一条联动边：从按键键盘指向发声键盘，
// offset 加到当前键号。Enabled 缺省（null）视为 true。
type Coupler struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	Offset  int    `json:"offset"`
	Enabled *bool  `json:"enabled,omitempty"`
}

// Active 报告该联动边是否启用。
func (c Coupler) Active() bool { return c.Enabled == nil || *c.Enabled }

// Config 为验算台导入的完整配置。
type Config struct {
	Keyboards []Keyboard `json:"keyboards"`
	Stops     []Stop     `json:"stops"`
	Couplers  []Coupler  `json:"couplers"`
}

// PressedKey 是用户在虚拟键盘上按下的一枚键。
type PressedKey struct {
	Keyboard string `json:"keyboard"`
	Key      int    `json:"key"`
}

// VerifyRequest 为一次核查的完整输入。
type VerifyRequest struct {
	Config  Config       `json:"config"`
	Pressed []PressedKey `json:"pressed"`
	Stops   []string     `json:"stops"`
}

// FieldError 把校验错误定位到具体 JSON 路径。
type FieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Source 记录一根音管被触发的最优来源路径。
type Source struct {
	Stop          string   `json:"stop"`          // 发声音栓
	Keyboard      string   `json:"keyboard"`      // 发声键盘
	Key           int      `json:"key"`           // 发声键号
	StartKeyboard string   `json:"startKeyboard"` // 起始键盘
	StartKey      int      `json:"startKey"`      // 起始键号
	Edges         int      `json:"edges"`         // 经过的联动边数
	Couplers      []string `json:"couplers"`      // 联动 ID 序列
}

// PipeEntry 是核查单中的一行：一根物理音管及其最优来源。
type PipeEntry struct {
	PipeID string `json:"pipeId"`
	Source Source `json:"source"`
}

// PrunedEdge 记录一次越界裁剪（单条分支被裁，不影响其他分支）。
type PrunedEdge struct {
	Keyboard       string   `json:"keyboard"` // 出发键盘
	Key            int      `json:"key"`      // 出发键号
	Coupler        string   `json:"coupler"`
	TargetKeyboard string   `json:"targetKeyboard"`
	TargetKey      int      `json:"targetKey"`
	Reason         string   `json:"reason"`
	StartKeyboard  string   `json:"startKeyboard"`
	StartKey       int      `json:"startKey"`
	Edges          int      `json:"edges"`    // 若未越界将达到的边数
	Couplers       []string `json:"couplers"` // 若未越界将形成的联动序列
}

// StateInfo 是闭包中一个已确定最优路径的状态（键盘＋键号）。
type StateInfo struct {
	Keyboard      string   `json:"keyboard"`
	Key           int      `json:"key"`
	Edges         int      `json:"edges"`
	Couplers      []string `json:"couplers"`
	StartKeyboard string   `json:"startKeyboard"`
	StartKey      int      `json:"startKey"`
	Pressed       bool     `json:"pressed"`
}

// TraversedEdge 是传播图上实际采纳的一条联动边。
type TraversedEdge struct {
	FromKeyboard string `json:"fromKeyboard"`
	FromKey      int    `json:"fromKey"`
	ToKeyboard   string `json:"toKeyboard"`
	ToKey        int    `json:"toKey"`
	Coupler      string `json:"coupler"`
}

// Unmapped 记录“启用的音栓在该键位没有音管映射”的潜在漏音点。
type Unmapped struct {
	Keyboard      string   `json:"keyboard"`
	Key           int      `json:"key"`
	Stop          string   `json:"stop"`
	Edges         int      `json:"edges"`
	Couplers      []string `json:"couplers"`
	StartKeyboard string   `json:"startKeyboard"`
	StartKey      int      `json:"startKey"`
}

// Summary 汇总一次核查的关键计数。
type Summary struct {
	States    int `json:"states"`
	Pipes     int `json:"pipes"`
	Pruned    int `json:"pruned"`
	Unmapped  int `json:"unmapped"`
	Traversed int `json:"traversed"`
}

// Result 为一次核查的完整输出。
type Result struct {
	Pipes     []PipeEntry     `json:"pipes"`
	States    []StateInfo     `json:"states"`
	Traversed []TraversedEdge `json:"traversed"`
	Pruned    []PrunedEdge    `json:"pruned"`
	Unmapped  []Unmapped      `json:"unmapped"`
	Summary   Summary         `json:"summary"`
}
