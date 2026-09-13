package core

import (
	"fmt"
	"sort"
	"strconv"
)

// Validate 对核查输入做完整校验，所有错误都带 JSON 路径定位。
// 键盘、音栓、联动 ID 必须是各自类型内全局唯一的非空字符串。
func Validate(req VerifyRequest) []FieldError {
	var errs []FieldError
	add := func(path, format string, args ...interface{}) {
		errs = append(errs, FieldError{Path: path, Message: fmt.Sprintf(format, args...)})
	}

	kbIndex := map[string]int{} // id -> 首次出现的下标
	for i, kb := range req.Config.Keyboards {
		p := fmt.Sprintf("config.keyboards[%d]", i)
		if kb.ID == "" {
			add(p+".id", "键盘 ID 不能为空")
		} else if first, dup := kbIndex[kb.ID]; dup {
			add(p+".id", "键盘 ID %q 与 config.keyboards[%d] 重复", kb.ID, first)
		} else {
			kbIndex[kb.ID] = i
		}
		if kb.MIDIMin < 0 || kb.MIDIMin > 127 {
			add(p+".midiMin", "非法音域：midiMin %d 超出 MIDI 范围 [0, 127]", kb.MIDIMin)
		}
		if kb.MIDIMax < 0 || kb.MIDIMax > 127 {
			add(p+".midiMax", "非法音域：midiMax %d 超出 MIDI 范围 [0, 127]", kb.MIDIMax)
		}
		if kb.MIDIMin >= 0 && kb.MIDIMax <= 127 && kb.MIDIMin > kb.MIDIMax {
			add(p+".midiMax", "非法音域：midiMin %d 大于 midiMax %d", kb.MIDIMin, kb.MIDIMax)
		}
	}

	stopIndex := map[string]int{}
	for i, st := range req.Config.Stops {
		p := fmt.Sprintf("config.stops[%d]", i)
		if st.ID == "" {
			add(p+".id", "音栓 ID 不能为空")
		} else if first, dup := stopIndex[st.ID]; dup {
			add(p+".id", "音栓 ID %q 与 config.stops[%d] 重复", st.ID, first)
		} else {
			stopIndex[st.ID] = i
		}
		kb, known := lookupKeyboard(req.Config, st.Keyboard)
		if st.Keyboard == "" {
			add(p+".keyboard", "音栓 %q 未指定所属键盘", st.ID)
		} else if !known {
			add(p+".keyboard", "未知键盘 %q（音栓 %q 的所属键盘不存在）", st.Keyboard, st.ID)
		}
		keys := make([]string, 0, len(st.Pipes))
		for k := range st.Pipes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			pp := p + ".pipes[" + strconv.Quote(k) + "]"
			n, err := strconv.Atoi(k)
			if err != nil {
				add(pp, "音栓 %q 的音管映射键 %q 不是整数键号", st.ID, k)
				continue
			}
			if known && (n < kb.MIDIMin || n > kb.MIDIMax) {
				add(pp, "音栓 %q 的键号 %d 超出键盘 %q 的有效音域 [%d, %d]", st.ID, n, st.Keyboard, kb.MIDIMin, kb.MIDIMax)
			}
			if st.Pipes[k] == "" {
				add(pp, "音栓 %q 键号 %d 映射的物理音管 ID 不能为空", st.ID, n)
			}
		}
	}

	couplerIndex := map[string]int{}
	for i, c := range req.Config.Couplers {
		p := fmt.Sprintf("config.couplers[%d]", i)
		if c.ID == "" {
			add(p+".id", "联动 ID 不能为空")
		} else if first, dup := couplerIndex[c.ID]; dup {
			add(p+".id", "联动 ID %q 与 config.couplers[%d] 重复", c.ID, first)
		} else {
			couplerIndex[c.ID] = i
		}
		if _, ok := lookupKeyboard(req.Config, c.From); !ok {
			add(p+".from", "未知键盘 %q（联动 %q 的按键键盘不存在）", c.From, c.ID)
		}
		if _, ok := lookupKeyboard(req.Config, c.To); !ok {
			add(p+".to", "未知键盘 %q（联动 %q 的发声键盘不存在）", c.To, c.ID)
		}
	}

	for i, pk := range req.Pressed {
		p := fmt.Sprintf("pressed[%d]", i)
		kb, ok := lookupKeyboard(req.Config, pk.Keyboard)
		if !ok {
			add(p+".keyboard", "未知键盘 %q（按下的键所在的键盘不存在）", pk.Keyboard)
			continue
		}
		if pk.Key < kb.MIDIMin || pk.Key > kb.MIDIMax {
			add(p+".key", "按下的键号 %d 超出键盘 %q 的有效音域 [%d, %d]", pk.Key, pk.Keyboard, kb.MIDIMin, kb.MIDIMax)
		}
	}

	for i, sid := range req.Stops {
		if _, ok := stopIndex[sid]; !ok {
			add(fmt.Sprintf("stops[%d]", i), "未知音栓 %q（启用的音栓不存在）", sid)
		}
	}

	return errs
}

func lookupKeyboard(cfg Config, id string) (Keyboard, bool) {
	for _, kb := range cfg.Keyboards {
		if kb.ID == id {
			return kb, true
		}
	}
	return Keyboard{}, false
}
