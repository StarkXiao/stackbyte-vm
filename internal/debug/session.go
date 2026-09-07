package debug
import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"stackbyte-vm/internal/compiler"
	"stackbyte-vm/internal/vm"
	"sync"
	"time"
)
type Snapshot struct {
	ID          string          `json:"id"`
	Done        bool            `json:"done"`
	Offset      int             `json:"offset"`
	Instruction string          `json:"instruction"`
	Line        int             `json:"line"`
	Stack       []string        `json:"stack"`
	Frames      []vm.TraceFrame `json:"frames"`
	Output      string          `json:"output"`
	Error       string          `json:"error,omitempty"`
}
type session struct {
	mu      sync.Mutex
	id      string
	machine *vm.VM
	expires time.Time
	err     error
}
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*session
	limits   vm.Limits
	ttl      time.Duration
	maximum  int
}
func NewManager(limits vm.Limits, ttl time.Duration, maximum int) *Manager { return &Manager{sessions: make(map[string]*session), limits: limits, ttl: ttl, maximum: maximum} }
func (m *Manager) Create(source string) (Snapshot, error) {
	function, err := compiler.Compile(source)
	if err != nil { return Snapshot{}, err }
	machine := vm.New(m.limits)
	if err := machine.Prepare(function); err != nil { return Snapshot{}, err }
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeExpiredLocked(time.Now())
	if len(m.sessions) >= m.maximum { return Snapshot{}, fmt.Errorf("debug session limit %d reached", m.maximum) }
	id, err := newID()
	if err != nil { return Snapshot{}, err }
	item := &session{id: id, machine: machine, expires: time.Now().Add(m.ttl)}
	m.sessions[id] = item
	return item.snapshot(), nil
}
func (m *Manager) Step(ctx context.Context, id string) (Snapshot, error) {
	item, err := m.get(id)
	if err != nil { return Snapshot{}, err }
	item.mu.Lock()
	defer item.mu.Unlock()
	if item.err == nil && !item.machine.Done() { item.err = item.machine.Step(ctx) }
	item.expires = time.Now().Add(m.ttl)
	return item.snapshot(), nil
}
func (m *Manager) Continue(ctx context.Context, id string) (Snapshot, error) {
	item, err := m.get(id)
	if err != nil { return Snapshot{}, err }
	item.mu.Lock()
	defer item.mu.Unlock()
	for item.err == nil && !item.machine.Done() {
		item.err = item.machine.Step(ctx)
	}
	item.expires = time.Now().Add(m.ttl)
	return item.snapshot(), nil
}
func (m *Manager) Delete(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[id]; !ok { return false }
	delete(m.sessions, id)
	return true
}
func (m *Manager) get(id string) (*session, error) {
	m.mu.RLock()
	item := m.sessions[id]
	m.mu.RUnlock()
	if item == nil || time.Now().After(item.expires) {
		if item != nil { m.Delete(id) }
		return nil, fmt.Errorf("debug session %q not found", id)
	}
	return item, nil
}
func (m *Manager) removeExpiredLocked(now time.Time) {
	for id, item := range m.sessions {
		if now.After(item.expires) { delete(m.sessions, id) }
	}
}
func (s *session) snapshot() Snapshot {
	offset, instruction, line := s.machine.CurrentInstruction()
	result := Snapshot{
		ID: s.id, Done: s.machine.Done(), Offset: offset, Instruction: instruction,
		Line: line, Stack: s.machine.StackSnapshot(), Frames: s.machine.FrameSnapshot(), Output: s.machine.Output(),
	}
	if s.err != nil {
		result.Error = s.err.Error()
		result.Done = true
	}
	return result
}
func newID() (string, error) {
	data := make([]byte, 12)
	if _, err := rand.Read(data); err != nil { return "", fmt.Errorf("create session id: %w", err) }
	return hex.EncodeToString(data), nil
}
