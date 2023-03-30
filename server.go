package audioactor

import (
	"bytes"
	"log"
	"net"
	"sync"
	"time"
)

// Handler is called when a new transcription is available
type Handler func(conn *net.UDPConn, source *net.UDPAddr, transcription string)

type SpeechBuffer struct {
	source *net.UDPAddr
	data   bytes.Buffer
	upd    *time.Timer
}

func (s *SpeechBuffer) Write(p []byte, expand time.Duration) (n int, err error) {
	return s.data.Write(p)
}

// Server listens for UDP audio packets and transcribes them using external tool.
// Current implement relies on a simple external CLI command, however it is possible
// to gain much higher performance via streaming approach.
type Server struct {
	// address to listen on
	address *net.UDPAddr

	// speech delay
	delay time.Duration

	// active connection
	mu   sync.Mutex
	conn *net.UDPConn

	// transcriber is used to transcribe audio packets
	transcriber Transcriber

	// active audio buffers
	mub sync.Mutex
	buf map[string]*SpeechBuffer

	// handler is called when a new transcription is available
	handler Handler
}

// NewServer creates a new Server instance
func NewServer(address *net.UDPAddr, transcriber Transcriber, delay time.Duration, handler Handler) *Server {
	return &Server{
		address:     address,
		transcriber: transcriber,
		delay:       delay,
		buf:         make(map[string]*SpeechBuffer),
		handler:     handler,
	}
}

// Serve starts listening for UDP packets and transcribing them
func (s *Server) Serve() error {
	conn, err := net.ListenUDP("udp", s.address)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	buffer := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("Error reading from UDP:", err)
			continue
		}

		//log.Printf("Received %d bytes from %s\n", n, addr)
		s.handlePacket(buffer[:n], addr)
	}
}

func (s *Server) handlePacket(buffer []byte, addr *net.UDPAddr) {
	s.mub.Lock()
	defer s.mub.Unlock()

	if _, ok := s.buf[addr.String()]; !ok {
		s.buf[addr.String()] = &SpeechBuffer{
			source: addr,
			upd: time.AfterFunc(s.delay, func() {
				s.flushBuffer(addr)
			}),
		}
	}

	// expand the buffer
	s.buf[addr.String()].upd.Reset(s.delay)
	s.buf[addr.String()].Write(buffer, 1*time.Second)
}

func (s *Server) flushBuffer(addr *net.UDPAddr) {
	s.mub.Lock()
	defer s.mub.Unlock()

	if _, ok := s.buf[addr.String()]; !ok {
		return
	}

	result, err := s.transcriber.TranscribeBytes(s.buf[addr.String()].data)
	delete(s.buf, addr.String())

	if err != nil {
		log.Println(addr, ">", err)
		return
	}

	s.handler(s.conn, addr, result)
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return nil
	}

	return s.conn.Close()
}
