package handler

import (
	"context"
	"github.com/fatih/color"
	"github.com/sashabaranov/go-openai"
	"log"
	"net"
	"sync"
	"time"
)

// AIChatHandler is a simple demo of ChatGPT session handler. Session is maintained in memory
// for 5 minutes after last message.
type AIChatHandler struct {
	// ai client
	client *openai.Client

	// chat session
	mu      sync.Mutex
	chat    *openai.ChatCompletionRequest
	cleaner *time.Timer
}

// NewAIChatHandler creates new AIChatHandler instance.
func NewAIChatHandler(client *openai.Client) *AIChatHandler {
	handler := &AIChatHandler{
		client: client,
		chat: &openai.ChatCompletionRequest{
			Model:       openai.GPT3Dot5Turbo,
			Temperature: 0.1,
			Messages:    []openai.ChatCompletionMessage{},
		},
	}

	handler.cleaner = time.AfterFunc(time.Minute*5, func() {
		handler.mu.Lock()
		defer handler.mu.Unlock()
		handler.chat.Messages = []openai.ChatCompletionMessage{}
	})

	return handler
}

// Handle incoming message
func (h *AIChatHandler) Handle(conn *net.UDPConn, source *net.UDPAddr, transcription string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	log.Println(source, color.GreenString(transcription))

	h.chat.Messages = append(h.chat.Messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: transcription,
	})

	// getting response
	resp, err := h.client.CreateChatCompletion(context.Background(), *h.chat)

	if err != nil {
		log.Println("ai >", color.RedString(err.Error()))
		return
	}

	// display response
	log.Println("ai >", color.YellowString(resp.Choices[0].Message.Content))

	// add response to chat
	h.chat.Messages = append(h.chat.Messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: resp.Choices[0].Message.Content,
	})
}
