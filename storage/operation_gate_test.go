package storage

import (
	"testing"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

func TestEveryNativeResourceMethodSerializesWithClose(t *testing.T) {
	methods := []string{
		"CloudStorage.UpdateToken", "CloudStorage.ListRecordings",
		"CloudStorage.NewReplay", "CloudStorage.ExportRecording",
		"Replay.Play", "Replay.Pause", "Replay.Resume", "Replay.Seek", "Replay.SetSpeed",
		"Replay.Speed", "Replay.CurrentTime", "Replay.Stop", "Replay.StartRecording",
		"AudioOutput.Attach", "AudioOutput.Detach",
		"VideoOutput.Attach", "VideoOutput.Detach", "VideoOutput.TakeSnapshot",
		"EncodedAudioOutput.Attach", "EncodedAudioOutput.Detach",
		"EncodedVideoOutput.Attach", "EncodedVideoOutput.Detach",
	}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var gate nativeOperationGate
			operationEntered := make(chan struct{})
			releaseOperation := make(chan struct{})
			operationDone := make(chan struct{})
			go func() {
				gate.enter()
				close(operationEntered)
				<-releaseOperation
				gate.leave()
				close(operationDone)
			}()
			<-operationEntered
			closeEntered := make(chan struct{})
			go func() {
				gate.enter()
				close(closeEntered)
				gate.leave()
			}()
			select {
			case <-closeEntered:
				t.Fatalf("Close crossed in-flight %s", method)
			case <-time.After(time.Millisecond):
			}
			close(releaseOperation)
			select {
			case <-operationDone:
			case <-time.After(time.Second):
				t.Fatalf("%s did not leave operation gate", method)
			}
			select {
			case <-closeEntered:
			case <-time.After(time.Second):
				t.Fatalf("Close did not enter after %s", method)
			}
		})
	}
}

func TestAttachPinsReplayBeforeOutputWithoutPartialSideEffects(t *testing.T) {
	var replay nativeOperationGate
	var output nativeOperationGate
	replay.enter()
	outputEntered := make(chan struct{})
	go func() {
		replay.enter()
		defer replay.leave()
		output.enter()
		close(outputEntered)
		output.leave()
	}()
	select {
	case <-outputEntered:
		t.Fatal("Attach entered the Output gate before pinning Replay")
	case <-time.After(time.Millisecond):
	}
	output.enter()
	output.leave()
	replay.leave()
	select {
	case <-outputEntered:
	case <-time.After(time.Second):
		t.Fatal("Attach did not finish after Replay Close gate released")
	}
}

func TestReplayTerminalReservationIsSerializedWithStop(t *testing.T) {
	replay := &Replay{queue: newCallbackQueue()}
	defer replay.queue.close()
	if !replay.queue.reserveTerminal() {
		t.Fatal("failed to reserve active replay terminal")
	}
	stopEntered := make(chan struct{})
	releaseStop := make(chan struct{})
	stopDone := make(chan error, 1)
	go func() {
		stopDone <- replay.withNativeThen("test_stop", func(*native.CloudStorageReplay) int32 {
			close(stopEntered)
			<-releaseStop
			return 0
		}, replay.queue.releaseTerminal)
	}()
	<-stopEntered
	playEntered := make(chan struct{})
	playDone := make(chan error, 1)
	go func() {
		playDone <- replay.withReservedTerminalNative("test_play", func(*native.CloudStorageReplay) int32 {
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
	if !replay.queue.postReservedTerminal(func() {}) {
		t.Fatal("successful Play lost its terminal reservation")
	}
}

func TestFailedReplayReplacementCannotReleaseSuccessorReservation(t *testing.T) {
	replay := &Replay{queue: newCallbackQueue()}
	defer replay.queue.close()
	failedEntered := make(chan struct{})
	releaseFailed := make(chan struct{})
	failedDone := make(chan error, 1)
	go func() {
		failedDone <- replay.withReservedTerminalNative("test_failed_play", func(*native.CloudStorageReplay) int32 {
			close(failedEntered)
			<-releaseFailed
			return 6001
		})
	}()
	<-failedEntered
	successEntered := make(chan struct{})
	successDone := make(chan error, 1)
	go func() {
		successDone <- replay.withReservedTerminalNative("test_successful_play", func(*native.CloudStorageReplay) int32 {
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
	if !replay.queue.postReservedTerminal(func() {}) {
		t.Fatal("successor Play lost its terminal reservation")
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
