package storage

import (
	"testing"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

func TestReplayStopSerializesNewPlay(t *testing.T) {
	replay := &Replay{queue: newCallbackQueue()}
	defer replay.queue.close()
	stopEntered := make(chan struct{})
	releaseStop := make(chan struct{})
	stopDone := make(chan error, 1)
	go func() {
		stopDone <- replay.withNative("test_stop", func(*native.CloudStorageReplay) int32 {
			close(stopEntered)
			<-releaseStop
			return 0
		})
	}()
	<-stopEntered
	playEntered := make(chan struct{})
	playDone := make(chan error, 1)
	go func() {
		playDone <- replay.withNative("test_play", func(*native.CloudStorageReplay) int32 {
			close(playEntered)
			return 0
		})
	}()
	select {
	case <-playEntered:
		t.Fatal("Play crossed in-flight Stop")
	case <-time.After(time.Millisecond):
	}
	close(releaseStop)
	if err := <-stopDone; err != nil {
		t.Fatal(err)
	}
	select {
	case <-playEntered:
	case <-time.After(time.Second):
		t.Fatal("Play did not run after Stop")
	}
	if err := <-playDone; err != nil {
		t.Fatal(err)
	}
}

func TestFailedReplayReplacementSerializesSuccessor(t *testing.T) {
	replay := &Replay{queue: newCallbackQueue()}
	defer replay.queue.close()
	failedEntered := make(chan struct{})
	releaseFailed := make(chan struct{})
	failedDone := make(chan error, 1)
	go func() {
		failedDone <- replay.withNative("test_failed_play", func(*native.CloudStorageReplay) int32 {
			close(failedEntered)
			<-releaseFailed
			return 6001
		})
	}()
	<-failedEntered
	successEntered := make(chan struct{})
	successDone := make(chan error, 1)
	go func() {
		successDone <- replay.withNative("test_successful_play", func(*native.CloudStorageReplay) int32 {
			close(successEntered)
			return 0
		})
	}()
	select {
	case <-successEntered:
		t.Fatal("successor Play crossed failed Play")
	case <-time.After(time.Millisecond):
	}
	close(releaseFailed)
	if err := <-failedDone; err == nil {
		t.Fatal("failed Play unexpectedly succeeded")
	}
	select {
	case <-successEntered:
	case <-time.After(time.Second):
		t.Fatal("successor Play did not run")
	}
	if err := <-successDone; err != nil {
		t.Fatal(err)
	}
}

func TestNativeCloseReleasesGateBeforeDrainingCallbacks(t *testing.T) {
	var gate nativeOperationGate
	queue := newCallbackQueue()
	gateEntered := make(chan struct{})
	beginDrain := make(chan struct{})
	closeDone := make(chan struct{})
	go func() {
		gate.enter()
		close(gateEntered)
		<-beginDrain
		finishNativeClose(&gate, queue)
		close(closeDone)
	}()
	<-gateEntered
	callbackDone := make(chan struct{})
	if !queue.post(func() {
		gate.enter()
		gate.leave()
		close(callbackDone)
	}) {
		t.Fatal("failed to admit callback")
	}
	close(beginDrain)
	select {
	case <-callbackDone:
	case <-time.After(time.Second):
		t.Fatal("callback reentry deadlocked behind native Close")
	}
	select {
	case <-closeDone:
	case <-time.After(time.Second):
		t.Fatal("native Close did not finish callback drain")
	}
}

func TestReplayCallbackDoesNotWaitForOperationDrainingIt(t *testing.T) {
	replay := &Replay{queue: newCallbackQueue()}
	defer replay.queue.close()
	methods := map[string]func() error{
		"Pause":          replay.Pause,
		"Resume":         replay.Resume,
		"CurrentTime":    func() error { _, _, err := replay.CurrentTime(); return err },
		"StartRecording": func() error { _, err := replay.StartRecording(StartRecordingOptions{}); return err },
		"Close":          replay.Close,
	}
	for name, call := range methods {
		t.Run(name, func(t *testing.T) {
			replay.op.enter()
			result := make(chan error, 1)
			replay.queue.postReliable(func() { result <- call() })
			select {
			case err := <-result:
				replay.op.leave()
				if err != ErrInUse {
					t.Fatalf("callback returned %v, want ErrInUse", err)
				}
			case <-time.After(time.Second):
				replay.op.leave()
				t.Fatal("callback waited behind the operation draining it")
			}
			replay.queue.waitIdle()
		})
	}
}
