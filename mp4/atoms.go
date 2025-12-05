package mp4

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// 解析不同的Atom类型
const (
	BoxTypeFTYP = "ftyp"
	BoxTypeMOOV = "moov"
	BoxTypeMVHD = "mvhd"
	BoxTypeTRAK = "trak"
	BoxTypeTKHD = "tkhd"
	BoxTypeMDIA = "mdia"
	BoxTypeMDHD = "mdhd"
	BoxTypeHDLR = "hdlr"
	BoxTypeMINF = "minf"
	BoxTypeSTBL = "stbl"
	BoxTypeSTSD = "stsd"
	BoxTypeSTTS = "stts"
	BoxTypeSTSZ = "stsz"
	BoxTypeSTSC = "stsc"
	BoxTypeSTCO = "stco"
	BoxTypeCO64 = "co64"
	BoxTypeCTTS = "ctts"
	BoxTypeSTSS = "stss"
	BoxTypeVMHD = "vmhd"
	BoxTypeSMHD = "smhd"
	BoxTypeDINF = "dinf"
	BoxTypeFREE = "free"
	BoxTypeSKIP = "skip"
	BoxTypeUDTA = "udta"
	BoxTypeMDAT = "mdat"
	BoxTypeIODS = "iods"
)

// parseStsd parses a stsd box from the current position of an io.ReadSeeker.
// The caller should position the reader at the beginning of the stsd box payload
// (i.e. right after the 8-byte box header). stsdSize is the full box size including
// the 8-byte header. For typical usage you pass the size you read from the box header.
//
// The function strictly follows ISO BMFF: it reads version/flags, entry_count and then
// iterates over each sample entry. For unknown/unsupported child boxes it will skip
// them based on their declared size so the stream position remains correct.
// aligned(8) class SampleDescriptionBox (unsigned int(32) handler_type)
//
//		extends FullBox('stsd', 0, 0){
//			int i ;
//			unsigned int(32) entry_count;
//			for (i = 1 ; i <= entry_count ; i++){
//	         switch (handler_type){
//			        case ‘soun’: // for audio tracks
//			            AudioSampleEntry();
//			        	break;
//			        case ‘vide’: // for video tracks
//			        	VisualSampleEntry();
//			        	break;
//			        case ‘hint’: // Hint track
//			        	HintSampleEntry();
//			        	break;
//			        case ‘meta’: // Metadata track
//			        	MetadataSampleEntry();
//			        	break;
//			        }
//			    }
//			}
//	}
func parseStsd(rs io.ReadSeeker, stsdSize uint32, info *TrackInfo) error {
	if rs == nil || info == nil {
		return errors.New("nil argument")
	}

	startPos, err := rs.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	// stsd payload length = stsdSize - 8 (box header consumed by caller usually),
	// but here we assume caller provided stsdSize and that we are at payload start.
	payloadRemaining := int64(stsdSize - 8)

	// read version(1) + flags(3)
	var verFlags [4]byte
	if _, err := io.ReadFull(rs, verFlags[:]); err != nil {
		return fmt.Errorf("read stsd version/flags: %w", err)
	}
	payloadRemaining -= 4

	// read entry_count (uint32)
	var entryCount uint32
	if err := binary.Read(rs, binary.BigEndian, &entryCount); err != nil {
		return fmt.Errorf("read stsd entry_count: %w", err)
	}
	payloadRemaining -= 4

	for i := uint32(0); i < entryCount; i++ {
		if payloadRemaining <= 0 {
			return fmt.Errorf("stsd: payload exhausted before reading all entries (i=%d)", i)
		}

		entryStart, _ := rs.Seek(0, io.SeekCurrent)

		// each sample entry starts with uint32 size and 4-byte format
		// SampleEntry具有类似Box的结构，比如拥有size
		// aligned(8) abstract class SampleEntry (unsigned int(32) format)
		//     extends Box(format){
		//     const unsigned int(8)[6] reserved = 0;
		// 	   unsigned int(16) data_reference_index;
		//}
		var entrySize uint32
		if err := binary.Read(rs, binary.BigEndian, &entrySize); err != nil {
			return fmt.Errorf("read sample entry size: %w", err)
		}

		var format [4]byte
		if _, err := io.ReadFull(rs, format[:]); err != nil {
			return fmt.Errorf("read sample entry format: %w", err)
		}

		payloadRemaining -= int64(8)
		entryPayloadSize := int64(entrySize) - 8 // remaining bytes inside this sample entry

		if entrySize < 8 {
			return fmt.Errorf("invalid sample entry size %d", entrySize)
		}

		entryType := string(format[:])
		// Set codec hint
		if info.Codec == "" {
			info.Codec = entryType
		}

		// Parse common sample entry header for both audio and video.
		// According to ISO BMFF, sample entry begins with:
		//   6 bytes reserved (0)
		//   2 bytes data_reference_index
		// For visual sample entries (e.g., avc1) there are additional 16 bytes
		// of pre_defined/reserved and then width/height etc.

		// moov (Movie Box)
		// ├── trak (Track Box)
		//    ├── mdia (Media Box)
		//       ├── minf (Media Information Box)
		//           ├── stbl (Sample Table Box)
		//               ├── stsd (Sample Description Box) ← SampleEntry在这里

		// SampleEntry不是一种单独的box类型，它只是一种内嵌在stsd中的数据结构

		// read 6 reserved + 2 data_reference_index
		if entryPayloadSize < 8 {
			// malformed
			// seek to end of entry and continue
			if _, err := rs.Seek(entryStart+int64(entrySize), io.SeekStart); err != nil {
				return err
			}
			payloadRemaining -= int64(entrySize - 8)
			continue
		}

		var reserved [6]byte
		if _, err := io.ReadFull(rs, reserved[:]); err != nil {
			return fmt.Errorf("read sample entry reserved: %w", err)
		}
		var dataRef uint16
		if err := binary.Read(rs, binary.BigEndian, &dataRef); err != nil {
			return fmt.Errorf("read data_reference_index: %w", err)
		}
		entryPayloadSize -= 8
		payloadRemaining -= 8

		switch entryType {
		case "avc1", "encv", "avc3":
			// Visual sample entry: parse fields described in ISO/IEC 14496-12
			// next 16 bytes: pre_defined (2), reserved (2), pre_defined[3] (12)
			if entryPayloadSize < 16 {
				// skip this entry
				if _, err := rs.Seek(entryStart+int64(entrySize), io.SeekStart); err != nil {
					return err
				}
				payloadRemaining -= int64(entrySize - 8)
				continue
			}
			// skip 16 bytes
			// skip pre_defined(2bytes), reserved(2bytes) and pre_defined(12bytes)
			if _, err := rs.Seek(16, io.SeekCurrent); err != nil {
				return fmt.Errorf("seek pre_defined: %w", err)
			}
			entryPayloadSize -= 16
			payloadRemaining -= 16

			// read width & height (uint16 each)
			var width, height uint16
			if err := binary.Read(rs, binary.BigEndian, &width); err != nil {
				return fmt.Errorf("read width: %w", err)
			}
			if err := binary.Read(rs, binary.BigEndian, &height); err != nil {
				return fmt.Errorf("read height: %w", err)
			}
			entryPayloadSize -= 4
			payloadRemaining -= 4
			info.Width = uint32(width)
			info.Height = uint32(height)

			// skip horiz/vert resolution (8 bytes), reserved (4), frame_count(2), compressorname(32), depth(2), pre-defined(2)
			skipBytes := int64(8 + 4 + 2 + 32 + 2 + 2)
			if entryPayloadSize < skipBytes {
				// try to skip what's left and then handle children by size
				if _, err := rs.Seek(entryStart+int64(entrySize), io.SeekStart); err != nil {
					return err
				}
				payloadRemaining -= int64(entrySize - 8)
				continue
			}
			if _, err := rs.Seek(skipBytes, io.SeekCurrent); err != nil {
				return fmt.Errorf("seek visual tail: %w", err)
			}
			entryPayloadSize -= skipBytes
			payloadRemaining -= skipBytes

			// Now inside the sample entry there may be child boxes (avcC, btrt, pasp ...)
			// parse them by reading child box headers until we've consumed entryPayloadSize
			for entryPayloadSize > 0 {
				// read child box header
				var childSize uint32
				var childType [4]byte
				if err := binary.Read(rs, binary.BigEndian, &childSize); err != nil {
					return fmt.Errorf("read child size: %w", err)
				}
				if _, err := io.ReadFull(rs, childType[:]); err != nil {
					return fmt.Errorf("read child type: %w", err)
				}
				if childSize == 0 {
					// child extends to end of parent
					childSize = uint32(entryPayloadSize)
				}
				childBodySize := int64(childSize - 8)
				entryPayloadSize -= int64(childSize)
				payloadRemaining -= int64(childSize)

				switch string(childType[:]) {
				case "avcC":
					// read avcC box into info.AVCConfig
					buf := make([]byte, childBodySize)
					if _, err := io.ReadFull(rs, buf); err != nil {
						return fmt.Errorf("read avcC: %w", err)
					}
					// info.AVCConfig = buf
				default:
					// skip unknown child
					if _, err := rs.Seek(childBodySize, io.SeekCurrent); err != nil {
						return fmt.Errorf("skip child %s: %w", string(childType[:]), err)
					}
				}
			}

		case "mp4a", "enca":
			// Audio sample entry
			// after data_reference_index there's 8 bytes reserved, then:
			// channelcount(2), samplesize(2), pre_defined(2), reserved(2), samplerate(32 as 16.16)
			if entryPayloadSize < 8+2+2+2+2+4 {
				if _, err := rs.Seek(entryStart+int64(entrySize), io.SeekStart); err != nil {
					return err
				}
				payloadRemaining -= int64(entrySize - 8)
				continue
			}
			// skip 8 reserved
			if _, err := rs.Seek(8, io.SeekCurrent); err != nil {
				return err
			}
			entryPayloadSize -= 8
			payloadRemaining -= 8

			var channelCount uint16
			var sampleSize uint16
			if err := binary.Read(rs, binary.BigEndian, &channelCount); err != nil {
				return fmt.Errorf("read channelCount: %w", err)
			}
			if err := binary.Read(rs, binary.BigEndian, &sampleSize); err != nil {
				return fmt.Errorf("read sampleSize: %w", err)
			}
			entryPayloadSize -= 4
			payloadRemaining -= 4

			// skip pre_defined + reserved
			if _, err := rs.Seek(4, io.SeekCurrent); err != nil {
				return err
			}
			entryPayloadSize -= 4
			payloadRemaining -= 4

			// read sample rate (32-bit 16.16 fixed)
			var sampleRate uint32
			if err := binary.Read(rs, binary.BigEndian, &sampleRate); err != nil {
				return fmt.Errorf("read sampleRate: %w", err)
			}
			entryPayloadSize -= 4
			payloadRemaining -= 4

			// parse child boxes inside audio sample entry
			for entryPayloadSize > 0 {
				var childSize uint32
				var childType [4]byte
				if err := binary.Read(rs, binary.BigEndian, &childSize); err != nil {
					return err
				}
				if _, err := io.ReadFull(rs, childType[:]); err != nil {
					return err
				}
				if childSize == 0 {
					childSize = uint32(entryPayloadSize)
				}
				childBodySize := int64(childSize - 8)
				entryPayloadSize -= int64(childSize)
				payloadRemaining -= int64(childSize)

				switch string(childType[:]) {
				case "esds":
					buf := make([]byte, childBodySize)
					if _, err := io.ReadFull(rs, buf); err != nil {
						return fmt.Errorf("read esds: %w", err)
					}
					// info.ESDS = buf
				default:
					if _, err := rs.Seek(childBodySize, io.SeekCurrent); err != nil {
						return err
					}
				}
			}

		default:
			// Unknown sample entry type: skip its payload
			if _, err := rs.Seek(entryPayloadSize, io.SeekCurrent); err != nil {
				return fmt.Errorf("skip unknown sample entry %s: %w", entryType, err)
			}
			payloadRemaining -= entryPayloadSize
		}

		// ensure we are positioned at the end of the sample entry exactly
		endPos := entryStart + int64(entrySize)
		if cur, _ := rs.Seek(0, io.SeekCurrent); cur != endPos {
			if _, err := rs.Seek(endPos, io.SeekStart); err != nil {
				return fmt.Errorf("seek to end of entry: %w", err)
			}
		}

	}

	// final sanity: ensure we haven't read beyond stsd box
	curPos, _ := rs.Seek(0, io.SeekCurrent)
	if curPos-startPos > int64(stsdSize-8) {
		return fmt.Errorf("parsed past stsd payload: consumed=%d payload=%d", curPos-startPos, stsdSize-8)
	}

	return nil
}

