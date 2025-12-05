package mp4

import (
	"time"
)

// MP4文件的元信息结构
type MP4Metadata struct {
	Duration         time.Duration // 视频时长
	Width            uint32        // 视频宽度
	Height           uint32        // 视频高度
	FPS              float64       // 帧率
	VideoCodec       string        // 视频编码格式
	AudioCodec       string        // 音频编码格式
	CreationTime     time.Time     // 创建时间
	ModificationTime time.Time     // 修改时间
	HasVideo         bool          // 是否包含视频轨
	HasAudio         bool          // 是否包含音频轨
	HasHint          bool          // 是否包含hint轨
	HasMeta          bool          // 是否包含meta轨
	HasAuxv          bool          // 是否包含auxv轨
	VideoBitrate     uint32        // 视频码率 (bps)
	AudioBitrate     uint32        // 音频码率 (bps)
	Rotation         int           // 视频旋转角度
	MajorBrand       string        // 主要品牌
	CompatibleBrands []string      // 兼容品牌
	AudioSampleRate  uint32        // 音频采样率
	AudioChannels    uint16        // 音频声道数
	AudioSampleSize  uint16        // 音频采样大小（位）
	VideoProfile     string        // 视频编码配置（如H.264的profile）
	VideoLevel       byte          // 视频编码级别
}

// MP4 Box (Atom) 头部
type BoxHeader struct {
	Size      uint32
	Type      string
	LargeSize uint64 // 当Size为1时使用
	UserType  string // 当Type为'uuid'时使用
	Version   uint8  // Full Box的version
	Flags     uint32 // Full Box的flags
}

// Track信息
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
	SampleRate    uint32 // 音频采样率
	Channels      uint16 // 音频声道数
	SampleSize    uint16 // 音频采样大小
	AVCProfile    byte   // H.264 profile
	AVCLevel      byte   // H.264 level
	AudioCodecTag uint32 // 音频编码标签
	VideoCodecTag uint32 // 视频编码标签
	SttsBox       *sttsBox
	CttsBox       *cttsBox
}

// AVC1编解码器配置信息
type AVC1CodecInfo struct {
	Width          uint16
	Height         uint16
	FrameCount     uint16
	CompressorName string
	Depth          uint16
	ColorTableID   int16
	AVCC           *AVCConfigurationBox // AVC配置信息
}

// AVC配置盒子 (avcC)
type AVCConfigurationBox struct {
	ConfigurationVersion      byte
	AVCProfileIndication      byte
	ProfileCompatibility      byte
	AVCLevelIndication        byte
	LengthSizeMinusOne        byte
	SequenceParameterSets     []AVCParameterSet
	PictureParameterSets      []AVCParameterSet
	ChromaFormat              byte
	BitDepthLumaMinus8        byte
	BitDepthChromaMinus8      byte
	NumOfSequenceParameterSet byte
	NumOfPictureParameterSet  byte
}

// AVC参数集
type AVCParameterSet struct {
	Length uint16
	Data   []byte
}

// MP4A编解码器配置信息
type MP4ACodecInfo struct {
	SampleRate          uint32
	Channels            uint16
	SampleSize          uint16
	ESD                 *ESDescriptor // 基本流描述符
	AudioSpecificConfig []byte        // 音频特定配置
}

// ES描述符
type ESDescriptor struct {
	ESID                    uint16
	StreamDependenceFlag    bool
	URLFlag                 bool
	OCRStreamFlag           bool
	StreamPriority          byte
	DependsOnESID           uint16
	URLLength               byte
	URLString               string
	OCR_ESID                uint16
	DecoderConfigDescriptor *DecoderConfigDescriptor
}

// 解码器配置描述符
type DecoderConfigDescriptor struct {
	ObjectTypeIndication byte
	StreamType           byte
	BufferSizeDB         uint32
	MaxBitrate           uint32
	AvgBitrate           uint32
	DecoderSpecificInfo  []byte
}

type STSDBox struct {
	EntryCount uint32
	Entries    []SampleEntry
}

// Sample Entry接口
type SampleEntry interface {
	GetCodecType() string
	GetWidth() uint32
	GetHeight() uint32
	GetSampleRate() uint32
	GetChannels() uint16
	GetSampleSize() uint16
}

// AVC1视频样本入口
type VisualSampleEntry struct {
	DataReferenceIndex uint16
	Width              uint16
	Height             uint16
	HorizResolution    uint32 // 72 dpi
	VertResolution     uint32 // 72 dpi
	FrameCount         uint16
	CompressorName     [32]byte
	Depth              uint16
	ColorTableID       int16
	AVCConfiguration   *AVCConfigurationBox
}

func (v *VisualSampleEntry) GetCodecType() string {
	return "avc1"
}

func (v *VisualSampleEntry) GetWidth() uint32 {
	return uint32(v.Width)
}

func (v *VisualSampleEntry) GetHeight() uint32 {
	return uint32(v.Height)
}

func (v *VisualSampleEntry) GetSampleRate() uint32 {
	return 0
}

func (v *VisualSampleEntry) GetChannels() uint16 {
	return 0
}

func (v *VisualSampleEntry) GetSampleSize() uint16 {
	return 0
}

// MP4A音频样本入口
type AudioSampleEntry struct {
	DataReferenceIndex uint16
	EntryVersion       uint16
	ChannelCount       uint16
	SampleSize         uint16
	Predefined         uint16
	Reserved           uint16
	SampleRate         uint32
	ESD                *ESDescriptor
}

func (a *AudioSampleEntry) GetCodecType() string {
	return "mp4a"
}

func (a *AudioSampleEntry) GetWidth() uint32 {
	return 0
}

func (a *AudioSampleEntry) GetHeight() uint32 {
	return 0
}

func (a *AudioSampleEntry) GetSampleRate() uint32 {
	return a.SampleRate >> 16 // 高16位是实际采样率
}

func (a *AudioSampleEntry) GetChannels() uint16 {
	return a.ChannelCount
}

func (a *AudioSampleEntry) GetSampleSize() uint16 {
	return a.SampleSize
}
