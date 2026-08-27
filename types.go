package tirtc

import "time"

type ConnState uint8

const (
	ConnIdle ConnState = iota
	ConnConnecting
	ConnConnected
	ConnDisconnected
)

type OutputState uint8

const (
	OutputIdle OutputState = iota
	OutputBuffering
	OutputDelivering
	OutputFailed
	OutputPaused
	OutputCompleted
)

type AudioCodec uint8

const (
	AudioCodecNone  AudioCodec = 0
	AudioCodecG711A AudioCodec = 1
	AudioCodecAAC   AudioCodec = 2
	AudioCodecPCM   AudioCodec = 3
	AudioCodecOpus  AudioCodec = 4
	AudioCodecAMR   AudioCodec = 5
)

type VideoCodec uint8

const (
	VideoCodecNone  VideoCodec = 0
	VideoCodecH264  VideoCodec = 65
	VideoCodecH265  VideoCodec = 66
	VideoCodecMJPEG VideoCodec = 67
)

type AudioProcessingLevel uint8

const (
	AudioProcessingDisabled AudioProcessingLevel = iota
	AudioProcessingLow
	AudioProcessingMedium
	AudioProcessingHigh
)

type VideoDecoderPreference uint8

const (
	VideoDecoderAuto VideoDecoderPreference = iota
	VideoDecoderSoftware
	VideoDecoderHardware
)

type AudioBitstreamFormat uint8

const (
	AudioBitstreamNone AudioBitstreamFormat = iota
	AudioBitstreamG711APacket
	AudioBitstreamAACADTS
	AudioBitstreamAACRawAccessUnit
	AudioBitstreamPCMInt16LEInterleaved
	AudioBitstreamOpusPacket
	AudioBitstreamAMRNBFrame
)

type VideoBitstreamFormat uint8

const (
	VideoBitstreamNone VideoBitstreamFormat = iota
	VideoBitstreamH264AnnexB
	VideoBitstreamH265AnnexB
	VideoBitstreamMJPEGJFIF
)

type AudioSampleFormat uint8

const (
	AudioSampleNone AudioSampleFormat = iota
	AudioSampleInt16LEInterleaved
)

type PixelFormat uint8

const (
	PixelNone PixelFormat = iota
	PixelI420
	PixelNV12
	PixelRGBA8888
)

type OutputBufferStrategy uint8

const (
	OutputBufferAutomatic OutputBufferStrategy = iota
	OutputBufferNoBuffer
)

type OutputBufferOptions struct {
	Strategy           OutputBufferStrategy
	MaxBufferWatermark *time.Duration
}

type AudioFrame struct {
	Data              []byte
	PTS               time.Duration
	SourceTime        time.Time
	Format            AudioSampleFormat
	SampleRateHz      int
	Channels          int
	SamplesPerChannel int
	Discontinuity     bool
}

type VideoPlane struct {
	Stride int
	Data   []byte
}

type VideoFrame struct {
	PTS           time.Duration
	SourceTime    time.Time
	PixelFormat   PixelFormat
	Width         int
	Height        int
	Planes        []VideoPlane
	Discontinuity bool
}

type EncodedAudioFrame struct {
	Data            []byte
	CodecConfig     []byte
	PTS             time.Duration
	SourceTime      time.Time
	Codec           AudioCodec
	BitstreamFormat AudioBitstreamFormat
	SampleRateHz    int
	Channels        int
	Discontinuity   bool
}

type EncodedVideoFrame struct {
	Data            []byte
	CodecConfig     []byte
	PTS             time.Duration
	SourceTime      time.Time
	Codec           VideoCodec
	BitstreamFormat VideoBitstreamFormat
	Width           int
	Height          int
	KeyFrame        bool
	Discontinuity   bool
}

type StartRecordingOptions struct {
	VideoStreamID uint8
	AudioStreamID *uint8
}

type RecordingFile struct {
	Path     string
	Duration time.Duration
}

// Delete synchronously removes this Runtime-owned temporary media file.
func (f RecordingFile) Delete() error { return deleteMediaFile(f.Path) }

type SnapshotFile struct {
	Path string
}

// Delete synchronously removes this Runtime-owned temporary media file.
func (f SnapshotFile) Delete() error { return deleteMediaFile(f.Path) }
