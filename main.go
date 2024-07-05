package main

import (
	"context"
	"fmt"
	"goredis/client"
	"log"
	"log/slog"
	"net"
	"time"
)

const defaultListenAddress = ":8080"

type Config struct {
	ListenAddress string
}

type Message struct {
	cmd  Command
	peer *Peer
}

type Server struct {
	Config
	peers     map[*Peer]bool
	ln        net.Listener
	addPeerCh chan *Peer
	quitCh    chan struct{}
	msgCh     chan Message

	kv *KV
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
		msgCh:     make(chan Message),
		kv:        NewKV(),
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
		case msg := <-s.msgCh:
			if err := s.handleMessage(msg); err != nil {
				slog.Error("raw message error", "err", err)
			}
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

func (s *Server) handleMessage(msg Message) error {
	switch v := msg.cmd.(type) {
	case GetCommand:
		// slog.Info("somebody wants to get a key from the hash table", "key", v.key)
		val, ok := s.kv.Get(v.key)

		if !ok {
			return fmt.Errorf("key not found")
		}
		_, err := msg.peer.Send(val)
		if err != nil {
			slog.Error("peer send error", "err", err)
			return err
		}

	case SetCommand:
		// slog.Info("somebody wants to set a key into the hash table", "key", v.key, "val", v.val)
		return s.kv.Set(v.key, v.val)
	}

	return nil
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

	go func() {
		log.Fatal(server.Start())
	}()

	time.Sleep(time.Second)

	client, err := client.New("localhost:8080")

	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		fmt.Println("SET =>", fmt.Sprintf("foo_%d", i), fmt.Sprintf("bar_%d", i))
		// SET COMMAND
		if err := client.Set(context.Background(), fmt.Sprintf("foo_%d", i), fmt.Sprintf("bar_%d", i)); err != nil {
			log.Fatal(err)
		}
		// GET COMMAND
		val, err := client.Get(context.Background(), fmt.Sprintf("foo_%d", i))

		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("GET => " + val)
	}

	fmt.Println(server.kv.data)
}
