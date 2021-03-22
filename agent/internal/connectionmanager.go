package internal

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/vicxqh/srp/log"
	"github.com/vicxqh/srp/proto"
	"github.com/vicxqh/srp/transport"
)

func ForwardToService(segment transport.Segment) error {
	log.Debug("user(%s) -> service(%s) : %d bytes", segment.Header.User(), segment.Header.Service(),
		segment.Header.PayloadLength())
	sc := GetConnection(segment.Header)
	if sc == nil {
		return fmt.Errorf("no conneciton for user(%s):service(%s) was available",
			segment.Header.User(), segment.Header.Service())
	}
	return sc.send(segment.Payload)
}

var connections sync.Map

type serviceConnection struct {
	ctx      context.Context
	cancel   context.CancelFunc
	user     string
	service  string
	conn     net.Conn
	sendChan chan []byte
}

func (sc *serviceConnection) send(data []byte) error {
	sc.sendChan <- data
	return nil
}

func (sc *serviceConnection) Serve() {
	go sc.SendLoop()
	go sc.RecvLoop()

	<-sc.ctx.Done()
	key := sc.user + "->" + sc.service
	connections.Delete(key)
	log.Info("removed service connection %s", key)
	sc.conn.Close()
}

func (sc *serviceConnection) SendLoop() {
	for {
		select {
		case <-sc.ctx.Done():
			return
		case data := <-sc.sendChan:
			_, err := sc.conn.Write(data)
			if err != nil {
				log.Error("failed to write to service %s, %v", sc.service, err)
				sc.cancel()
				return
			}
		}
	}
}

func (sc *serviceConnection) RecvLoop() {
	for {
		select {
		case <-sc.ctx.Done():
			return
		default:
			buffer := make([]byte, 1024)
			n, err := sc.conn.Read(buffer)
			if err != nil {
				log.Error("failed to read from service %s, %v", sc.service, err)
				sc.cancel()
				return
			}
			data := buffer[:n]
			log.Debug("received %d bytes from service %s", len(data), sc.service)
			header, _ := proto.NewHeader(sc.user, sc.service)
			header.SetPayloadLength(uint32(len(data)))
			SendToServer(transport.Segment{header, data})
		}
	}
}

func GetConnection(header proto.Header) *serviceConnection {
	key := header.User() + "->" + header.Service()
	var sc *serviceConnection
	scv, ok := connections.Load(key)
	if ok {
		sc = scv.(*serviceConnection)
	} else {
		log.Info("creating new connection for %s", key)
		conn, err := net.Dial("tcp", header.Service())
		if err != nil {
			log.Error("failed to dial to service %s, %v", header.Service(), err)
			return nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		sc = &serviceConnection{
			user:     header.User(),
			service:  header.Service(),
			conn:     conn,
			ctx:      ctx,
			cancel:   cancel,
			sendChan: make(chan []byte, 1),
		}

		go sc.Serve()
		connections.Store(key, sc)
	}
	return sc
}