// 解析stts (Decoding Time to Sample) Atom
func parseStts(r io.Reader, trackInfo *TrackInfo) error {
	io.CopyN(io.Discard, r, 4) // 跳过version和flags

	var entryCount uint32
	if err := binary.Read(r, binary.BigEndian, &entryCount); err != nil {
		return err
	}

	var totalSampleCount uint32
	var totalDuration uint32

	entries := make([]TimeToSampleEntry, entryCount)
	for i := uint32(0); i < entryCount; i++ {
		var sampleCount, sampleDelta uint32
		if err := binary.Read(r, binary.BigEndian, &sampleCount); err != nil {
			return err
		}
		entries[i] = TimeToSampleEntry{
			Count: sampleCount,
			Delta: sampleDelta,
		}
		if err := binary.Read(r, binary.BigEndian, &sampleDelta); err != nil {
			return err
		}

		totalSampleCount += sampleCount
		totalDuration += sampleCount * sampleDelta
	}

	// 计算帧率（如果可能）
	if totalDuration > 0 && trackInfo.Timescale > 0 {
		trackInfo.FrameCount = totalSampleCount
		trackInfo.SampleCount = totalSampleCount
	}
	trackInfo.SttsBox = &sttsBox{
		Entries: entries,
	}

	return nil
}

