module AudioStreamer

go 1.18

require (
	github.com/ggerganov/whisper.cpp/bindings/go v0.0.0-20230307193630-09e90680072d
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.1.0
	github.com/sashabaranov/go-openai v1.5.3
)

require github.com/go-audio/riff v1.0.0 // indirect
replace github.com/ggerganov/whisper.cpp/bindings/go => ../whisper.cpp/bindings/go
