package tirtc

import (
	"errors"
	"sync"
	"time"

	"github.com/tangeai/tirtc-client-go/v2/internal/native"
)

type RawDumpStopReason uint32

const (
	RawDumpStopUnknown       RawDumpStopReason = 0
	RawDumpStopUser          RawDumpStopReason = 1
	RawDumpStopTimeLimit     RawDumpStopReason = 2
	RawDumpStopByteLimit     RawDumpStopReason = 3
	RawDumpStopSourceClosed  RawDumpStopReason = 4
	RawDumpStopResourceLimit RawDumpStopReason = 5
	RawDumpStopWriteFailed   RawDumpStopReason = 6
)

type RawDumpOptions struct {
	AudioStreamIDs []uint8
	VideoStreamIDs []uint8
}

type RawDumpArchive struct {
	CaptureID, Path, SHA256              string
	Size                                 uint64
	CapturedDuration                     time.Duration
	CaptureComplete                      bool
	StopReason                           RawDumpStopReason
	UnsavedPacketCount, UnsavedByteCount uint64
	Empty                                bool
}

type rawDumpNative interface {
	Stop() (native.RawDumpArchive, int32)
	Close() int32
}

type RawDump struct {
	mu      sync.Mutex
	native  rawDumpNative
	parent  *Conn
	archive *RawDumpArchive
	closed  bool
}

const rawDumpCloseTimeout = 5 * time.Second

func normalizeRawDumpIDs(audio, video []uint8, maximum uint8) ([]uint8, []uint8, error) {
	if len(audio)+len(video) == 0 {
		return nil, nil, ErrInvalidArgument
	}
	normalize := func(values []uint8) ([]uint8, error) {
		seen := make(map[uint8]struct{}, len(values))
		result := make([]uint8, 0, len(values))
		for _, value := range values {
			if value > maximum {
				return nil, ErrInvalidArgument
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			result = append(result, value)
		}
		return result, nil
	}
	audioIDs, err := normalize(audio)
	if err != nil {
		return nil, nil, err
	}
	videoIDs, err := normalize(video)
	return audioIDs, videoIDs, err
}

func rawDumpArchiveFromNative(value native.RawDumpArchive) RawDumpArchive {
	return RawDumpArchive{
		CaptureID: value.CaptureID, Path: value.Path, Size: value.Size, SHA256: value.SHA256,
		CapturedDuration: time.Duration(value.CapturedDurationMS) * time.Millisecond,
		CaptureComplete:  value.CaptureComplete, StopReason: RawDumpStopReason(value.StopReason),
		UnsavedPacketCount: value.UnsavedPacketCount, UnsavedByteCount: value.UnsavedByteCount,
		Empty: value.Empty,
	}
}

func (c *Conn) StartRawDump(options RawDumpOptions) (*RawDump, error) {
	audio, video, err := normalizeRawDumpIDs(options.AudioStreamIDs, options.VideoStreamIDs, 15)
	if err != nil {
		return nil, err
	}
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.mu.Lock()
	if c.closed || c.native == nil {
		c.mu.Unlock()
		return nil, ErrClosed
	}
	handle := c.native
	c.mu.Unlock()
	nativeDump, code := handle.StartRawDump(audio, video)
	if err = nativeError(code); err != nil {
		return nil, err
	}
	dump := &RawDump{native: nativeDump, parent: c}
	c.mu.Lock()
	c.dumps[dump] = struct{}{}
	c.mu.Unlock()
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
	archive := rawDumpArchiveFromNative(value)
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
	var parent *Conn
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
