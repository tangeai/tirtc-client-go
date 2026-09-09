package storage

import (
	"sync"
	"sync/atomic"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

var callbackQueueSequence atomic.Uint64

const (
	callbackQueueCapacity   = 256
	callbackControlCapacity = 16
	callbackCriticalSlots   = 3
	criticalTerminalState   = 0
	criticalError           = 1
	criticalLatestState     = 2
)

type callbackQueue struct {
	owner    uint64
	mu       sync.Mutex
	cond     *sync.Cond
	events   []func()
	controls []func()
	critical [callbackCriticalSlots]func()
	active   int
	closed   bool
	stopped  bool
}

func newCallbackQueue() *callbackQueue {
	q := &callbackQueue{
		owner:    callbackQueueSequence.Add(1),
		events:   make([]func(), 0, callbackQueueCapacity),
		controls: make([]func(), 0, callbackControlCapacity),
	}
	q.cond = sync.NewCond(&q.mu)
	go q.run()
	return q
}

// post admits ordinary high-frequency work without blocking native callback threads.
func (q *callbackQueue) post(event func()) bool {
	if event == nil {
		return true
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed || len(q.events) == cap(q.events) {
		return false
	}
	q.events = append(q.events, event)
	q.cond.Signal()
	return true
}

// postReliable applies bounded backpressure to source facts that cannot be dropped.
// Export cancellation and progress remain callable while the producer waits here.
func (q *callbackQueue) postReliable(event func()) bool {
	if event == nil {
		return true
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	for !q.closed && len(q.events) == cap(q.events) {
		q.cond.Wait()
	}
	if q.closed {
		return false
	}
	q.events = append(q.events, event)
	q.cond.Signal()
	return true
}

// postControl uses a separately bounded, non-blocking lane so a synchronous native
// callback can never deadlock behind a user callback that re-enters the SDK.
func (q *callbackQueue) postControl(event func()) bool {
	if event == nil {
		return true
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed || len(q.controls) == cap(q.controls) {
		return false
	}
	q.controls = append(q.controls, event)
	q.cond.Signal()
	return true
}

// postCritical admits one bounded notification independently of the ordinary
// and control lanes. Terminal/error slots preserve the first notification;
// the latest-state slot intentionally coalesces transitional state changes.
func (q *callbackQueue) postCritical(slot int, event func(), replace bool) bool {
	if event == nil {
		return true
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed || slot < 0 || slot >= len(q.critical) {
		return false
	}
	if q.critical[slot] != nil && !replace {
		return true
	}
	q.critical[slot] = event
	q.cond.Signal()
	return true
}

func (q *callbackQueue) run() {
	q.mu.Lock()
	defer q.mu.Unlock()
	for {
		for !q.hasCriticalLocked() &&
			len(q.controls) == 0 && len(q.events) == 0 && !q.closed {
			q.cond.Wait()
		}
		if !q.hasCriticalLocked() &&
			len(q.controls) == 0 && len(q.events) == 0 && q.closed {
			q.stopped = true
			q.cond.Broadcast()
			return
		}
		var event func()
		if slot := q.firstCriticalLocked(); slot >= 0 {
			event = q.critical[slot]
			q.critical[slot] = nil
		} else if len(q.controls) != 0 {
			event = q.controls[0]
			copy(q.controls, q.controls[1:])
			q.controls[len(q.controls)-1] = nil
			q.controls = q.controls[:len(q.controls)-1]
		} else if len(q.events) != 0 {
			event = q.events[0]
			copy(q.events, q.events[1:])
			q.events[len(q.events)-1] = nil
			q.events = q.events[:len(q.events)-1]
		}
		q.active++
		q.cond.Broadcast()
		q.mu.Unlock()
		func() {
			defer func() { _ = recover() }()
			native.RunCallback(q.owner, event)
		}()
		q.mu.Lock()
		q.active--
		q.cond.Broadcast()
	}
}

func (q *callbackQueue) hasCriticalLocked() bool {
	return q.firstCriticalLocked() >= 0
}

func (q *callbackQueue) firstCriticalLocked() int {
	for slot, event := range q.critical {
		if event != nil {
			return slot
		}
	}
	return -1
}

func (q *callbackQueue) idle() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.active == 0 && len(q.events) == 0 && len(q.controls) == 0 && !q.hasCriticalLocked()
}

func (q *callbackQueue) inHandler() bool {
	return native.InCallback(q.owner)
}

func (q *callbackQueue) waitIdle() {
	q.mu.Lock()
	for q.active != 0 || len(q.events) != 0 || len(q.controls) != 0 ||
		q.hasCriticalLocked() {
		q.cond.Wait()
	}
	q.mu.Unlock()
}

func (q *callbackQueue) replayCloseReady() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.active == 0 && len(q.events) == 0 && len(q.controls) == 0 &&
		!q.hasCriticalLocked()
}

func (q *callbackQueue) close() {
	q.mu.Lock()
	q.closed = true
	q.cond.Broadcast()
	for !q.stopped {
		q.cond.Wait()
	}
	q.mu.Unlock()
}
