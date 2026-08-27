package tirtc_test

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	tirtc "github.com/tangeai/tirtc-client-go/v2"
)

func TestPublicAPIDump(t *testing.T) {
	want, err := os.ReadFile("testdata/public_api.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := exec.Command("go", "doc", "-all", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go doc: %v\n%s", err, got)
	}
	got = append(bytes.TrimRight(got, "\n"), '\n')
	if !bytes.Equal(got, want) {
		t.Fatal("root public API differs from testdata/public_api.txt; review the contract and regenerate the dump")
	}
	for _, name := range []string{
		"ConnService", "Input", "Metrics", "LinkMode", "Log", "LogLevel", "OnSizeChanged",
		"ExportTask", "ExportOptions", "RecordingRange", "TokenExpiredHandler",
	} {
		if regexp.MustCompile(`\b` + name + `\b`).Match(got) {
			t.Fatalf("forbidden public identifier %s is present", name)
		}
	}
}

func TestPublicAPISignatures(t *testing.T) {
	var _ func(tirtc.InitOptions) error = tirtc.Init
	var _ func() error = tirtc.Shutdown
	var _ func() (string, error) = tirtc.UploadLogs
	var _ func(tirtc.ConnOptions) (*tirtc.Conn, error) = tirtc.NewConn
	var _ func(tirtc.AudioOutputOptions) (*tirtc.AudioOutput, error) = tirtc.NewAudioOutput
	var _ func(tirtc.VideoOutputOptions) (*tirtc.VideoOutput, error) = tirtc.NewVideoOutput
	var _ func(tirtc.EncodedAudioOutputOptions) (*tirtc.EncodedAudioOutput, error) = tirtc.NewEncodedAudioOutput
	var _ func(tirtc.EncodedVideoOutputOptions) (*tirtc.EncodedVideoOutput, error) = tirtc.NewEncodedVideoOutput

	var _ func(*tirtc.Conn, string, string) error = (*tirtc.Conn).Connect
	var _ func(*tirtc.Conn) error = (*tirtc.Conn).Disconnect
	var _ func(*tirtc.Conn) tirtc.ConnState = (*tirtc.Conn).State
	var _ func(*tirtc.Conn, uint8) error = (*tirtc.Conn).SubscribeAudio
	var _ func(*tirtc.Conn, uint8) error = (*tirtc.Conn).SubscribeVideo
	var _ func(*tirtc.Conn, uint8) error = (*tirtc.Conn).UnsubscribeAudio
	var _ func(*tirtc.Conn, uint8) error = (*tirtc.Conn).UnsubscribeVideo
	var _ func(*tirtc.Conn, uint32, []byte) error = (*tirtc.Conn).SendCommand
	var _ func(*tirtc.Conn, uint8, time.Duration, []byte) error = (*tirtc.Conn).SendStreamMessage
	var _ func(*tirtc.Conn, uint8) error = (*tirtc.Conn).RequestVideoKeyframe
	var _ func(*tirtc.Conn, tirtc.StartRecordingOptions) (*tirtc.RecordingTask, error) = (*tirtc.Conn).StartRecording
	var _ func(*tirtc.Conn) error = (*tirtc.Conn).Close

	_ = tirtc.InitOptions{AppID: "", CacheDir: "", Endpoint: "", ConsoleLogEnabled: false}
	_ = tirtc.ConnOptions{
		OnStateChanged:  func(tirtc.ConnState, error) {},
		OnCommand:       func(uint32, []byte) {},
		OnStreamMessage: func(uint8, time.Duration, []byte) {},
	}
	_ = tirtc.AudioOutputOptions{
		AGCLevel: tirtc.AudioProcessingDisabled, ANSLevel: tirtc.AudioProcessingDisabled,
		Buffer: tirtc.OutputBufferOptions{}, OnFrame: func(tirtc.AudioFrame) {},
		OnStateChanged: func(tirtc.OutputState) {}, OnError: func(error) {},
	}
	_ = tirtc.VideoOutputOptions{
		DecoderPreference: tirtc.VideoDecoderAuto, Buffer: tirtc.OutputBufferOptions{},
		OnFrame: func(tirtc.VideoFrame) {}, OnStateChanged: func(tirtc.OutputState) {}, OnError: func(error) {},
	}
}
