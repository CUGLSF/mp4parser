package main

import (
	"fmt"
	"mp4parser/mp4"
)

func main() {
	// 解析MP4文件
	parser, err := mp4.NewParser("sample.mp4")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	// 获取元数据
	metadata, err := parser.Parse()
	if err != nil {
		fmt.Printf("解析错误: %v\n", err)
		return
	}

	// 使用自定义格式显示
	fmt.Println("视频信息:")
	fmt.Println("--------")
	fmt.Printf("分辨率:  %d × %d\n", metadata.Width, metadata.Height)
	fmt.Printf("时长:    %s\n", mp4.FormatDuration(metadata.Duration))
	fmt.Printf("帧率:    %.2f fps\n", metadata.FPS)
	fmt.Printf("视频编码: %s\n", metadata.VideoCodec)

	if metadata.HasAudio {
		fmt.Printf("音频编码: %s\n", metadata.AudioCodec)
	}

	// 检查是否为竖屏视频
	if metadata.Rotation == 90 || metadata.Rotation == 270 {
		fmt.Println("方向:    竖屏")
	} else {
		fmt.Println("方向:    横屏")
	}

	// 获取所有轨道信息
	tracks := parser.GetTracks()
	fmt.Printf("\n共 %d 个轨道:\n", len(tracks))

	for i, track := range tracks {
		fmt.Printf("  轨道 %d: %s (%s)",
			i+1,
			getTrackType(track.HandlerType),
			track.Codec)

		if track.HandlerType == "vide" {
			fmt.Printf(" - %d×%d", track.Width, track.Height)
		}
		fmt.Println()
	}
}

func getTrackType(handlerType string) string {
	switch handlerType {
	case "vide":
		return "视频"
	case "soun":
		return "音频"
	case "hint":
		return "提示"
	case "text":
		return "文本"
	case "subt":
		return "字幕"
	default:
		return handlerType
	}
}
