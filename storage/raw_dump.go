package storage

import (
	"errors"
	"sync"
	"time"

	tirtc "github.com/tangeai/tirtc-client-go/v2"
	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

type RawDumpStopReason = tirtc.RawDumpStopReason

const (
	RawDumpStopUnknown       = tirtc.RawDumpStopUnknown
	RawDumpStopUser          = tirtc.RawDumpStopUser
	RawDumpStopTimeLimit     = tirtc.RawDumpStopTimeLimit
	RawDumpStopByteLimit     = tirtc.RawDumpStopByteLimit
	RawDumpStopSourceClosed  = tirtc.RawDumpStopSourceClosed
	RawDumpStopResourceLimit = tirtc.RawDumpStopResourceLimit
	RawDumpStopWriteFailed   = tirtc.RawDumpStopWriteFailed
)

type RawDumpOptions struct {
	AudioChannelIDs []uint8
	VideoChannelIDs []uint8
}

type RawDumpArchive = tirtc.RawDumpArchive

type rawDumpNative interface {
	Stop() (native.RawDumpArchive, int32)
	Close() int32
}

type RawDump struct {
	mu      sync.Mutex
	native  rawDumpNative
	parent  *Replay
	archive *RawDumpArchive
	closed  bool
}

const rawDumpCloseTimeout = 5 * time.Second

func normalizeRawDumpIDs(audio, video []uint8) ([]uint8, []uint8, error) {
	if len(audio)+len(video) == 0 {
		return nil, nil, ErrInvalidArgument
	}
	normalize := func(values []uint8) []uint8 {
		seen := make(map[uint8]struct{}, len(values))
		result := make([]uint8, 0, len(values))
		for _, value := range values {
			if _, ok := seen[value]; !ok {
				seen[value] = struct{}{}
				result = append(result, value)
			}
		}
		return result
	}
	return normalize(audio), normalize(video), nil
}

func (r *Replay) StartRawDump(options RawDumpOptions) (*RawDump, error) {
	audio, video, err := normalizeRawDumpIDs(options.AudioChannelIDs, options.VideoChannelIDs)
	if err != nil {
		return nil, err
	}
	if !r.enterNative() {
		return nil, ErrInUse
	}
	defer r.op.leave()
	r.mu.Lock()
	if r.closed || r.native == nil {
		r.mu.Unlock()
		return nil, ErrClosed
	}
	handle := r.native
	r.mu.Unlock()
	nativeDump, code := handle.StartRawDump(audio, video)
	if err = nativeError(code); err != nil {
		return nil, err
	}
	dump := &RawDump{native: nativeDump, parent: r}
	r.mu.Lock()
	r.dumps[dump] = struct{}{}
	r.mu.Unlock()
	return dump, nil
}

func (d *RawDump) Stop() (RawDumpArchive, error) {
	if d == nil {
		return RawDumpArchive{}, ErrClosed
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.archive != nil {
		return *d.archive, nil
	}
	if d.closed || d.native == nil {
		return RawDumpArchive{}, ErrClosed
	}
	value, code := d.native.Stop()
	if err := nativeError(code); err != nil {
		return RawDumpArchive{}, err
	}
	archive := RawDumpArchive{
		CaptureID: value.CaptureID, Path: value.Path, Size: value.Size, SHA256: value.SHA256,
		CapturedDuration: time.Duration(value.CapturedDurationMS) * time.Millisecond,
		CaptureComplete:  value.CaptureComplete, StopReason: RawDumpStopReason(value.StopReason),
		UnsavedPacketCount: value.UnsavedPacketCount, UnsavedByteCount: value.UnsavedByteCount,
		Empty: value.Empty,
	}
	d.archive = &archive
	return archive, nil
}

func (d *RawDump) Close() error {
	if d == nil {
		return ErrClosed
	}
	_, stopErr := d.Stop()
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	handle := d.native
	destroyErr := closeRawDumpNative(handle)
	var parent *Replay
	if destroyErr == nil {
		d.closed = true
		d.native = nil
		parent = d.parent
		d.parent = nil
	}
	d.mu.Unlock()
	if parent != nil {
		parent.mu.Lock()
		delete(parent.dumps, d)
		parent.mu.Unlock()
	}
	return errors.Join(stopErr, destroyErr)
}

func (d *RawDump) released() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.closed
}

func closeRawDumpNative(handle rawDumpNative) error {
	deadline := time.Now().Add(rawDumpCloseTimeout)
	delay := time.Millisecond
	for {
		err := nativeError(handle.Close())
		if !errors.Is(err, ErrInUse) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(delay)
		if delay < 20*time.Millisecond {
			delay *= 2
		}
	}
}
