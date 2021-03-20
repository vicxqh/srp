package plain

import (
	"io"
	"net"

	"github.com/vicxqh/srp/log"
	"github.com/vicxqh/srp/proto"

	"github.com/vicxqh/srp/transport"
)

type Connection struct {
	conn net.Conn
}

func NewConnection(conn net.Conn) *Connection {
	return &Connection{conn}
}

func (c *Connection) Receive() (seg transport.Segment, err error) {
	seg.Header = make(proto.Header, proto.HeaderSize)
	n, err := io.ReadFull(c.conn, seg.Header)
	//n, err := c.conn.Read(seg.Header)
	if err != nil || n != proto.HeaderSize {
		log.Error("failed to read header, size %d, error %v", n, err)
		return
	}
	seg.Payload = make([]byte, seg.Header.PayloadLength())
	n, err = io.ReadFull(c.conn, seg.Payload)
	//n, err = c.conn.Read(seg.Payload)
	if err != nil || n != len(seg.Payload) {
		log.Error("failed to read payload, size %d (expected %d), error %v", n, len(seg.Payload), err)
		return
	}
	return
}

func (c *Connection) Send(segment transport.Segment) error {
	_, err := c.conn.Write(segment.Header)
	if err != nil {
		log.Error("failed to write segment header, %v", err)
		return err
	}
	_, err = c.conn.Write(segment.Payload)
	if err != nil {
		log.Error("failed to write segment payload, %v", err)
		return err
	}
	return nil
}
