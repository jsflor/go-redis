package client

import (
	"bytes"
	"context"
	"log"
	"net"

	"github.com/tidwall/resp"
)

type Client struct {
	addr string
}

func New(addr string) *Client {
	return &Client{
		addr: addr,
	}
}

func (c *Client) Set(ctx context.Context, key string, val string) error {
	conn, err := net.Dial("tcp", c.addr)

	if err != nil {
		log.Fatal(err)
	}

	buf := &bytes.Buffer{}

	wr := resp.NewWriter(buf)
	wr.WriteArray([]resp.Value{resp.StringValue("SET"), resp.StringValue(key), resp.StringValue(val)})

	_, err = conn.Write(buf.Bytes())

	return err
}
