package mp4

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// MP4解析器
type MP4Parser struct {
	file       *os.File
	metadata   MP4Metadata
	tracks     []TrackInfo
	moovOffset int64
}

// 创建新的MP4解析器
func NewParser(filename string) (*MP4Parser, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}

	return &MP4Parser{
		file:     file,
		metadata: MP4Metadata{},
		tracks:   []TrackInfo{},
	}, nil
}

// 解析MP4文件
func (p *MP4Parser) Parse() (*MP4Metadata, error) {
	defer p.file.Close()

	for {
		size, boxType, err := readHeader(p.file)
		if err != nil {
			return nil, err
		}

		fmt.Printf("DEBUG: parsing atom: %s, size: %d\n", boxType, size)

		switch boxType {
		case "moov":
			if err := p.parseMoovAtom(size - 8); err != nil {
				return nil, err
			}
			if err := p.calculateMetadata(); err != nil {
				return nil, err
			}

			return &p.metadata, nil
		default:
			p.file.Seek(int64(size-8), io.SeekCurrent)
		}
	}
}

func readHeader(f *os.File) (uint32, string, error) {
	buf := make([]byte, 8)
	n, err := io.ReadFull(f, buf)
	if err != nil {
		return 0, "", err
	}
	if n != 8 {
		return 0, "", fmt.Errorf("读取文件头失败")
	}
	for i := 0; i < 8; i++ {
		fmt.Printf("%02x ", buf[i])
	}
	fmt.Printf("\n")
	size := binary.BigEndian.Uint32(buf[0:4])
	boxType := string(buf[4:8])
	return size, boxType, nil
}

func cur(f *os.File) int64 {
	pos, _ := f.Seek(0, io.SeekCurrent)
	return pos
}

func (p *MP4Parser) parseMoovAtom(dataSize uint32) error {
	end := cur(p.file) + int64(dataSize)

	for cur(p.file) < end {
		atomSize, atomType, err := readHeader(p.file)
		if err != nil {
			fmt.Printf("error reading atom header: %v", err)
			return err
		}
		fmt.Printf("parsing atom type: %s, size: %d\n", atomType, atomSize)
		switch atomType {
		case BoxTypeMVHD:
			if err := p.parseMvhdAtom(); err != nil {
				return err
			}
		case BoxTypeTRAK:
			if err := p.parseTrakAtom(atomSize - 8); err != nil {
				return err
			}
		default:
			p.file.Seek(int64(atomSize-8), io.SeekCurrent)
		}
	}
	return nil
}
func readByte(f *os.File) byte {
	buf := []byte{0}
	f.Read(buf)
	return buf[0]
}
func readU32(f *os.File) uint32 {
	buf := make([]byte, 4)
	f.Read(buf)
	return binary.BigEndian.Uint32(buf)
}

func readU64(f *os.File) uint64 {
	buf := make([]byte, 8)
	f.Read(buf)
	return binary.BigEndian.Uint64(buf)
}

// 解析mvhd atom
// aligned(8) class MovieHeaderBox extends FullBox(‘mvhd’, version, 0) {
//
//	if (version==1) {
//		unsigned int(64) creation_time;
//		unsigned int(64) modification_time;
//		unsigned int(32) timescale;
//		unsigned int(64) duration;
//		} else { // version==0
//		unsigned int(32) creation_time;
//		unsigned int(32) modification_time;
//		unsigned int(32) timescale;
//		unsigned int(32) duration;
//		}
//		template int(32) rate = 0x00010000; // typically 1.0
//		template int(16) volume = 0x0100; // typically, full volume
//		const bit(16) reserved = 0;
//		const unsigned int(32)[2] reserved = 0;
//		template int(32)[9] matrix =
//		{ 0x00010000,0,0,0,0x00010000,0,0,0,0x40000000 };
//		// Unity matrix
//		bit(32)[6] pre_defined = 0;
//		unsigned int(32) next_track_ID;
//		}
func (p *MP4Parser) parseMvhdAtom() error {
	// 读取版本
	version := readByte(p.file)
	// 跳过flags
	p.file.Seek(3, io.SeekCurrent)

	var creationTime, modificationTime, duration uint64
	var timescale uint32
	if version == 1 {
		creationTime = readU64(p.file)
		modificationTime = readU64(p.file)
		timescale = readU32(p.file)
		duration = readU64(p.file)
	} else {
		creationTime = uint64(readU32(p.file))
		modificationTime = uint64(readU32(p.file))
		timescale = readU32(p.file)
		duration = uint64(readU32(p.file))
	}
	fmt.Printf("DEBUG: mvhd, creationTime: %d, modificationTime: %d, timescale: %d, duration: %d\n",
		creationTime, modificationTime, timescale, duration)

	// 设置时间信息
	if timescale > 0 {
		durationSeconds := float64(duration) / float64(timescale)
		p.metadata.Duration = time.Duration(durationSeconds * float64(time.Second))
	}

	// 转换Mac时间戳到Go时间（Mac时间从1904-01-01开始）
	macEpoch := time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	p.metadata.CreationTime = macEpoch.Add(time.Duration(creationTime) * time.Second)
	p.metadata.ModificationTime = macEpoch.Add(time.Duration(modificationTime) * time.Second)

	// 4 + 2 + 2 + 8 + 36 + 24 + 4
	// skip rate, volume, reserved, matrix, pre_defined, next_track_ID
	p.file.Seek(80, io.SeekCurrent)

	return nil
}

