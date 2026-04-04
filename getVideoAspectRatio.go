package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {

	// args for ffprobe
	args := []string{
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		filePath,
	}

	cmd := exec.Command("ffprobe", args...)

	// create buffers to capture standard output and standard error
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// run the command
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	// unmarshal json out into our structs
	var output struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return "", err
	}
	if len(output.Streams) == 0 {
		return "", errors.New("No Video")
	}

	outputWidth := float64(output.Streams[0].Width)
	outputHeight := float64(output.Streams[0].Height)

	if math.Abs(outputWidth/outputHeight-(16.0/9.0)) < 0.01 {
		return "16:9", nil
	} else if math.Abs(outputWidth/outputHeight-(9.0/16.0)) < 0.01 {
		return "9:16", nil
	} else {
		return "other", nil
	}

}
