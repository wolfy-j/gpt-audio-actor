package main

import (
	"audioactor"
	"audioactor/handler"
	"flag"
	"github.com/sashabaranov/go-openai"
	"log"
	"net"
	"os"
	"time"
)

// os args
var (
	address      = flag.String("address", "0.0.0.0:5001", "address to listen on")
	whisper      = flag.String("whisper", "whisper/main", "whisper to use (default: whisper/main)")
	whisperModel = flag.String("model", "whisper/models/ggml-base.en.bin", "whisper model (default: whisper/models/ggml-base.en.bin)")
)

func main() {
	flag.Parse()

	udpAddress, err := net.ResolveUDPAddr("udp", *address)
	if err != nil {
		log.Fatalf("error parsing address: %v", err)
	}

	openaiClient := openai.NewClient(os.Getenv("OPENAI_KEY"))

	// handlers
	//chatHandler := handler.NewAIChatHandler(openaiClient)

	// 5 RGB led device
	jsonDevice, err := handler.NewDevice(
		openaiClient,
		"[{'red':0,'green':0,'blue':0},"+
			"{'red':0,'green':0,'blue':0},"+
			"{'red':0,'green':0,'blue':0},"+
			"{'red':0,'green':0,'blue':0},"+
			"{'red':0,'green':0,'blue':0}]",
		"You are tasked to update JSON array of RGB colors, each of them represent a LED on a remote device. You must return a valid JSON array of 5 RGB colors, omit any comments, or code samples, only respond with JSON.",
		"Update LED array {state} based on my request: {input}",
	)
	if err != nil {
		log.Fatalf("error parsing address: %v", err)
	}

	server := audioactor.NewServer(
		udpAddress,
		audioactor.NewCLITranscriber(*whisper, *whisperModel),
		time.Millisecond*500,
		//chatHandler.Handle,
		jsonDevice.Handle,
	)
	defer server.Close()

	log.Println("Listening on", udpAddress)

	err = server.Serve()
	if err != nil {
		log.Fatalf("error serving: %v", err)
	}
}
