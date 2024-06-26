package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
)

const defaultListenAddress = ":8080"

type Config struct {
	ListenAddress string
}

type Server struct {
	Config
	peers     map[*Peer]bool
	ln        net.Listener
	addPeerCh chan *Peer
	quitCh    chan struct{}
	msgCh     chan []byte
}

func NewServer(cfg Config) *Server {
	if len(cfg.ListenAddress) == 0 {
		cfg.ListenAddress = defaultListenAddress
	}

	return &Server{
		Config:    cfg,
		peers:     make(map[*Peer]bool),
		addPeerCh: make(chan *Peer),
		quitCh:    make(chan struct{}),
		msgCh:     make(chan []byte),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddress)

	if err != nil {
		return err
	}

	s.ln = ln

	go s.loop()

	slog.Info("server running", "listenAddrs", s.ListenAddress)

	return s.acceptLoop()
}

func (s *Server) loop() {
	for {
		select {
		case rawMsg := <-s.msgCh:
			fmt.Println(rawMsg)
		case <-s.quitCh:
			return
		case peer := <-s.addPeerCh:
			s.peers[peer] = true
		}
	}
}

func (s *Server) acceptLoop() error {
	for {
		conn, err := s.ln.Accept()

		if err != nil {
			slog.Error("accept error", "err", err)
			continue
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	peer := NewPeer(conn, s.msgCh)
	s.addPeerCh <- peer

	slog.Info("new peer connected", "remoteAddr", conn.RemoteAddr())

	if err := peer.readLoop(); err != nil {
		slog.Error("peer read error", "err", err, "remoteAddr", conn.RemoteAddr())
	}
}

func main() {
	server := NewServer(Config{})
	log.Fatal(server.Start())
}
