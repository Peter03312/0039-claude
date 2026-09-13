package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Snapshot 是“采纳”后保存的输入与结果快照。
type Snapshot struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	CreatedAt time.Time     `json:"createdAt"`
	Input     VerifyRequest `json:"input"`
	Result    Result        `json:"result"`
}

// SnapshotSummary 是列表接口使用的轻量视图。
type SnapshotSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Pipes     int       `json:"pipes"`
	States    int       `json:"states"`
	Pruned    int       `json:"pruned"`
}

// SnapshotStore 为内存快照仓库；配置文件路径后每次变更都会落盘。
type SnapshotStore struct {
	mu    sync.RWMutex
	items []Snapshot
	file  string
}

// NewSnapshotStore 创建仓库；file 非空时尝试从磁盘恢复。
func NewSnapshotStore(file string) *SnapshotStore {
	s := &SnapshotStore{file: file}
	if file != "" {
		if data, err := os.ReadFile(file); err == nil {
			var items []Snapshot
			if json.Unmarshal(data, &items) == nil {
				s.items = items
			}
		}
	}
	return s
}

// Add 保存一份快照并返回它。
func (s *SnapshotStore) Add(name string, input VerifyRequest, result Result) Snapshot {
	snap := Snapshot{
		ID:        newSnapshotID(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
		Input:     input,
		Result:    result,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, snap)
	s.persistLocked()
	return snap
}

// List 按创建时间升序返回摘要。
func (s *SnapshotStore) List() []SnapshotSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SnapshotSummary, 0, len(s.items))
	for _, it := range s.items {
		out = append(out, SnapshotSummary{
			ID:        it.ID,
			Name:      it.Name,
			CreatedAt: it.CreatedAt,
			Pipes:     it.Result.Summary.Pipes,
			States:    it.Result.Summary.States,
			Pruned:    it.Result.Summary.Pruned,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Get 按 ID 取快照。
func (s *SnapshotStore) Get(id string) (Snapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, it := range s.items {
		if it.ID == id {
			return it, true
		}
	}
	return Snapshot{}, false
}

// Delete 按 ID 删除快照。
func (s *SnapshotStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, it := range s.items {
		if it.ID == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			s.persistLocked()
			return true
		}
	}
	return false
}

func (s *SnapshotStore) persistLocked() {
	if s.file == "" {
		return
	}
	data, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return
	}
	if dir := filepath.Dir(s.file); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return
		}
	}
	_ = os.WriteFile(s.file, data, 0o644)
}

func newSnapshotID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return time.Now().UTC().Format("20060102150405") + "-" + hex.EncodeToString(b[:])
}
