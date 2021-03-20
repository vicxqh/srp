package proto

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Header
//
// 0                                 31
// +---------------------------------+  ---+
// |             user ip             |     |
// +---------------------------------+     |
// |            service ip           |     |
// +----------------+----------------+     |-- 16 bytes
// |    user port   |  service port  |     |
// +---------------------------------+     |
// |          payload length         |     |
// +----------------+----------------+  ---+
// |                                 |
// |            payload              |
// |                                 |
// +---------------------------------+
//
type Header []byte

const HeaderSize = 16 //16bytes, since we only support ipV4

func (h Header) User() string {
	ip := net.IPv4(h[0], h[1], h[2], h[3]).String()
	port := fmt.Sprintf("%d", int(h[8])<<8|int(h[9]))
	return ip + ":" + port
}

func (h Header) Service() string {
	ip := net.IPv4(h[4], h[5], h[6], h[7]).String()
	port := fmt.Sprintf("%d", int(h[10])<<8|int(h[11]))
	return ip + ":" + port
}

// SetPayloadLength set length of data payload. Length can NOT exceed uint32
func (h Header) SetPayloadLength(l uint32) {
	h[12] = byte(l >> 24)
	h[13] = byte(l >> 16)
	h[14] = byte(l >> 8)
	h[15] = byte(l)
}

func (h Header) PayloadLength() uint32 {
	return uint32(h[12])<<24 | uint32(h[13])<<16 | uint32(h[14])<<8 | uint32(h[15])
}

func NewHeader(user, service string) (Header, error) {
	uip, uport, err := parseAddr(user)
	if err != nil {
		return nil, err
	}
	sip, sport, err := parseAddr(service)
	if err != nil {
		return nil, err
	}
	h := make([]byte, HeaderSize)
	copy(h, []byte{
		uip[0], uip[1], uip[2], uip[3],
		sip[0], sip[1], sip[2], sip[3],
		byte(uport >> 8), byte(uport), byte(sport >> 8), byte(sport),
	})
	return h, nil
}

func parseAddr(addr string) (ip net.IP, port int, err error) {
	ss := strings.Split(addr, ":")
	if len(ss) != 2 {
		err = fmt.Errorf("%s is not a valid address, expected ip:port", addr)
		return
	}
	ip = net.ParseIP(ss[0])
	ip = ip.To4()
	if ip == nil {
		err = fmt.Errorf("%s is not a valid ip", ss[0])
		return
	}
	port, err = strconv.Atoi(ss[1])
	if err != nil {
		err = fmt.Errorf("%s is not an int, %v", ss[1], err)
		return
	}
	if port <= 0 || port > 65535 {
		err = fmt.Errorf("port %d is not valid", port)
		return
	}
	return
}
