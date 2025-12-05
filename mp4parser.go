package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"mp4parser/mp4"
)

func main() {
	filename := flag.String("f", "", "MP4 file path")
	verbose := flag.Bool("v", false, "Display detailed track information")
	flag.Parse()

	if *filename == "" {
		fmt.Println("usage: mp4parser -f <file_name> [-v]")
		fmt.Println("example: mp4parser -f video.mp4")
		return
	}

	if _, err := os.Stat(*filename); os.IsNotExist(err) {
		fmt.Printf("err: file '%s' does not exist\n", *filename)
		return
	}

	// create a parser
	parser, err := mp4.NewParser(*filename)
	if err != nil {
		fmt.Printf("create parser failed: %v\n", err)
		return
	}

	// start to parse file
	metadata, err := parser.Parse()
	if err != nil {
		fmt.Printf("parse file failed: %v\n", err)
		return
	}

	// get file information
	fileInfo, _ := os.Stat(*filename)
	fmt.Printf("file: %s\n", filepath.Base(*filename))
	fmt.Printf("size: %s\n\n", mp4.FormatFileSize(fileInfo.Size()))

	// print metadata
	mp4.PrintMetadata(metadata)

	// if verbose mode is enabled, print track information
	if *verbose {
		fmt.Println("\n=== detailed track information ===")
		tracks := parser.GetTracks()
		for i, track := range tracks {
			fmt.Printf("\ntrack %d:\n", i+1)
			fmt.Printf("  ID: %d\n", track.TrackID)
			fmt.Printf("  Type: %s\n", track.HandlerType)
			fmt.Printf("  Codec: %s\n", track.Codec)

			if track.HandlerType == "vide" {
				fmt.Printf("  Resolution: %d × %d\n", track.Width, track.Height)
			}

			if track.Timescale > 0 {
				duration := float64(track.Duration) / float64(track.Timescale)
				fmt.Printf("  Duration: %.2f seconds\n", duration)
			}

			if track.FrameCount > 0 {
				fmt.Printf("  FrameCount: %d\n", track.FrameCount)
			}

			if track.Language != "" && track.Language != "und" {
				fmt.Printf("  Language: %s\n", track.Language)
			}
		}
	}
}
