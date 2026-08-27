package tirtc

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestMailboxIsBoundedAndCloseSeesActiveHandler(t *testing.T) {
	var box mailbox
	started := make(chan struct{})
	release := make(chan struct{})
	if !box.post(func() { close(started); <-release }) {
		t.Fatal("first post rejected")
	}
	<-started
	if !errors.Is(box.preflightClose(), ErrInUse) {
		t.Fatal("close crossed an active handler")
	}
	for index := 0; index < mailboxDataCapacity; index++ {
		_ = box.post(func() {})
	}
	if box.post(func() {}) {
		t.Fatal("mailbox exceeded its fixed capacity")
	}
	close(release)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if box.preflightClose() == nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("mailbox did not drain")
}

func TestMailboxLatestSlotsCoalesceAndReservedEventsSurviveDataPressure(t *testing.T) {
	var box mailbox
	box.paused = true
	for index := 0; index < mailboxDataCapacity; index++ {
		if !box.postData(func() {}) {
			t.Fatal("data post rejected before capacity")
		}
	}
	if box.postData(func() {}) {
		t.Fatal("data ring exceeded capacity")
	}

	var mu sync.Mutex
	state := 0
	terminal := false
	for value := 1; value <= 100; value++ {
		value := value
		if !box.postLatest(mailboxLatestState, func() {
			mu.Lock()
			state = value
			mu.Unlock()
		}) {
			t.Fatal("latest state rejected")
		}
	}
	if !box.postTerminal(func() {
		mu.Lock()
		terminal = true
		mu.Unlock()
	}) {
		t.Fatal("terminal slot consumed by data pressure")
	}
	box.unpause()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		done := state == 100 && terminal
		mu.Unlock()
		if done {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("latest or terminal callback was not delivered")
}

func TestMailboxStopRequestCoalescesUnderRepeatedOverflow(t *testing.T) {
	var box mailbox
	box.paused = true
	called := 0
	for value := 1; value <= 100; value++ {
		value := value
		if !box.postStopRequest(func() { called = value }) {
			t.Fatal("reserved stop request rejected")
		}
	}
	box.unpause()
	box.waitIdle()
	if called != 100 {
		t.Fatalf("stop request did not coalesce to latest callback: %d", called)
	}
}

func TestMailboxSerializesHandlers(t *testing.T) {
	var box mailbox
	var mu sync.Mutex
	active, maxActive, finished := 0, 0, 0
	done := make(chan struct{})
	for index := 0; index < 64; index++ {
		if !box.post(func() {
			mu.Lock()
			active++
			if active > maxActive {
				maxActive = active
			}
			mu.Unlock()
			time.Sleep(time.Microsecond)
			mu.Lock()
			active--
			finished++
			if finished == 64 {
				close(done)
			}
			mu.Unlock()
		}) {
			t.Fatal("post rejected")
		}
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handlers timed out")
	}
	if maxActive != 1 {
		t.Fatalf("max concurrent handlers = %d", maxActive)
	}
}

func TestMailboxCloseGateRejectsHandlerAndSuppressesQueuedCallbacks(t *testing.T) {
	var box mailbox
	handlerResult := make(chan error, 1)
	release := make(chan struct{})
	if !box.postData(func() {
		handlerResult <- box.beginClose()
		<-release
	}) {
		t.Fatal("handler post rejected")
	}
	if err := <-handlerResult; !errors.Is(err, ErrInUse) {
		t.Fatalf("beginClose from handler = %v", err)
	}
	close(release)
	box.waitIdle()

	box.mu.Lock()
	box.paused = true
	box.mu.Unlock()
	called := false
	if !box.postData(func() { called = true }) {
		t.Fatal("queued callback rejected")
	}
	if err := box.beginClose(); err != nil {
		t.Fatal(err)
	}
	if box.postControl(func() { called = true }) {
		t.Fatal("callback accepted after close gate")
	}
	box.finishClose()
	box.unpause()
	time.Sleep(10 * time.Millisecond)
	if called {
		t.Fatal("callback crossed successful close gate")
	}
}