// 解码语言代码
func decodeLanguage(lang uint16) string {
	if lang == 0 {
		return "und"
	}

	var language [3]byte
	language[0] = byte((lang>>10)&0x1F) + 0x60
	language[1] = byte((lang>>5)&0x1F) + 0x60
	language[2] = byte(lang&0x1F) + 0x60

	return string(language[:])
}

// Box结构定义
type MVHDBox struct {
	CreationTime     uint64
	ModificationTime uint64
	Timescale        uint32
	Duration         uint64
	Rate             uint32   // 播放速率（16.16定点数）
	Volume           uint16   // 音量（8.8定点数）
	Matrix           [9]int32 // 变换矩阵
	NextTrackID      uint32
}

type TKHDBox struct {
	TrackID  uint32
	Width    uint32
	Height   uint32
	Duration uint64
}

type HDLRBox struct {
	HandlerType string
}

type MDHDBox struct {
	Timescale uint32
	Duration  uint64
	Language  string
}

// --- Added: MP4 timestamp continuity analyzer ---
type SampleTime struct {
	DTS uint64
	PTS uint64
}

type TimeTables struct {
	STTS []TimeToSampleEntry
	CTTS []CompositionOffsetEntry
}

type TimeToSampleEntry struct {
	Count uint32
	Delta uint32
}

