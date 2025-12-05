package mp4

import (
	"fmt"
	"time"
)

func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	milliseconds := int(d.Milliseconds()) % 1000

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, milliseconds)
	}
	return fmt.Sprintf("%02d:%02d.%03d", minutes, seconds, milliseconds)
}

func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func PrintMetadata(metadata *MP4Metadata) {
	fmt.Println("=== MP4 file metadata ===")
	fmt.Printf("Duration: %s\n", FormatDuration(metadata.Duration))
	fmt.Printf("Resolution: %d × %d\n", metadata.Width, metadata.Height)
	fmt.Printf("FPS: %.2f fps\n", metadata.FPS)
	fmt.Printf("Video Codec: %s\n", metadata.VideoCodec)
	fmt.Printf("Audio Codec: %s\n", metadata.AudioCodec)

	if metadata.HasVideo && metadata.HasAudio {
		fmt.Println("Tracks: video + audio")
	} else if metadata.HasVideo {
		fmt.Println("Track: only video")
	} else if metadata.HasAudio {
		fmt.Println("Track: only audio")
	}

	fmt.Printf("CreateTime: %s\n", metadata.CreationTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("ModificationTime: %s\n", metadata.ModificationTime.Format("2006-01-02 15:04:05"))

	if metadata.VideoBitrate > 0 {
		fmt.Printf("Video Bitrate: %.2f Mbps\n", float64(metadata.VideoBitrate)/1000000)
	}

	if metadata.AudioBitrate > 0 {
		fmt.Printf("Audio Bitrate: %.2f Kbps\n", float64(metadata.AudioBitrate)/1000)
	}

	if metadata.Rotation != 0 {
		fmt.Printf("Rotation: %d°\n", metadata.Rotation)
	}
}
