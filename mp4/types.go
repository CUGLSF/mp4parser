package mp4

import (
	"time"
)

type MP4Metadata struct {
	Duration         time.Duration // Duration
	Width            uint32        // Video Width
	Height           uint32        // Video Height
	FPS              float64       // FPS
	VideoCodec       string        // Video Codec
	AudioCodec       string        // Audio Codec
	CreationTime     time.Time     // Creation Time
	ModificationTime time.Time     // Modification Time
	HasVideo         bool          // Whether the file contains video tracks
	HasAudio         bool          // Whether the file contains audio tracks
	HasHint          bool          // Whether the file contains hint tracks
	HasMeta          bool          // Whether the file contains meta tracks
	HasAuxv          bool          // Whether the file contains auxv tracks
	VideoBitrate     uint32        // Video Bitrate (bps)
	AudioBitrate     uint32        // Audio Bitrate (bps)
	Rotation         int           // Video Rotation Angle
	MajorBrand       string        // Major Brand
	CompatibleBrands []string      // Compatible Brands
	AudioSampleRate  uint32        // Audio Sample Rate
	AudioChannels    uint16        // Audio Channels
	AudioSampleSize  uint16        // Audio Sample Size
	VideoProfile     string        // Video Encoding Configuration
	VideoLevel       byte          // Video Encoding Level
}

type TrackInfo struct {
	TrackID       uint32
	HandlerType   string
	Width         uint32
	Height        uint32
	Codec         string
	Duration      uint64
	Timescale     uint32
	SampleCount   uint32
	FrameCount    uint32
	Language      string
	Bitrate       uint32
	SampleRate    uint32
	Channels      uint16
	SampleSize    uint16
	AVCProfile    byte
	AVCLevel      byte
	AudioCodecTag uint32
	VideoCodecTag uint32
	SttsBox       *sttsBox
	CttsBox       *cttsBox
}
