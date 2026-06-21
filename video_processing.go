package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
)

type ffprobeOutput struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"

	cmd := exec.Command(
		"ffmpeg",
		"-i", filePath,
		"-c", "copy",
		"-movflags", "faststart",
		"-f", "mp4",
		outputPath,
	)

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		filePath,
	)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	var output ffprobeOutput

	err = json.Unmarshal(stdout.Bytes(), &output)
	if err != nil {
		return "", err
	}

	width := output.Streams[0].Width
	height := output.Streams[0].Height

	ratio := float64(width) / float64(height)

	if ratio > 1.7 && ratio < 1.8 {
		return "landscape", nil
	}

	if ratio > 0.5 && ratio < 0.6 {
		return "portrait", nil
	}

	return "other", nil
}