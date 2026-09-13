package core

import (
	"container/heap"
	"fmt"
	"sort"
	"strconv"
)

// state 是状态图上的一个节点：键盘＋键号。
type state struct {
	keyboard string
	key      int
}

// pathTuple 是路径裁决键：边数、联动 ID 序列、起始键号。
// 联动 ID 序列按 Go 字符串序比较；UTF-8 字节序与 Unicode 码点序一致，
// 因此字符串比较即码点字典序。
type pathTuple struct {
	edges         int
	couplers      []string
	startKey      int
	startKeyboard string
}

func compareCouplerSeq(a, b []string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

// less 实现“边数、联动 ID 序列、起始键号依次取最小”。
// startKeyboard 不在规格裁决键内，仅作确定性兜底，不改变规格语义。
func (t pathTuple) less(o pathTuple) bool {
	if t.edges != o.edges {
		return t.edges < o.edges
	}
	if c := compareCouplerSeq(t.couplers, o.couplers); c != 0 {
		return c < 0
	}
	if t.startKey != o.startKey {
		return t.startKey < o.startKey
	}
	return t.startKeyboard < o.startKeyboard
}

func (t pathTuple) equal(o pathTuple) bool {
	return !t.less(o) && !o.less(t)
}

// parentInfo 记录状态最优路径的最后一条边，用于重建传播图。
type parentInfo struct {
	from    state
	coupler string
}

// pqItem 是优先队列元素。
type pqItem struct {
	st    state
	tuple pathTuple
}

// priorityQueue 按裁决键有序弹出；键盘与键号参与比较仅为确定性。
type priorityQueue []pqItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	a, b := pq[i], pq[j]
	if a.tuple.less(b.tuple) {
		return true
	}
	if b.tuple.less(a.tuple) {
		return false
	}
	if a.st.keyboard != b.st.keyboard {
		return a.st.keyboard < b.st.keyboard
	}
	return a.st.key < b.st.key
}

func (pq priorityQueue) Swap(i, j int)       { pq[i], pq[j] = pq[j], pq[i] }
func (pq *priorityQueue) Push(x interface{}) { *pq = append(*pq, x.(pqItem)) }
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	it := old[n-1]
	*pq = old[:n-1]
	return it
}

