package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func processVideoForFastStart(filePath string) (string, error) {
	outputFilePath := filePath + ".processing"

	// args for ffprobe
	args := []string{
		"-i", filePath,
		"-movflags", "faststart",
		"-c", "copy",
		"-f", "mp4",
		outputFilePath,
	}

	cmd := exec.Command("ffmpeg", args...)

	// create buffers to capture standard output and standard error
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// run the command
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error processing video: %s, %v", stderr.String(), err)
	}

	fileInfo, err := os.Stat(outputFilePath)
	if err != nil {
		return "", fmt.Errorf("could not stat processed file: %v", err)
	}
	if fileInfo.Size() == 0 {
		return "", fmt.Errorf("processed file is empty")
	}

	return outputFilePath, nil
}
