package storage

import (
	"time"

	tirtc "github.com/tangeai/tirtc-client-go/v2"
)

type AudioCodec = tirtc.AudioCodec
type VideoCodec = tirtc.VideoCodec
type AudioBitstreamFormat = tirtc.AudioBitstreamFormat
type VideoBitstreamFormat = tirtc.VideoBitstreamFormat
type AudioSampleFormat = tirtc.AudioSampleFormat
type PixelFormat = tirtc.PixelFormat
type AudioFrame = tirtc.AudioFrame
type VideoPlane = tirtc.VideoPlane
type VideoFrame = tirtc.VideoFrame
type EncodedAudioFrame = tirtc.EncodedAudioFrame
type EncodedVideoFrame = tirtc.EncodedVideoFrame
type OutputState = tirtc.OutputState

const (
	AudioCodecNone  = tirtc.AudioCodecNone
	AudioCodecG711A = tirtc.AudioCodecG711A
	AudioCodecAAC   = tirtc.AudioCodecAAC
	AudioCodecPCM   = tirtc.AudioCodecPCM
	AudioCodecOpus  = tirtc.AudioCodecOpus
	AudioCodecAMR   = tirtc.AudioCodecAMR

	VideoCodecNone  = tirtc.VideoCodecNone
	VideoCodecH264  = tirtc.VideoCodecH264
	VideoCodecH265  = tirtc.VideoCodecH265
	VideoCodecMJPEG = tirtc.VideoCodecMJPEG

	AudioBitstreamNone                  = tirtc.AudioBitstreamNone
	AudioBitstreamG711APacket           = tirtc.AudioBitstreamG711APacket
	AudioBitstreamAACADTS               = tirtc.AudioBitstreamAACADTS
	AudioBitstreamAACRawAccessUnit      = tirtc.AudioBitstreamAACRawAccessUnit
	AudioBitstreamPCMInt16LEInterleaved = tirtc.AudioBitstreamPCMInt16LEInterleaved
	AudioBitstreamOpusPacket            = tirtc.AudioBitstreamOpusPacket
	AudioBitstreamAMRNBFrame            = tirtc.AudioBitstreamAMRNBFrame

	VideoBitstreamNone       = tirtc.VideoBitstreamNone
	VideoBitstreamH264AnnexB = tirtc.VideoBitstreamH264AnnexB
	VideoBitstreamH265AnnexB = tirtc.VideoBitstreamH265AnnexB
	VideoBitstreamMJPEGJFIF  = tirtc.VideoBitstreamMJPEGJFIF

	AudioSampleNone               = tirtc.AudioSampleNone
	AudioSampleInt16LEInterleaved = tirtc.AudioSampleInt16LEInterleaved
	PixelNone                     = tirtc.PixelNone
	PixelI420                     = tirtc.PixelI420
	PixelNV12                     = tirtc.PixelNV12
	PixelRGBA8888                 = tirtc.PixelRGBA8888
	OutputIdle                    = tirtc.OutputIdle
	OutputBuffering               = tirtc.OutputBuffering
	OutputDelivering              = tirtc.OutputDelivering
	OutputFailed                  = tirtc.OutputFailed
	OutputPaused                  = tirtc.OutputPaused
	OutputCompleted               = tirtc.OutputCompleted
)

type RecordingRange struct {
	StartTime time.Time
	EndTime   time.Time
}

type RecordingDay struct {
	Date         string
	HasRecording bool
}

type ReplaySpeed uint8

const (
	ReplaySpeed1x     ReplaySpeed = 0
	ReplaySpeed2x     ReplaySpeed = 1
	ReplaySpeed4x     ReplaySpeed = 2
	ReplaySpeed8x     ReplaySpeed = 3
	ReplaySpeed0_5x   ReplaySpeed = 4
	ReplaySpeed0_25x  ReplaySpeed = 5
	ReplaySpeed0_125x ReplaySpeed = 6
)

type ReplayOptions struct {
	OnTimeChanged func(time.Time)
	OnCompleted   func()
	OnError       func(error)
}

type AudioOutputOptions struct {
	OnFrame        func(AudioFrame)
	OnStateChanged func(OutputState)
	OnError        func(error)
}

type VideoOutputOptions struct {
	OnFrame        func(VideoFrame)
	OnStateChanged func(OutputState)
	OnError        func(error)
}

type EncodedAudioOutputOptions struct {
	OnFrame        func(EncodedAudioFrame)
	OnStateChanged func(OutputState)
	OnError        func(error)
}

type EncodedVideoOutputOptions struct {
	OnFrame        func(EncodedVideoFrame)
	OnStateChanged func(OutputState)
	OnError        func(error)
}

type RecordingFile struct {
	Path     string
	Duration time.Duration
}

type SnapshotFile struct {
	Path string
}

type StartRecordingOptions struct {
	VideoChannelID uint8
	AudioChannelID *uint8
}

type ExportOptions struct {
	StartTime      time.Time
	EndTime        time.Time
	VideoChannelID uint8
	AudioChannelID *uint8
}

func unixMilliseconds(value time.Time) int64 { return value.UTC().UnixMilli() }