// 解析trak atom
func (p *MP4Parser) parseTrakAtom(size uint32) error {
	fmt.Printf("DEBUG: parsing trak atom\n")
	end := cur(p.file) + int64(size)

	trackInfo := TrackInfo{}

	for cur(p.file) < end {
		atomSize, atomType, err := readHeader(p.file)
		if err != nil {
			fmt.Printf("error reading atom header: %v", err)
			return err
		}

		switch atomType {
		case BoxTypeTKHD:
			if err := p.parseTkhdAtom(&trackInfo); err != nil {
				return err
			}
		case BoxTypeMDIA:
			if err := p.parseMdiaAtom(atomSize-8, &trackInfo); err != nil {
				return err
			}
		default:
			p.file.Seek(int64(atomSize-8), io.SeekCurrent)
		}
	}

	// 添加到轨道列表
	p.tracks = append(p.tracks, trackInfo)

	return nil
}

// 解析tkhd atom
//
//	aligned(8) class TrackHeaderBox extends FullBox(‘tkhd’, version, flags) {
//	    if (version==1) {
//	        creation_time       8 bytes
//	        modification_time   8 bytes
//	        track_ID            4 bytes
//	        reserved            4 bytes
//	        duration            8 bytes
//	    } else { // version 0
//	        creation_time       4 bytes
//	        modification_time   4 bytes
//	        track_ID            4 bytes
//	        reserved            4 bytes
//	        duration            4 bytes
//	    }
//	    reserved                8 bytes
//	    layer                   2 bytes
//	    alternate_group         2 bytes
//	    volume                  2 bytes
//	    reserved                2 bytes
//	    matrix                  36 bytes
//	    width                   32-bit fixed-point 16.16
//	    height                  32-bit fixed-point 16.16
//	}
func (p *MP4Parser) parseTkhdAtom(track *TrackInfo) error {
	fmt.Printf("DEBUG: parsing tkhd atom\n")

	version := readByte(p.file)
	// 跳过flags
	p.file.Seek(3, io.SeekCurrent)

	var trackID uint32
	var duration uint64
	if version == 1 {
		// skip creationTime, modificationTime
		p.file.Seek(16, io.SeekCurrent)
		trackID = readU32(p.file)
		// skip reserved
		p.file.Seek(4, io.SeekCurrent)
		duration = readU64(p.file)
	} else {
		// skip creationTime, modificationTime
		p.file.Seek(8, io.SeekCurrent)
		trackID = readU32(p.file)
		// skip reserved
		p.file.Seek(4, io.SeekCurrent)
		duration = uint64(readU32(p.file))
	}
	// skip reserved, layer, alternate_group, volume, reserved
	p.file.Seek(16, io.SeekCurrent)
	// matrix (36 byte)
	p.file.Seek(36, io.SeekCurrent)

	width := readU32(p.file) >> 16
	height := readU32(p.file) >> 16

	track.TrackID = trackID
	track.Width = width
	track.Height = height
	track.Duration = duration
	fmt.Printf("DEBUG tkhd, trackID: %d, duration: %d, width: %d, height: %d\n",
		track.TrackID, track.Duration, track.Width, track.Height)

	return nil
}

// 解析mdia atom
func (p *MP4Parser) parseMdiaAtom(size uint32, track *TrackInfo) error {
	fmt.Printf("parsing mdia atom\n")
	end := cur(p.file) + int64(size)

	for cur(p.file) < end {
		atomSize, atomType, err := readHeader(p.file)
		if err != nil {
			fmt.Printf("error reading atom header: %v", err)
			return err
		}

		fmt.Printf("DEBUG: parsing %s atom, size: %d\n", atomType, atomSize)

		switch atomType {
		case "mdhd":
			if err := p.parseMdhdAtom(track); err != nil {
				return err
			}
		case "hdlr":
			if err := p.parseHdlrAtom(track, atomSize-8); err != nil {
				return err
			}
		case "minf":
			if err := p.parseMinfAtom(atomSize-8, track); err != nil {
				return err
			}
		default:
			p.file.Seek(int64(atomSize-8), io.SeekCurrent)
		}
	}

	return nil
}

