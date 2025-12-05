package mp4

import (
	"fmt"
	"time"
)

// 工具函数

// 格式化时长
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

// 格式化文件大小
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

// 打印元数据
func PrintMetadata(metadata *MP4Metadata) {
	fmt.Println("=== MP4 文件元信息 ===")
	fmt.Printf("时长: %s\n", FormatDuration(metadata.Duration))
	fmt.Printf("尺寸: %d × %d\n", metadata.Width, metadata.Height)
	fmt.Printf("帧率: %.2f fps\n", metadata.FPS)
	fmt.Printf("视频编码: %s\n", metadata.VideoCodec)
	fmt.Printf("音频编码: %s\n", metadata.AudioCodec)

	if metadata.HasVideo && metadata.HasAudio {
		fmt.Println("轨道: 视频 + 音频")
	} else if metadata.HasVideo {
		fmt.Println("轨道: 仅视频")
	} else if metadata.HasAudio {
		fmt.Println("轨道: 仅音频")
	}

	fmt.Printf("创建时间: %s\n", metadata.CreationTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("修改时间: %s\n", metadata.ModificationTime.Format("2006-01-02 15:04:05"))

	if metadata.VideoBitrate > 0 {
		fmt.Printf("视频码率: %.2f Mbps\n", float64(metadata.VideoBitrate)/1000000)
	}

	if metadata.AudioBitrate > 0 {
		fmt.Printf("音频码率: %.2f Kbps\n", float64(metadata.AudioBitrate)/1000)
	}

	if metadata.Rotation != 0 {
		fmt.Printf("旋转角度: %d°\n", metadata.Rotation)
	}
}
