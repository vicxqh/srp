package proto

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAddr(t *testing.T) {
	require := require.New(t)
	var ip net.IP
	var port int
	var err error

	ip, port, err = parseAddr("192.168.2.1")
	require.NotNil(err)

	ip, port, err = parseAddr("192.168.2.1:11:11")
	require.NotNil(err)

	ip, port, err = parseAddr("192.168.2.256:11")
	require.NotNil(err)

	ip, port, err = parseAddr("192.168.2.255:0")
	require.NotNil(err)

	ip, port, err = parseAddr("192.168.2.255:65536")
	require.NotNil(err)

	ip, port, err = parseAddr("192.168.2.255:1")
	require.Nil(err)
	require.Equal(1, port)
	require.True(ip.Equal(net.IPv4(192, 168, 2, 255)))

	ip, port, err = parseAddr("192.168.2.255:65535")
	require.Nil(err)
	require.Equal(65535, port)
}

func TestHeader(t *testing.T) {
	require := require.New(t)

	h, err := NewHeader("1.2.3.4:5", "192.168.1.255:8080")
	require.Nil(err)
	require.Equal("1.2.3.4:5", h.User())
	require.Equal("192.168.1.255:8080", h.Service())

	h.SetPayloadLength(4294967295)
	require.Equal(uint32(4294967295), h.PayloadLength())
	h.SetPayloadLength(0)
	require.Equal(uint32(0), h.PayloadLength())
	h.SetPayloadLength(123456789)
	require.Equal(uint32(123456789), h.PayloadLength())
}
