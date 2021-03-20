package transport

import "github.com/vicxqh/srp/proto"

type Segment struct {
	Header  proto.Header
	Payload []byte
}

type Transport interface {
	Receive() (Segment, error)
	Send(Segment) error
}