type cttsBox struct {
	Entries []CompositionOffsetEntry
}

type CompositionOffsetEntry struct {
	Count  uint32
	Offset int32
}

// Parse stts box
type sttsBox struct {
	Entries []TimeToSampleEntry
}

// Parse ctts box (if exists)
func parseCtts(r io.Reader, info *TrackInfo) error {
	var versionFlags uint32
	binary.Read(r, binary.BigEndian, &versionFlags)
	version := versionFlags >> 24
	var entryCount uint32
	binary.Read(r, binary.BigEndian, &entryCount)
	items := make([]CompositionOffsetEntry, entryCount)
	for i := 0; i < int(entryCount); i++ {
		binary.Read(r, binary.BigEndian, &items[i].Count)
		if version == 0 {
			var offset uint32
			binary.Read(r, binary.BigEndian, &offset)
			items[i].Offset = int32(offset)
		} else {
			binary.Read(r, binary.BigEndian, &items[i].Offset)
		}
	}
	if info != nil {
		info.CttsBox = &cttsBox{
			Entries: items,
		}
	}
	return nil
}

// Build DTS timeline
func buildDTSTimeline(stts []TimeToSampleEntry) []uint64 {
	var dtsList []uint64
	current := uint64(0)
	for _, e := range stts {
		for i := 0; i < int(e.Count); i++ {
			dtsList = append(dtsList, current)
			current += uint64(e.Delta)
		}
	}
	return dtsList
}

// Build PTS timeline
func buildPTSTimeline(dts []uint64, ctts []CompositionOffsetEntry) []uint64 {
	if len(ctts) == 0 {
		return dts
	}
	pts := make([]uint64, len(dts))
	idx := 0
	for _, e := range ctts {
		for i := 0; i < int(e.Count); i++ {
			offset := int64(e.Offset)
			if offset < 0 {
				pts[idx] = uint64(int64(dts[idx]) + offset)
			} else {
				pts[idx] = dts[idx] + uint64(offset)
			}
			idx++
		}
	}
	return pts
}

// Detect timestamp discontinuities
func DetectDiscontinuities(trackID int64, dts, pts []uint64) []string {
	var issues []string
	for i := 1; i < len(dts); i++ {
		if dts[i] <= dts[i-1] {
			issues = append(issues, fmt.Sprintf("track id: %d, DTS backward at sample %d: %d -> %d", trackID, i, dts[i-1], dts[i]))
		}
	}
	for i := 1; i < len(pts); i++ {
		if pts[i] < pts[i-1] {
			issues = append(issues, fmt.Sprintf("track id: %d, PTS backward at sample %d: %d -> %d", trackID, i, pts[i-1], pts[i]))
		}
	}
	return issues
}
