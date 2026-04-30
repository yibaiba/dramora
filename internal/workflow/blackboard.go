package workflow

import "sync"

type BlackboardEvent struct {
	Role   string
	Output any
}

type Blackboard struct {
	mu          sync.RWMutex
	state       map[string]any
	subscribers []chan BlackboardEvent
}

func NewBlackboard() *Blackboard {
	return &Blackboard{
		state: make(map[string]any),
	}
}

func (b *Blackboard) Write(role string, output any) {
	b.mu.Lock()
	b.state[role] = output
	subs := make([]chan BlackboardEvent, len(b.subscribers))
	copy(subs, b.subscribers)
	b.mu.Unlock()

	event := BlackboardEvent{Role: role, Output: output}
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (b *Blackboard) Read(role string) (any, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	v, ok := b.state[role]
	return v, ok
}

func (b *Blackboard) ReadAll() map[string]any {
	b.mu.RLock()
	defer b.mu.RUnlock()
	cp := make(map[string]any, len(b.state))
	for k, v := range b.state {
		cp[k] = v
	}
	return cp
}

func (b *Blackboard) Snapshot() map[string]any {
	return b.ReadAll()
}

func (b *Blackboard) Restore(state map[string]any) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.state = make(map[string]any, len(state))
	for key, value := range state {
		b.state[key] = value
	}
}

func (b *Blackboard) Subscribe(bufSize int) <-chan BlackboardEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan BlackboardEvent, bufSize)
	b.subscribers = append(b.subscribers, ch)
	return ch
}
