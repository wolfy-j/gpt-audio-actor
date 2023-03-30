package listener

import (
	"bytes"
	"fmt"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"os"
	"os/exec"
	"regexp"
)

type Transcriber interface {
	// TranscribeBytes transcribes audio data from pcmBuffer and returns the result.
	// Expects int16 format (low, high), 16kHz, mono.
	TranscribeBytes(pcmBuffer bytes.Buffer) (string, error)
}

// CLITranscriber is a simple transcriber using Whisper CLI
type CLITranscriber struct {
	// Path to Whisper CLI executable
	whisperPath string

	// Model name
	model string
}

// NewCLITranscriber creates a new CLITranscriber
func NewCLITranscriber(whisperPath, model string) *CLITranscriber {
	return &CLITranscriber{
		whisperPath: whisperPath,
		model:       model,
	}
}

func (wsp *CLITranscriber) TranscribeBytes(pcmBuffer bytes.Buffer) (string, error) {
	// working with temporary file
	file, err := os.CreateTemp(os.TempDir(), "whisper-*")
	if err != nil {
		return "", fmt.Errorf("unable to create temp file: %s", err)
	}
	defer os.Remove(file.Name())

	// write PCM data to the file
	err = writePCM(file, pcmBuffer.Bytes())
	if err != nil {
		return "", fmt.Errorf("unable to write PCM data to temp file: %s", err)
	}
	file.Close()

	cmd := exec.Command(wsp.whisperPath, "-m", wsp.model, file.Name())
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return extractTranscript(outputBytes)
}

func writePCM(file *os.File, pcmData []byte) error {
	enc := wav.NewEncoder(
		file,
		16000, // sample rate
		16,    // bits per sample
		1,     // number of channels
		0x01,  // format
	)

	n := len(pcmData)
	intBuffer := make([]int, n/2)
	for i := 0; i < n; i += 2 {
		intBuffer[i/2] = int(pcmData[i]) | int(pcmData[i+1])<<8
	}

	err := enc.Write(&audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: 1,
			SampleRate:  8000,
		},
		Data:           intBuffer,
		SourceBitDepth: 16,
	})
	enc.Close()

	return err
}

// Extracts transcript from the output of Whisper CLI
func extractTranscript(output []byte) (string, error) {
	r := regexp.MustCompile(`\[\d\d:\d\d:\d\d\.\d\d\d --> \d\d:\d\d:\d\d\.\d\d\d\]\s+(.*)`)
	matches := r.FindSubmatch(output)

	if len(matches) < 2 {
		return "", fmt.Errorf("unable to find transcript in output")
	}

	return string(matches[1]), nil
}
