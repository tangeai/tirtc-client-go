package storage

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestAcceptedNativeTasksKeepSourceOrder(t *testing.T) {
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
	if !q.postReliable(func() { order <- "terminal" }) {
		t.Fatal("failed to admit terminal")
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

func TestRecordingGapQueueBackpressurePreservesEveryAcceptedEvent(t *testing.T) {
	q := newCallbackQueue()
	entered, release := make(chan struct{}), make(chan struct{})
	q.post(func() { close(entered); <-release })
	<-entered
	var delivered atomic.Int32
	for i := 0; i < callbackQueueCapacity; i++ {
		if !q.post(func() { delivered.Add(1) }) {
			t.Fatal("queue rejected within capacity")
		}
	}
	accepted := make(chan bool, 1)
	go func() { accepted <- q.postReliable(func() { delivered.Add(1) }) }()
	early := false
	select {
	case <-accepted:
		early = true
		t.Error("reliable event did not wait for capacity")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if !early && !<-accepted {
		t.Error("reliable event was dropped")
	}
	q.close()
	if got := delivered.Load(); got != callbackQueueCapacity+1 {
		t.Fatalf("delivered = %d", got)
	}
}
