package core

import (
	"encoding/json"
	"reflect"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

// pipeIndex 把结果音管列表转成 pipeID -> Source 便于断言。
func pipeIndex(res *Result) map[string]Source {
	m := map[string]Source{}
	for _, p := range res.Pipes {
		m[p.PipeID] = p.Source
	}
	return m
}

func stateSet(res *Result) map[[2]interface{}]bool {
	m := map[[2]interface{}]bool{}
	for _, s := range res.States {
		m[[2]interface{}{s.Keyboard, s.Key}] = true
	}
	return m
}

// 无联动：按键只经本键盘启用的音栓发声；未启用的音栓不发声。
func TestNoCoupler(t *testing.T) {
	req := VerifyRequest{
		Config: Config{
			Keyboards: []Keyboard{{ID: "I", MIDIMin: 36, MIDIMax: 96}},
			Stops: []Stop{
				{ID: "S1", Keyboard: "I", Pipes: map[string]string{"60": "P-60"}},
				{ID: "S2", Keyboard: "I", Pipes: map[string]string{"60": "Q-60"}},
			},
			Couplers: []Coupler{},
		},
		Pressed: []PressedKey{{Keyboard: "I", Key: 60}},
		Stops:   []string{"S1"},
	}
	res, errs := Verify(req)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(res.Pipes) != 1 || res.Pipes[0].PipeID != "P-60" {
		t.Fatalf("expected only P-60, got %+v", res.Pipes)
	}
	src := res.Pipes[0].Source
	if src.Edges != 0 || len(src.Couplers) != 0 || src.StartKey != 60 || src.Stop != "S1" {
		t.Fatalf("bad source: %+v", src)
	}
	if len(res.States) != 1 || len(res.Traversed) != 0 || len(res.Pruned) != 0 {
		t.Fatalf("bad graph: states=%d traversed=%d pruned=%d", len(res.States), len(res.Traversed), len(res.Pruned))
	}
}

// 多级移调：I -C1(+12)-> II -C2(+7)-> III，逐边累加。
func TestMultiLevelTransposition(t *testing.T) {
	req := VerifyRequest{
		Config: Config{
			Keyboards: []Keyboard{
				{ID: "I", MIDIMin: 36, MIDIMax: 96},
				{ID: "II", MIDIMin: 36, MIDIMax: 96},
				{ID: "III", MIDIMin: 36, MIDIMax: 96},
			},
			Stops: []Stop{
				{ID: "S-III", Keyboard: "III", Pipes: map[string]string{"79": "P-79"}},
			},
			Couplers: []Coupler{
				{ID: "C1", From: "I", To: "II", Offset: 12},
				{ID: "C2", From: "II", To: "III", Offset: 7},
			},
		},
		Pressed: []PressedKey{{Keyboard: "I", Key: 60}},
		Stops:   []string{"S-III"},
	}
	res, errs := Verify(req)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	states := stateSet(res)
	for _, want := range [][2]interface{}{{"I", 60}, {"II", 72}, {"III", 79}} {
		if !states[want] {
			t.Fatalf("missing state %v in %+v", want, res.States)
		}
	}
	pipes := pipeIndex(res)
	src, ok := pipes["P-79"]
	if !ok {
		t.Fatalf("P-79 not sounded: %+v", res.Pipes)
	}
	if src.Edges != 2 || !reflect.DeepEqual(src.Couplers, []string{"C1", "C2"}) || src.StartKey != 60 || src.Key != 79 {
		t.Fatalf("bad multi-level source: %+v", src)
	}
}

// 闭环终止：I<->II 循环联动与零偏移自环都必须终止，且状态、音管不重复。
func TestCycleTermination(t *testing.T) {
	req := VerifyRequest{
		Config: Config{
			Keyboards: []Keyboard{
				{ID: "I", MIDIMin: 36, MIDIMax: 96},
				{ID: "II", MIDIMin: 36, MIDIMax: 96},
			},
			Stops: []Stop{
				{ID: "S1", Keyboard: "I", Pipes: map[string]string{"60": "P-60"}},
				{ID: "S2", Keyboard: "II", Pipes: map[string]string{"72": "P-72"}},
			},
			Couplers: []Coupler{
				{ID: "CA", From: "I", To: "II", Offset: 12},
				{ID: "CB", From: "II", To: "I", Offset: -12},
				{ID: "CC", From: "I", To: "I", Offset: 0}, // 零偏移自环
			},
		},
		Pressed: []PressedKey{{Keyboard: "I", Key: 60}},
		Stops:   []string{"S1", "S2"},
	}
	res, errs := Verify(req)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(res.States) != 2 {
		t.Fatalf("cycle must terminate with 2 states, got %+v", res.States)
	}
	pipes := pipeIndex(res)
	if len(pipes) != 2 {
		t.Fatalf("pipes must be deduplicated, got %+v", res.Pipes)
	}
	if src := pipes["P-60"]; src.Edges != 0 || len(src.Couplers) != 0 {
		t.Fatalf("P-60 must keep the direct (0-edge) source, got %+v", src)
	}
	if src := pipes["P-72"]; src.Edges != 1 || !reflect.DeepEqual(src.Couplers, []string{"CA"}) {
		t.Fatalf("P-72 must come via CA once, got %+v", src)
	}
	// 每个状态只被确定一次：闭环回边与自环不产生新状态。
	if res.Summary.States != 2 || res.Summary.Pipes != 2 {
		t.Fatalf("bad summary: %+v", res.Summary)
	}
}

// 边缘越界：越界分支被裁剪并记录原因，其余分支不受影响。
func TestEdgeOutOfRangePruning(t *testing.T) {
	req := VerifyRequest{
		Config: Config{
			Keyboards: []Keyboard{
				{ID: "I", MIDIMin: 36, MIDIMax: 96},
				{ID: "II", MIDIMin: 36, MIDIMax: 72},
			},
			Stops: []Stop{
				{ID: "S-II", Keyboard: "II", Pipes: map[string]string{"70": "P-70"}},
			},
			Couplers: []Coupler{
				{ID: "C-up", From: "I", To: "II", Offset: 12}, // 65+12=77 越界
				{ID: "C-lo", From: "I", To: "II", Offset: 5},  // 65+5=70 有效
			},
		},
		Pressed: []PressedKey{{Keyboard: "I", Key: 65}},
		Stops:   []string{"S-II"},
	}
	res, errs := Verify(req)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(res.Pruned) != 1 {
		t.Fatalf("expected exactly 1 pruned branch, got %+v", res.Pruned)
	}
	pr := res.Pruned[0]
	if pr.Coupler != "C-up" || pr.TargetKey != 77 || pr.TargetKeyboard != "II" {
		t.Fatalf("bad pruned record: %+v", pr)
	}
	if pr.Reason == "" {
		t.Fatalf("pruned record must carry a reason")
	}
	// 未越界分支照常发声。
	pipes := pipeIndex(res)
	src, ok := pipes["P-70"]
	if !ok {
		t.Fatalf("in-range branch must still sound: %+v", res.Pipes)
	}
	if src.Edges != 1 || !reflect.DeepEqual(src.Couplers, []string{"C-lo"}) {
		t.Fatalf("bad surviving source: %+v", src)
	}
}

// 共享音管去重：多音栓、多路径触达同一物理音管只计一次，
// 来源按边数、联动序列、起始键号、音栓 ID 取最小。
func TestSharedPipeDeduplication(t *testing.T) {
	req := VerifyRequest{
		Config: Config{
			Keyboards: []Keyboard{
				{ID: "I", MIDIMin: 36, MIDIMax: 96},
				{ID: "II", MIDIMin: 36, MIDIMax: 96},
			},
			Stops: []Stop{
				{ID: "SA", Keyboard: "I", Pipes: map[string]string{"60": "P-shared"}},
				{ID: "SB", Keyboard: "I", Pipes: map[string]string{"60": "P-shared"}},
				{ID: "SC", Keyboard: "II", Pipes: map[string]string{"72": "P-shared"}},
			},
			Couplers: []Coupler{
				{ID: "C1", From: "I", To: "II", Offset: 12},
			},
		},
		Pressed: []PressedKey{{Keyboard: "I", Key: 60}},
		Stops:   []string{"SA", "SB", "SC"},
	}
	res, errs := Verify(req)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(res.Pipes) != 1 {
		t.Fatalf("shared pipe must be counted once, got %+v", res.Pipes)
	}
	src := res.Pipes[0].Source
	// 直按(0 边)胜出；同边数同键位时音栓 ID 取最小 SA。
	if src.Edges != 0 || src.Stop != "SA" || src.Keyboard != "I" || src.Key != 60 {
		t.Fatalf("bad dedup winner: %+v", src)
	}
}

// 稳定路径裁决：同边数时按联动 ID 序列取最小；
// 序列也相同时按起始键号取最小。
func TestStablePathArbitration(t *testing.T) {
	req := VerifyRequest{
		Config: Config{
			Keyboards: []Keyboard{
				{ID: "I", MIDIMin: 36, MIDIMax: 96},
				{ID: "II", MIDIMin: 36, MIDIMax: 96},
			},
			Stops: []Stop{
				// 同一音栓把 72、73 都映射到同一根音管，
				// 使两条同序列路径竞争同一音管，逼出起始键号裁决。
				{ID: "S-II", Keyboard: "II", Pipes: map[string]string{"72": "P-x", "73": "P-x"}},
			},
			Couplers: []Coupler{
				{ID: "CA", From: "I", To: "II", Offset: 12}, // 60->72, 61->73
				{ID: "CB", From: "I", To: "II", Offset: 12}, // 60->72（与 CA 竞争）
			},
		},
		Pressed: []PressedKey{{Keyboard: "I", Key: 61}, {Keyboard: "I", Key: 60}},
		Stops:   []string{"S-II"},
	}
	res, errs := Verify(req)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(res.Pipes) != 1 {
		t.Fatalf("expected single shared pipe, got %+v", res.Pipes)
	}
	src := res.Pipes[0].Source
	// CA < CB（码点字典序），起始键号 60 < 61。
	if src.Edges != 1 || !reflect.DeepEqual(src.Couplers, []string{"CA"}) || src.StartKey != 60 || src.Key != 72 {
		t.Fatalf("arbitration must pick (1, [CA], 60), got %+v", src)
	}
	// 状态 II:72 的最优路径也必须来自 CA 而非 CB。
	for _, st := range res.States {
		if st.Keyboard == "II" && st.Key == 72 {
			if !reflect.DeepEqual(st.Couplers, []string{"CA"}) {
				t.Fatalf("state II:72 must be reached via CA, got %+v", st)
			}
		}
	}
}

// 校验：重复联动 ID、未知键盘、非法音域等都必须定位反馈。
func TestValidationLocatedErrors(t *testing.T) {
	req := VerifyRequest{
		Config: Config{
			Keyboards: []Keyboard{
				{ID: "I", MIDIMin: 36, MIDIMax: 96},
				{ID: "BAD", MIDIMin: 90, MIDIMax: 20}, // 非法音域
				{ID: "I", MIDIMin: 36, MIDIMax: 96},   // 重复键盘 ID
			},
			Stops: []Stop{
				{ID: "S1", Keyboard: "GHOST", Pipes: map[string]string{"60": "P-1"}}, // 未知键盘
				{ID: "", Keyboard: "I", Pipes: map[string]string{"x": "P-2"}},        // 空 ID + 非法键
			},
			Couplers: []Coupler{
				{ID: "C1", From: "I", To: "II", Offset: 12},  // 未知键盘 II
				{ID: "C1", From: "I", To: "I", Offset: 0},    // 重复联动 ID
				{ID: "C2", From: "VOID", To: "I", Offset: 1}, // 未知键盘 VOID
			},
		},
		Pressed: []PressedKey{
			{Keyboard: "I", Key: 120},      // 超出有效音域
			{Keyboard: "NOWHERE", Key: 60}, // 未知键盘
		},
		Stops: []string{"S1", "GHOST-STOP"}, // 未知音栓
	}
	_, errs := Verify(req)
	if len(errs) == 0 {
		t.Fatalf("expected validation errors")
	}
	byPath := map[string]string{}
	for _, e := range errs {
		byPath[e.Path] = e.Message
	}
	wantPaths := []string{
		"config.keyboards[1].midiMax",
		"config.keyboards[2].id",
		"config.stops[0].keyboard",
		"config.stops[1].id",
		`config.stops[1].pipes["x"]`,
		"config.couplers[0].to",
		"config.couplers[1].id",
		"config.couplers[2].from",
		"pressed[0].key",
		"pressed[1].keyboard",
		"stops[1]",
	}
	for _, p := range wantPaths {
		if _, ok := byPath[p]; !ok {
			t.Fatalf("missing located error at %q; got %+v", p, errs)
		}
	}
	for _, e := range errs {
		if e.Message == "" {
			t.Fatalf("error at %q has empty message", e.Path)
		}
	}
}

// 禁用的联动边不参与传播。
func TestDisabledCouplerIgnored(t *testing.T) {
	req := VerifyRequest{
		Config: Config{
			Keyboards: []Keyboard{
				{ID: "I", MIDIMin: 36, MIDIMax: 96},
				{ID: "II", MIDIMin: 36, MIDIMax: 96},
			},
			Stops: []Stop{
				{ID: "S-II", Keyboard: "II", Pipes: map[string]string{"72": "P-72"}},
			},
			Couplers: []Coupler{
				{ID: "C1", From: "I", To: "II", Offset: 12, Enabled: boolPtr(false)},
			},
		},
		Pressed: []PressedKey{{Keyboard: "I", Key: 60}},
		Stops:   []string{"S-II"},
	}
	res, errs := Verify(req)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(res.Pipes) != 0 || len(res.States) != 1 {
		t.Fatalf("disabled coupler must not propagate: %+v", res)
	}
}

// 启用音栓在可达键位缺少映射时记入漏音点。
func TestUnmappedKeyReported(t *testing.T) {
	req := VerifyRequest{
		Config: Config{
			Keyboards: []Keyboard{
				{ID: "I", MIDIMin: 36, MIDIMax: 96},
				{ID: "II", MIDIMin: 36, MIDIMax: 96},
			},
			Stops: []Stop{
				{ID: "S-II", Keyboard: "II", Pipes: map[string]string{"72": "P-72"}},
			},
			Couplers: []Coupler{
				{ID: "C1", From: "I", To: "II", Offset: 12},
			},
		},
		Pressed: []PressedKey{{Keyboard: "I", Key: 60}, {Keyboard: "I", Key: 61}},
		Stops:   []string{"S-II"},
	}
	res, errs := Verify(req)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(res.Unmapped) != 1 {
		t.Fatalf("expected 1 unmapped point, got %+v", res.Unmapped)
	}
	if res.Unmapped[0].Key != 73 || res.Unmapped[0].Stop != "S-II" {
		t.Fatalf("bad unmapped record: %+v", res.Unmapped[0])
	}
}

// 确定性：同一输入重复求值结果完全一致（传播顺序不依赖 map 遍历）。
func TestDeterministic(t *testing.T) {
	req := VerifyRequest{
		Config: Config{
			Keyboards: []Keyboard{
				{ID: "I", MIDIMin: 36, MIDIMax: 96},
				{ID: "II", MIDIMin: 36, MIDIMax: 80},
			},
			Stops: []Stop{
				{ID: "SA", Keyboard: "I", Pipes: map[string]string{"60": "P1", "61": "P2"}},
				{ID: "SB", Keyboard: "II", Pipes: map[string]string{"72": "P3", "67": "P1"}},
			},
			Couplers: []Coupler{
				{ID: "C1", From: "I", To: "II", Offset: 12},
				{ID: "C2", From: "I", To: "II", Offset: 7},
				{ID: "C3", From: "II", To: "I", Offset: -12},
			},
		},
		Pressed: []PressedKey{{Keyboard: "I", Key: 60}, {Keyboard: "I", Key: 61}},
		Stops:   []string{"SA", "SB"},
	}
	first, errs := Verify(req)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	for i := 0; i < 20; i++ {
		again, errs := Verify(req)
		if len(errs) > 0 {
			t.Fatalf("unexpected errors: %+v", errs)
		}
		if !reflect.DeepEqual(first, again) {
			t.Fatalf("run %d differs:\nfirst=%+v\nagain=%+v", i, first, again)
		}
	}
}

// 序列化契约：所有数组字段（含嵌套 couplers）必须输出 [] 而非 null，
// 否则前端按数组访问会整页崩溃。
func TestResultJSONArraysNeverNull(t *testing.T) {
	cases := map[string]VerifyRequest{
		// 恰好没有缺映射、没有裁剪、没有联动边的合法核查。
		"empty-collections": {
			Config: Config{
				Keyboards: []Keyboard{{ID: "I", MIDIMin: 36, MIDIMax: 96}},
				Stops:     []Stop{{ID: "S1", Keyboard: "I", Pipes: map[string]string{"60": "P-60"}}},
				Couplers:  []Coupler{},
			},
			Pressed: []PressedKey{{Keyboard: "I", Key: 60}},
			Stops:   []string{"S1"},
		},
		// 完全没有按键：所有集合都为空。
		"no-input": {
			Config: Config{
				Keyboards: []Keyboard{{ID: "I", MIDIMin: 36, MIDIMax: 96}},
				Stops:     []Stop{{ID: "S1", Keyboard: "I", Pipes: map[string]string{"60": "P-60"}}},
				Couplers:  []Coupler{},
			},
			Pressed: []PressedKey{},
			Stops:   []string{},
		},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			res, errs := Verify(req)
			if len(errs) > 0 {
				t.Fatalf("unexpected errors: %+v", errs)
			}
			data, err := json.Marshal(res)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			for _, field := range []string{"pipes", "states", "traversed", "pruned", "unmapped"} {
				if string(raw[field]) == "null" {
					t.Fatalf("field %q serialized as null: %s", field, data)
				}
			}
			// 嵌套的联动序列也必须是数组。
			var probe struct {
				States []struct {
					Couplers json.RawMessage `json:"couplers"`
				} `json:"states"`
				Pipes []struct {
					Source struct {
						Couplers json.RawMessage `json:"couplers"`
					} `json:"source"`
				} `json:"pipes"`
			}
			if err := json.Unmarshal(data, &probe); err != nil {
				t.Fatalf("probe: %v", err)
			}
			for _, s := range probe.States {
				if string(s.Couplers) == "null" {
					t.Fatalf("state couplers serialized as null: %s", data)
				}
			}
			for _, p := range probe.Pipes {
				if string(p.Source.Couplers) == "null" {
					t.Fatalf("source couplers serialized as null: %s", data)
				}
			}
		})
	}
}
