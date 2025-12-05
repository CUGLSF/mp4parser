package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"mp4parser/mp4"
)

func main() {
	// 解析命令行参数
	filename := flag.String("f", "", "MP4文件路径")
	verbose := flag.Bool("v", false, "显示详细信息")
	flag.Parse()

	if *filename == "" {
		fmt.Println("用法: mp4parser -f <文件名> [-v]")
		fmt.Println("示例: mp4parser -f video.mp4")
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(*filename); os.IsNotExist(err) {
		fmt.Printf("错误: 文件 '%s' 不存在\n", *filename)
		return
	}

	// 创建解析器
	parser, err := mp4.NewParser(*filename)
	if err != nil {
		fmt.Printf("创建解析器失败: %v\n", err)
		return
	}

	// 解析文件
	metadata, err := parser.Parse()
	if err != nil {
		fmt.Printf("解析文件失败: %v\n", err)
		return
	}

	// 获取文件信息
	fileInfo, _ := os.Stat(*filename)
	fmt.Printf("文件: %s\n", filepath.Base(*filename))
	fmt.Printf("大小: %s\n\n", mp4.FormatFileSize(fileInfo.Size()))

	// 显示基本元信息
	mp4.PrintMetadata(metadata)

	// 如果启用详细模式，显示轨道信息
	if *verbose {
		fmt.Println("\n=== 详细轨道信息 ===")
		tracks := parser.GetTracks()
		for i, track := range tracks {
			fmt.Printf("\n轨道 %d:\n", i+1)
			fmt.Printf("  ID: %d\n", track.TrackID)
			fmt.Printf("  类型: %s\n", track.HandlerType)
			fmt.Printf("  编码: %s\n", track.Codec)

			if track.HandlerType == "vide" {
				fmt.Printf("  尺寸: %d × %d\n", track.Width, track.Height)
			}

			if track.Timescale > 0 {
				duration := float64(track.Duration) / float64(track.Timescale)
				fmt.Printf("  时长: %.2f 秒\n", duration)
			}

			if track.FrameCount > 0 {
				fmt.Printf("  帧数: %d\n", track.FrameCount)
			}

			if track.Language != "" && track.Language != "und" {
				fmt.Printf("  语言: %s\n", track.Language)
			}
		}
	}
}
