package storage

import (
	"sync/atomic"
	"testing"
)

func TestCallbackQueuePreservesTerminalAndControlWhenOrdinaryLaneIsFull(t *testing.T) {
	for _, terminalName := range []string{"completed", "error"} {
		t.Run(terminalName, func(t *testing.T) {
			q := newCallbackQueue()
			var delivered atomic.Int32
			started := make(chan struct{})
			release := make(chan struct{})
			if !q.post(func() {
				close(started)
				<-release
			}) {
				t.Fatal("failed to admit blocking ordinary event")
			}
			<-started
			accepted := 0
			for index := 0; index < callbackQueueCapacity+64; index++ {
				if q.post(func() {}) {
					accepted++
				}
			}
			if accepted != callbackQueueCapacity {
				t.Fatalf("ordinary accepted=%d want=%d", accepted, callbackQueueCapacity)
			}

			for index := 0; index < callbackControlCapacity; index++ {
				if !q.postControl(func() {}) {
					t.Fatalf("failed to fill control slot %d", index)
				}
			}
			if q.postControl(func() {}) {
				t.Fatal("control overflow was unexpectedly admitted")
			}
			if !q.reserveTerminal() {
				t.Fatalf("failed to reserve %s terminal", terminalName)
			}
			if !q.postReservedTerminal(func() { delivered.Add(1) }) {
				t.Fatalf("failed to admit %s terminal", terminalName)
			}
			if q.post(func() {}) || q.postControl(func() {}) {
				t.Fatal("work admitted after terminal")
			}
			close(release)
			q.close()
			if got := delivered.Load(); got != 1 {
				t.Fatalf("terminal deliveries=%d want=1", got)
			}
		})
	}
}

func TestCallbackQueueDrainsAdmittedEventsBeforeTerminal(t *testing.T) {
	q := newCallbackQueue()
	started := make(chan struct{})
	release := make(chan struct{})
	order := make(chan string, 3)
	if !q.post(func() {
		close(started)
		<-release
		order <- "blocking"
	}) {
		t.Fatal("failed to admit blocking event")
	}
	<-started
	if !q.post(func() { order <- "ordinary" }) {
		t.Fatal("failed to admit ordinary event")
	}
	if !q.reserveTerminal() || !q.postReservedTerminal(func() { order <- "terminal" }) {
		t.Fatal("failed to admit terminal")
	}
	if q.post(func() { order <- "late" }) {
		t.Fatal("late event was admitted")
	}
	close(release)
	q.close()

	want := []string{"blocking", "ordinary", "terminal"}
	for index, value := range want {
		if got := <-order; got != value {
			t.Fatalf("delivery %d=%q want=%q", index, got, value)
		}
	}
}

func TestCallbackQueueReusesOnlyAnEmptyReplayTerminalReservation(t *testing.T) {
	q := newCallbackQueue()
	started := make(chan struct{})
	release := make(chan struct{})
	if !q.post(func() {
		close(started)
		<-release
	}) {
		t.Fatal("failed to admit blocking event")
	}
	<-started
	created, ok := q.ensureTerminalReservation()
	if !created || !ok {
		t.Fatalf("first reservation created=%v ok=%v", created, ok)
	}
	created, ok = q.ensureTerminalReservation()
	if created || !ok {
		t.Fatalf("empty reservation reuse created=%v ok=%v", created, ok)
	}
	if !q.postReservedTerminal(func() {}) {
		t.Fatal("failed to admit reserved terminal")
	}
	created, ok = q.ensureTerminalReservation()
	if created || ok {
		t.Fatalf("posted terminal reservation created=%v ok=%v", created, ok)
	}
	close(release)
	q.close()
}

func TestUnknownOutputStateMapsToFailure(t *testing.T) {
	if got := outputStateFromNative(99); got != OutputFailed {
		t.Fatalf("unknown output state=%v want failed", got)
	}
}

func TestCallbackQueuePreservesOutputTerminalAndErrorWhenControlLaneIsFull(t *testing.T) {
	q := newCallbackQueue()
	started := make(chan struct{})
	release := make(chan struct{})
	if !q.post(func() {
		close(started)
		<-release
	}) {
		t.Fatal("failed to admit blocking ordinary event")
	}
	<-started
	for index := 0; index < callbackControlCapacity; index++ {
		if !q.postControl(func() {}) {
			t.Fatalf("failed to fill control slot %d", index)
		}
	}
	var terminal atomic.Int32
	var outputError atomic.Int32
	if !postOutputState(q, OutputCompleted, func() { terminal.Add(1) }) {
		t.Fatal("output terminal was not admitted")
	}
	if !postOutputError(q, func() { outputError.Add(1) }) {
		t.Fatal("output error was not admitted")
	}
	close(release)
	q.close()
	if terminal.Load() != 1 || outputError.Load() != 1 {
		t.Fatalf("critical callbacks terminal=%d error=%d", terminal.Load(), outputError.Load())
	}
}

func TestCallbackQueueCoalescesTransitionalOutputStateWithoutReplacingTerminal(t *testing.T) {
	q := newCallbackQueue()
	started := make(chan struct{})
	release := make(chan struct{})
	if !q.post(func() {
		close(started)
		<-release
	}) {
		t.Fatal("failed to admit blocking ordinary event")
	}
	<-started
	var state atomic.Int32
	if !postOutputState(q, OutputBuffering, func() { state.Store(1) }) ||
		!postOutputState(q, OutputDelivering, func() { state.Store(2) }) ||
		!postOutputState(q, OutputCompleted, func() { state.Add(10) }) ||
		!postOutputState(q, OutputIdle, func() { state.Store(3) }) {
		t.Fatal("failed to admit output state")
	}
	close(release)
	q.close()
	if state.Load() != 3 {
		t.Fatalf("state=%d want latest transitional delivery after preserved terminal", state.Load())
	}
}