// Verify 先校验输入，再在状态图上求可达闭包并汇总核查结果。
// 校验失败时返回全部定位错误；算法不设任何深度上限，
// 仅靠有限状态空间（键盘数 × 键域）保证闭环终止。
func Verify(req VerifyRequest) (*Result, []FieldError) {
	if errs := Validate(req); len(errs) > 0 {
		return nil, errs
	}

	kbByID := map[string]Keyboard{}
	for _, kb := range req.Config.Keyboards {
		kbByID[kb.ID] = kb
	}

	// 启用的联动边按 ID 排序，保证松弛顺序确定。
	var couplers []Coupler
	for _, c := range req.Config.Couplers {
		if c.Active() {
			couplers = append(couplers, c)
		}
	}
	sort.Slice(couplers, func(i, j int) bool { return couplers[i].ID < couplers[j].ID })

	// 每个键盘上启用的音栓，按音栓 ID 排序。
	stopsByKb := map[string][]Stop{}
	for _, st := range req.Config.Stops {
		stopsByKb[st.Keyboard] = append(stopsByKb[st.Keyboard], st)
	}
	enabled := map[string]bool{}
	for _, sid := range req.Stops {
		enabled[sid] = true
	}
	for kb := range stopsByKb {
		sort.Slice(stopsByKb[kb], func(i, j int) bool { return stopsByKb[kb][i].ID < stopsByKb[kb][j].ID })
	}

	// 起始状态去重后按确定性顺序入队。
	pressedSet := map[state]bool{}
	var starts []state
	for _, pk := range req.Pressed {
		st := state{pk.Keyboard, pk.Key}
		if !pressedSet[st] {
			pressedSet[st] = true
			starts = append(starts, st)
		}
	}
	sort.Slice(starts, func(i, j int) bool {
		if starts[i].keyboard != starts[j].keyboard {
			return starts[i].keyboard < starts[j].keyboard
		}
		return starts[i].key < starts[j].key
	})

	best := map[state]pathTuple{}
	parent := map[state]parentInfo{}
	finalized := map[state]bool{}
	pq := &priorityQueue{}
	heap.Init(pq)

	push := func(st state, t pathTuple, p parentInfo, hasParent bool) {
		if finalized[st] {
			return
		}
		if cur, ok := best[st]; ok && !t.less(cur) {
			return
		}
		best[st] = t
		if hasParent {
			parent[st] = p
		} else {
			delete(parent, st)
		}
		heap.Push(pq, pqItem{st, t})
	}

	for _, st := range starts {
		push(st, pathTuple{edges: 0, couplers: nil, startKey: st.key, startKeyboard: st.keyboard}, parentInfo{}, false)
	}

	res := &Result{}
	pipeSource := map[string]Source{}

	considerPipe := func(pipeID string, cand Source) {
		cur, ok := pipeSource[pipeID]
		if !ok || sourceLess(cand, cur) {
			pipeSource[pipeID] = cand
		}
	}

	for pq.Len() > 0 {
		item := heap.Pop(pq).(pqItem)
		if finalized[item.st] {
			continue
		}
		if cur := best[item.st]; !cur.equal(item.tuple) {
			continue // 过期条目：已有更优路径
		}
		finalized[item.st] = true
		t := item.tuple

		res.States = append(res.States, StateInfo{
			Keyboard:      item.st.keyboard,
			Key:           item.st.key,
			Edges:         t.edges,
			Couplers:      append([]string(nil), t.couplers...),
			StartKeyboard: t.startKeyboard,
			StartKey:      t.startKey,
			Pressed:       pressedSet[item.st],
		})
		if p, ok := parent[item.st]; ok {
			res.Traversed = append(res.Traversed, TraversedEdge{
				FromKeyboard: p.from.keyboard,
				FromKey:      p.from.key,
				ToKeyboard:   item.st.keyboard,
				ToKey:        item.st.key,
				Coupler:      p.coupler,
			})
		}

		// 该状态经每个启用音栓发声；缺映射记入潜在漏音点。
		for _, st2 := range stopsByKb[item.st.keyboard] {
			if !enabled[st2.ID] {
				continue
			}
			if pipeID, ok := st2.Pipes[strconv.Itoa(item.st.key)]; ok {
				considerPipe(pipeID, Source{
					Stop:          st2.ID,
					Keyboard:      item.st.keyboard,
					Key:           item.st.key,
					StartKeyboard: t.startKeyboard,
					StartKey:      t.startKey,
					Edges:         t.edges,
					Couplers:      append([]string(nil), t.couplers...),
				})
			} else {
				res.Unmapped = append(res.Unmapped, Unmapped{
					Keyboard:      item.st.keyboard,
					Key:           item.st.key,
					Stop:          st2.ID,
					Edges:         t.edges,
					Couplers:      append([]string(nil), t.couplers...),
					StartKeyboard: t.startKeyboard,
					StartKey:      t.startKey,
				})
			}
		}

		// 松弛所有出边：越界即裁剪该分支，其余分支照常。
		for _, c := range couplers {
			if c.From != item.st.keyboard {
				continue
			}
			target := kbByID[c.To]
			nk := item.st.key + c.Offset
			nt := pathTuple{
				edges:         t.edges + 1,
				couplers:      append(append([]string(nil), t.couplers...), c.ID),
				startKey:      t.startKey,
				startKeyboard: t.startKeyboard,
			}
			if nk < target.MIDIMin || nk > target.MIDIMax {
				res.Pruned = append(res.Pruned, PrunedEdge{
					Keyboard:       item.st.keyboard,
					Key:            item.st.key,
					Coupler:        c.ID,
					TargetKeyboard: c.To,
					TargetKey:      nk,
					Reason:         fmt.Sprintf("目标键号 %d 超出键盘 %q 的有效音域 [%d, %d]，该分支已裁剪", nk, c.To, target.MIDIMin, target.MIDIMax),
					StartKeyboard:  t.startKeyboard,
					StartKey:       t.startKey,
					Edges:          nt.edges,
					Couplers:       nt.couplers,
				})
				continue
			}
			push(state{c.To, nk}, nt, parentInfo{from: item.st, coupler: c.ID}, true)
		}
	}

	for pipeID, src := range pipeSource {
		res.Pipes = append(res.Pipes, PipeEntry{PipeID: pipeID, Source: src})
	}
	sort.Slice(res.Pipes, func(i, j int) bool { return res.Pipes[i].PipeID < res.Pipes[j].PipeID })

	res.Summary = Summary{
		States:    len(res.States),
		Pipes:     len(res.Pipes),
		Pruned:    len(res.Pruned),
		Unmapped:  len(res.Unmapped),
		Traversed: len(res.Traversed),
	}
	return res, nil
}

// sourceLess 实现音管来源裁决：边数、联动 ID 序列、起始键号、音栓 ID
// 依次取最小；其后字段仅为确定性兜底，不参与规格裁决。
func sourceLess(a, b Source) bool {
	if a.Edges != b.Edges {
		return a.Edges < b.Edges
	}
	if c := compareCouplerSeq(a.Couplers, b.Couplers); c != 0 {
		return c < 0
	}
	if a.StartKey != b.StartKey {
		return a.StartKey < b.StartKey
	}
	if a.Stop != b.Stop {
		return a.Stop < b.Stop
	}
	if a.StartKeyboard != b.StartKeyboard {
		return a.StartKeyboard < b.StartKeyboard
	}
	if a.Keyboard != b.Keyboard {
		return a.Keyboard < b.Keyboard
	}
	return a.Key < b.Key
}