// 解析mdhd atom
func (p *MP4Parser) parseMdhdAtom(track *TrackInfo) error {
	version := readByte(p.file)
	p.file.Seek(3, io.SeekCurrent)

	var timescale uint32
	var duration uint64
	if version == 1 {
		// skip creationTime, modificationTime
		p.file.Seek(16, io.SeekCurrent)
		timescale = readU32(p.file)
		duration = readU64(p.file)
	} else {
		// skip creationTime, modificationTime
		p.file.Seek(8, io.SeekCurrent)
		timescale = readU32(p.file)
		duration = uint64(readU32(p.file))
	}

	var language uint16
	if err := binary.Read(p.file, binary.BigEndian, &language); err != nil {
		return err
	}

	p.file.Seek(2, io.SeekCurrent)

	track.Timescale = timescale
	track.Duration = duration
	track.Language = decodeLanguage(language)

	return nil
}

func readBytes(f *os.File, n int) []byte {
	buf := make([]byte, n)
	f.Read(buf)
	return buf
}

// 解析hdlr atom
func (p *MP4Parser) parseHdlrAtom(track *TrackInfo, dataSize uint32) error {
	// skip version and flags
	// skip pre_defined
	p.file.Seek(8, io.SeekCurrent)
	handler := string(readBytes(p.file, 4))
	track.HandlerType = handler

	// 更新metadata中的轨道类型
	switch handler {
	case "vide":
		p.metadata.HasVideo = true
	case "soun":
		p.metadata.HasAudio = true
	case "hint":
		p.metadata.HasHint = true
	case "meta":
		p.metadata.HasMeta = true
	case "auxv":
		p.metadata.HasAuxv = true
	}

	p.file.Seek(12, io.SeekCurrent)
	left := dataSize - 12 - 8 - 4
	p.file.Seek(int64(left), io.SeekCurrent)

	return nil
}

// 解析minf atom
func (p *MP4Parser) parseMinfAtom(size uint32, track *TrackInfo) error {
	end := cur(p.file) + int64(size)
	for cur(p.file) < end {
		atomSize, atomType, err := readHeader(p.file)
		if err != nil {
			fmt.Printf("error reading atom header: %v", err)
			return err
		}

		switch atomType {
		case "stbl":
			if err := p.parseStblAtom(size, track); err != nil {
				return err
			}
		default:
			p.file.Seek(int64(atomSize-8), io.SeekCurrent)
		}

	}

	return nil
}

// 解析stbl atom
func (p *MP4Parser) parseStblAtom(size uint32, track *TrackInfo) error {
	end := cur(p.file) + int64(size)
	for cur(p.file) < end {
		atomSize, atomType, err := readHeader(p.file)
		if err != nil {
			fmt.Printf("error reading atom header: %v\n", err)
			return err
		}
		fmt.Printf("DEBUG: parsing %s atom, size: %d\n", atomType, atomSize)
		switch atomType {
		case "stsd":
			if err := parseStsd(p.file, atomSize, track); err != nil {
				return err
			}
		case "stts":
			if err := parseStts(p.file, track); err != nil {
				return err
			}
		case "ctts":
			if err := parseCtts(p.file, track); err != nil {
				return err
			}
		default:
			p.file.Seek(int64(atomSize-8), io.SeekCurrent)
		}
	}

	return nil
}

// 计算元数据
func (p *MP4Parser) calculateMetadata() error {
	for _, track := range p.tracks {
		switch track.HandlerType {
		case "vide":
			p.metadata.HasVideo = true
			p.metadata.Width = track.Width
			p.metadata.Height = track.Height
			p.metadata.VideoCodec = track.Codec

			// 计算帧率
			if track.Timescale > 0 && track.FrameCount > 0 {
				durationSeconds := float64(track.Duration) / float64(track.Timescale)
				if durationSeconds > 0 {
					p.metadata.FPS = float64(track.FrameCount) / durationSeconds
				}
			}
		case "soun":
			p.metadata.HasAudio = true
			p.metadata.AudioCodec = track.Codec
		}
		if len(track.SttsBox.Entries) > 0 {
			// dtsLines := buildDTSTimeline(track.SttsBox.Entries)
			// ptsLines := buildPTSTimeline(dtsLines, track.CttsBox.Entries)

			// issues := DetectDiscontinuities(int64(track.TrackID), dtsLines, ptsLines)
			// for _, issue := range issues {
			// 	fmt.Printf("WARNING: %s\n", issue)
			// }
		}
	}

	return nil
}

// 获取详细的轨道信息
func (p *MP4Parser) GetTracks() []TrackInfo {
	return p.tracks
}
