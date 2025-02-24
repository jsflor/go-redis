package main

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/tidwall/resp"
)

type Peer struct {
	conn  net.Conn
	msgCh chan Message
}

func NewPeer(conn net.Conn, msgCh chan Message) *Peer {
	return &Peer{
		conn:  conn,
		msgCh: msgCh,
	}
}

func (p *Peer) Send(msg []byte) (int, error) {
	return p.conn.Write(msg)
}

func (p *Peer) readLoop() error {
	rd := resp.NewReader(p.conn)

	for {
		v, _, err := rd.ReadValue()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal(err)
		}

		if v.Type() == resp.Array {
			for _, value := range v.Array() {

				switch value.String() {
				case CommandGET:
					if len(v.Array()) != 2 {
						return fmt.Errorf("invalid number or variables for GET command")
					}
					cmd := GetCommand{
						key: v.Array()[1].Bytes(),
						// val: v.Array()[2].Bytes(),
					}

					p.msgCh <- Message{
						cmd:  cmd,
						peer: p,
					}
				case CommandSET:
					if len(v.Array()) != 3 {
						return fmt.Errorf("invalid number or variables for SET command")
					}
					cmd := SetCommand{
						key: v.Array()[1].Bytes(),
						val: v.Array()[2].Bytes(),
					}
					p.msgCh <- Message{
						cmd:  cmd,
						peer: p,
					}
				}
			}
		}
	}

	return nil
	// buf := make([]byte, 1024)
	// for {
	// 	n, err := p.conn.Read(buf)

	// 	if err != nil {
	// 		slog.Error("peer read error", "err", err)
	// 		return err
	// 	}

	// 	msfBuf := make([]byte, n)
	// 	copy(msfBuf, buf[:n])

	// 	p.msgCh <- Message{
	// 		data: msfBuf,
	// 		peer: p,
	// 	}

	// }
}
