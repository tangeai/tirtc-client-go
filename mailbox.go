package tirtc

import "sync"

const (
	mailboxDataCapacity    = 256
	mailboxControlCapacity = 32
	mailboxLatestState     = 0
	mailboxLatestCount     = 1
)

type mailboxEvent struct {
	sequence uint64
	fn       func()
}

// mailbox serializes callbacks without keeping an idle goroutine alive. Data
// and control events have independent fixed bounds. State and size callbacks
// use fixed latest-wins slots, while a terminal and a stop request each have a
// reserved slot that cannot be consumed by frame or message pressure.
type mailbox struct {
	mu          sync.Mutex
	data        []mailboxEvent
	control     []mailboxEvent
	latest      [mailboxLatestCount]*mailboxEvent
	terminal    *mailboxEvent
	stopRequest *mailboxEvent
	next        uint64
	draining    bool
	paused      bool
	closing     bool
	closed      bool
	overflowed  bool
	inHandler   int
	idle        *sync.Cond
}

func (m *mailbox) post(fn func()) bool { return m.postData(fn) }

func (m *mailbox) postData(fn func()) bool {
	return m.postBounded(&m.data, mailboxDataCapacity, fn)
}

func (m *mailbox) postControl(fn func()) bool {
	return m.postBounded(&m.control, mailboxControlCapacity, fn)
}

func (m *mailbox) postBounded(queue *[]mailboxEvent, capacity int, fn func()) bool {
	if fn == nil {
		return true
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.closing {
		return false
	}
	if len(*queue) >= capacity {
		m.overflowed = true
		return false
	}
	*queue = append(*queue, m.nextEvent(fn))
	m.scheduleLocked()
	return true
}

func (m *mailbox) postLatest(slot int, fn func()) bool {
	if fn == nil {
		return true
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.closing || slot < 0 || slot >= len(m.latest) {
		return false
	}
	event := m.nextEvent(fn)
	m.latest[slot] = &event
	m.scheduleLocked()
	return true
}

func (m *mailbox) postTerminal(fn func()) bool {
	return m.postReserved(&m.terminal, fn, false)
}

func (m *mailbox) postStopRequest(fn func()) bool {
	return m.postReserved(&m.stopRequest, fn, true)
}

func (m *mailbox) postReserved(slot **mailboxEvent, fn func(), replace bool) bool {
	if fn == nil {
		return true
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.closing || (*slot != nil && !replace) {
		return false
	}
	event := m.nextEvent(fn)
	*slot = &event
	m.scheduleLocked()
	return true
}

func (m *mailbox) nextEvent(fn func()) mailboxEvent {
	m.next++
	return mailboxEvent{sequence: m.next, fn: fn}
}

func (m *mailbox) scheduleLocked() {
	if !m.paused && !m.draining {
		m.draining = true
		go m.drain()
	}
}

func (m *mailbox) drain() {
	for {
		m.mu.Lock()
		if m.closed || m.closing || m.paused {
			m.draining = false
			m.broadcastLocked()
			m.mu.Unlock()
			return
		}
		event, ok := m.takeNextLocked()
		if !ok {
			m.draining = false
			m.broadcastLocked()
			m.mu.Unlock()
			return
		}
		m.inHandler++
		m.mu.Unlock()
		m.run(event.fn)
	}
}

func (m *mailbox) run(fn func()) {
	defer func() {
		m.mu.Lock()
		m.inHandler--
		m.broadcastLocked()
		if recovered := recover(); recovered != nil {
			m.draining = false
			m.mu.Unlock()
			panic(recovered)
		}
		m.mu.Unlock()
	}()
	fn()
}

func (m *mailbox) takeNextLocked() (mailboxEvent, bool) {
	// Capacity pressure is a control condition, so a pending stop request runs
	// before ordinary events instead of waiting behind the saturated data ring.
	if m.stopRequest != nil {
		event := *m.stopRequest
		m.stopRequest = nil
		return event, true
	}
	var selected *mailboxEvent
	selectEvent := func(candidate *mailboxEvent) {
		if candidate != nil && (selected == nil || candidate.sequence < selected.sequence) {
			selected = candidate
		}
	}
	if len(m.data) > 0 {
		selectEvent(&m.data[0])
	}
	if len(m.control) > 0 {
		selectEvent(&m.control[0])
	}
	for _, event := range m.latest {
		selectEvent(event)
	}
	selectEvent(m.terminal)
	if selected == nil {
		return mailboxEvent{}, false
	}
	sequence := selected.sequence
	if len(m.data) > 0 && m.data[0].sequence == sequence {
		event := m.data[0]
		m.data[0] = mailboxEvent{}
		m.data = m.data[1:]
		return event, true
	}
	if len(m.control) > 0 && m.control[0].sequence == sequence {
		event := m.control[0]
		m.control[0] = mailboxEvent{}
		m.control = m.control[1:]
		return event, true
	}
	for index, event := range m.latest {
		if event != nil && event.sequence == sequence {
			m.latest[index] = nil
			return *event, true
		}
	}
	if m.terminal != nil && m.terminal.sequence == sequence {
		event := *m.terminal
		m.terminal = nil
		return event, true
	}
	return mailboxEvent{}, false
}

func (m *mailbox) unpause() {
	m.mu.Lock()
	m.paused = false
	if m.hasEventsLocked() {
		m.scheduleLocked()
	}
	m.mu.Unlock()
}

func (m *mailbox) hasEventsLocked() bool {
	if len(m.data) != 0 || len(m.control) != 0 || m.terminal != nil || m.stopRequest != nil {
		return true
	}
	for _, event := range m.latest {
		if event != nil {
			return true
		}
	}
	return false
}

func (m *mailbox) preflightClose() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inHandler != 0 || m.closing {
		return ErrInUse
	}
	return nil
}

func (m *mailbox) beginClose() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	if m.closing || m.inHandler != 0 {
		return ErrInUse
	}
	m.closing = true
	m.data = nil
	m.control = nil
	m.latest = [mailboxLatestCount]*mailboxEvent{}
	m.terminal = nil
	m.stopRequest = nil
	m.broadcastLocked()
	return nil
}

func (m *mailbox) cancelClose() {
	m.mu.Lock()
	if !m.closed {
		m.closing = false
		if m.hasEventsLocked() {
			m.scheduleLocked()
		}
	}
	m.broadcastLocked()
	m.mu.Unlock()
}

func (m *mailbox) waitIdle() {
	m.mu.Lock()
	if m.idle == nil {
		m.idle = sync.NewCond(&m.mu)
	}
	for m.inHandler != 0 || m.draining || m.hasEventsLocked() {
		m.idle.Wait()
	}
	m.mu.Unlock()
}

func (m *mailbox) hasPendingTerminal() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.terminal != nil
}

func (m *mailbox) broadcastLocked() {
	if m.idle != nil {
		m.idle.Broadcast()
	}
}

func (m *mailbox) isOverflowed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.overflowed
}

func (m *mailbox) finishClose() {
	m.mu.Lock()
	m.closed = true
	m.closing = false
	m.data = nil
	m.control = nil
	m.latest = [mailboxLatestCount]*mailboxEvent{}
	m.terminal = nil
	m.stopRequest = nil
	m.broadcastLocked()
	m.mu.Unlock()
}
