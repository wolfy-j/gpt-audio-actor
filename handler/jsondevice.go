package handler

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"github.com/sashabaranov/go-openai"
	"log"
	"net"
	"strings"
)

// JSONDevice maintains it's state via internal JSON and stream updates back to the original requester.
type JSONDevice struct {
	client *openai.Client

	// current state
	state string

	// Prompts for the OpenAI API.
	system string
	user   string
}

// NewDevice creates a new JSONDevice instance.
func NewDevice(client *openai.Client, state, system, user string) (*JSONDevice, error) {
	// check if user prompt contains {state} and {input} placeholder
	if !strings.Contains(user, "{state}") {
		return nil, fmt.Errorf("user prompt must contain {state} placeholder")
	}
	if !strings.Contains(user, "{input}") {
		return nil, fmt.Errorf("user prompt must contain {input} placeholder")
	}

	return &JSONDevice{
		client: client,
		state:  state,
		system: system,
		user:   user,
	}, nil
}

func (d *JSONDevice) Handle(conn *net.UDPConn, source *net.UDPAddr, transcription string) {
	log.Println(source, color.GreenString(transcription))

	request := strings.ReplaceAll(d.user, "{state}", d.state)
	request = strings.ReplaceAll(request, "{input}", transcription)

	// create completion request
	resp, err := d.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       openai.GPT3Dot5Turbo,
			Temperature: 0,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: d.system,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: request,
				},
			},
		})

	if err != nil {
		log.Println("ai >", color.RedString(err.Error()))
		return
	}

	// extract JSON
	json, err := findJSON(resp.Choices[0].Message.Content)
	if err != nil {
		log.Println("ai >", color.RedString(err.Error()))
		return
	}

	// update state
	d.state = json

	log.Println("ai >", color.CyanString(json))

	// push state to requester
	conn.WriteToUDP([]byte(json), source)
}

func findJSON(s string) (string, error) {
	start := strings.Index(s, "[{")
	if start == -1 {
		// The starting substring wasn't found.
		return "", fmt.Errorf("no JSON found: %s", s)
	}

	end := strings.Index(s[start+2:], "}]")
	if end == -1 {
		// The ending substring wasn't found.
		return "", fmt.Errorf("no JSON found: %s", s)
	}

	// Return the extracted substring, including the start and end substrings.
	return s[start : start+end+4], nil
}
