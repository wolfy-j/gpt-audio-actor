package main

import (
	"context"
	"fmt"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/sashabaranov/go-openai"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

func extractJSON(input string) (string, error) {
	re := regexp.MustCompile(`\[\s*(.*?)\s*\]`)
	match := re.FindStringSubmatch(input)
	if len(match) < 2 {
		return "", fmt.Errorf("no JSON object found in input string: %s", input)
	}

	return "[" + match[1] + "]", nil
}

func extractJSON2(s string) (string, error) {
	start := strings.Index(s, "[{")
	if start == -1 {
		// The starting substring wasn't found.
		return "", fmt.Errorf("no JSON object found in input string: %s", s)
	}

	end := strings.Index(s[start+2:], "}]")
	if end == -1 {
		// The ending substring wasn't found.
		return "", fmt.Errorf("no JSON object found in input string: %s", s)
	}

	// Return the extracted substring, including the start and end substrings.
	return s[start : start+end+4], nil
}

func main() {
	fmt.Println("Starting server...")

	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: 5001,
	})

	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer conn.Close()

	openaiClient := openai.NewClient(os.Getenv("OPENAI_KEY"))

	//	now := time.Now()
	//	resp, err := openaiClient.CreateChatCompletion(
	//		context.Background(),
	//		openai.ChatCompletionRequest{
	//			Model: openai.GPT3Dot5Turbo,
	//			Temperature: 0,
	//			Messages: []openai.ChatCompletionMessage{
	////				{
	////					Role:    openai.ChatMessageRoleSystem,
	////					Content: "Update LED status JSON according to user request and return it as response [{'red':0,'green':0,'blue':0},{'red':0,'green':0,'blue':0},{'red':0,'green':0,'blue':0},{'red':0,'green':0,'blue':0},{'red':0,'green':0,'blue':0}]",
	//				//},
	//
	//				{
	//					Role:    openai.ChatMessageRoleUser,
	//					Content: "Update LED status JSON according to user request and return it as response [{'red':0,'green':0,'blue':0},{'red':0,'green':0,'blue':0},{'red':0,'green':0,'blue':0},{'red':0,'green':0,'blue':0},{'red':0,'green':0,'blue':0}], RGB values are 0-255, only return JSON. user request: 'set to rainbow'",
	//				},
	//			},
	//		},
	//	)
	//	fmt.Println(resp.Choices[0].Message.Content)
	//	fmt.Println(time.Now().Sub(now))

	fmt.Println("Server listening")

	buffer := make([]byte, 1024)

	var (
		mu             sync.Mutex
		currentFile    *os.File
		currentAddr    *net.UDPAddr
		currentEncoder *wav.Encoder
		fileN          int
		lastState      string
	)

	lastState = "[{'red':0,'green':0,'blue':0},{'red':0,'green':0,'blue':0},{'red':0,'green':0,'blue':0},{'red':0,'green':0,'blue':0},{'red':0,'green':0,'blue':0}]"

	resetTimer := make(chan struct{})
	go func() {
		timer := time.NewTimer(time.Millisecond * 500)

		for {
			select {
			case <-resetTimer:
				timer.Reset(time.Millisecond * 500)
			case <-timer.C:
				mu.Lock()
				if currentFile != nil {
					currentEncoder.Close()
					currentFile.Close()
					fileN++
					fmt.Println("End of audio stream")
					currentFile = nil
					currentEncoder = nil

					// doing transcoding
					go func(file string) {
						defer os.Remove(file)

						// Run the external application and capture its output
						output, err := runWhisper(file)
						if err != nil {
							fmt.Println("Error running whisper.exe:", err)
							return
						}

						// Extract the transcribed text from the output
						transcript, err := extractTranscript(output)
						if err != nil {
							fmt.Println("Error extracting transcript:", err)
							return
						}

						fmt.Println("Transcript: ", transcript)
						return
						if strings.HasPrefix(transcript, "(") || strings.HasPrefix(transcript, "[") {
							return
						}

						resp, err := openaiClient.CreateChatCompletion(
							context.Background(),
							openai.ChatCompletionRequest{
								Model:       openai.GPT3Dot5Turbo,
								Temperature: 0,
								Messages: []openai.ChatCompletionMessage{
									{
										Role:    openai.ChatMessageRoleSystem,
										Content: "Update LED status JSON according to user request and return it as response " + lastState + ", only respond with JSON. RGB values are 0 to 255. DO not use JSON comments.",
									},
									{
										Role:    openai.ChatMessageRoleUser,
										Content: "json: " + lastState + " request: " + transcript,
									},
								},
							},
						)

						//fmt.Println(resp)
						if err != nil {
							fmt.Printf("ChatCompletion error: %v\n", err)
							return
						}
						fmt.Println(resp.Choices[0].Message.Content)

						js, err := extractJSON2(resp.Choices[0].Message.Content)
						if err != nil {
							fmt.Printf("Can't find command: %v\n", err)
							return
						}

						// finding JSON
						lastState = js

						conn.WriteToUDP(
							[]byte(js),
							currentAddr,
						)

						//conn.WriteToUDP([]byte("["+
						//	"{'red':100,'green':200,'blue':100},"+
						//	"{'red':100,'green':100,'blue':100},"+
						//	"{'red':100,'green':0,'blue':100},"+
						//	"{'red':0,'green':200,'blue':200},"+
						//	"{'red':100,'green':0,'blue':200}"+
						//	"]"), currentAddr)

					}(fmt.Sprintf("stream-%d.wav", fileN-1))
				}
				mu.Unlock()
			}
		}
	}()

	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}

		//fmt.Printf("Received %d bytes from %s\n", n, addr)

		mu.Lock()
		currentAddr = addr
		if currentFile == nil {
			fileName := fmt.Sprintf("stream-%d.wav", fileN)
			//fmt.Println("New audio file", fileName)

			currentFile, err = os.Create(fileName)
			if err != nil {
				fmt.Println("Error creating file:", err)
				continue
			}

			currentEncoder = wav.NewEncoder(
				currentFile,
				16000, // sample rate
				16,    // bits per sample
				1,     // number of channels
				0x01,  // format
			)
		}

		intBuffer := make([]int, n/2)
		for i := 0; i < n; i += 2 {
			intBuffer[i/2] = int(buffer[i]) | int(buffer[i+1])<<8
		}

		err = currentEncoder.Write(&audio.IntBuffer{
			Format: &audio.Format{
				NumChannels: 1,
				SampleRate:  8000,
			},
			Data:           intBuffer,
			SourceBitDepth: 16,
		})
		if err != nil {
			fmt.Println("Error writing wav: ", err)
		}

		resetTimer <- struct{}{}
		mu.Unlock()
	}
}

func runWhisper(wavFile string) (string, error) {
	cmd := exec.Command("whisper/main.exe", wavFile)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	//fmt.Println((string)(outputBytes))

	return string(outputBytes), nil
}

func extractTranscript(output string) (string, error) {
	r := regexp.MustCompile(`\[\d\d:\d\d:\d\d\.\d\d\d --> \d\d:\d\d:\d\d\.\d\d\d\]\s+(.*)`)
	matches := r.FindStringSubmatch(output)
	if len(matches) < 2 {
		return "", fmt.Errorf("unable to find transcript in output")
	}
	return matches[1], nil
}
