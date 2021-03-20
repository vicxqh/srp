package internal

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/vicxqh/srp/transport"

	"github.com/vicxqh/srp/log"
	"github.com/vicxqh/srp/proto"
)

var exposures sync.Map

type Exposure struct {
	ServiceId string
	AgentId   string
	Port      string
	lis       net.Listener
	ctx       context.Context
	cancel    context.CancelFunc
}

func (exp *Exposure) ServeUsers() {
	for {
		conn, err := exp.lis.Accept()
		if err != nil {
			log.Error("exposure %s failed to accept, %v", exp.ServiceId, err)
			continue
		}
		go exp.handleUserConnection(conn)
	}
}

var users sync.Map

func ForwardToUser(segment transport.Segment) error {
	header := segment.Header
	log.Debug("service(%s) -> user(%s) : %d bytes", header.Service(), header.User(), header.PayloadLength())

	cv, ok := users.Load(header.User())
	if !ok {
		err := fmt.Errorf("no connection for user %s", header.User())
		log.Error("%v", err)
		return err
	}
	uc := cv.(*userConnection)
	return uc.sendToUser(segment.Payload)
}

type userConnection struct {
	ctx      context.Context
	cancel   context.CancelFunc
	exposure *Exposure
	user     string
	sendChan chan []byte
	conn     net.Conn
}

func (uc *userConnection) sendToUser(data []byte) error {
	uc.sendChan <- data
	return nil
}

func (uc *userConnection) Stop() {
	uc.cancel()
	uc.conn.Close()
}

func (uc *userConnection) SendLoop() {
	for {
		select {
		case <-uc.ctx.Done():
			return
		case data := <-uc.sendChan:
			_, err := uc.conn.Write(data)
			if err != nil {
				log.Error("failed to write to user %s, %v", uc.user, err)
				uc.cancel()
				return
			}
		}
	}
}

func (uc *userConnection) RecvLoop() {
	defer uc.cancel()
	for {
		select {
		case <-uc.ctx.Done():
			return
		default:
			buffer := make([]byte, 1024)
			n, err := uc.conn.Read(buffer)
			if err != nil {
				log.Error("failed to read from user %s, %v", uc.user, err)
				return
			}
			data := buffer[:n]
			log.Debug("received %d bytes from users %s", len(data), uc.user)
			svc, err := getService(uc.ctx, uc.exposure.ServiceId)
			if err != nil {
				log.Error("failed to get service, %v. service might get updated", err)
				return
			}
			header, _ := proto.NewHeader(uc.user, svc.Addr)
			header.SetPayloadLength(uint32(len(data)))
			SendToAgent(uc.exposure.AgentId, transport.Segment{header, data})
		}
	}
}

func (exp *Exposure) handleUserConnection(conn net.Conn) {
	user := conn.RemoteAddr().String()
	log.Info("new user connection from %s", user)

	ctx, cancel := context.WithCancel(context.Background())
	uc := &userConnection{
		ctx:      ctx,
		cancel:   cancel,
		exposure: exp,
		user:     conn.RemoteAddr().String(),
		sendChan: make(chan []byte, 1),
		conn:     conn,
	}
	users.Store(user, uc)

	go uc.SendLoop()
	go uc.RecvLoop()

	<-uc.ctx.Done()
	users.Delete(user)
	log.Info("removed user connection %s", user)
	uc.Stop()
}

func (exp *Exposure) Stop() {
	exp.cancel()
	exp.lis.Close()
}

func NewExposure(serviceId, agentId, port string) error {
	if old, ok := exposures.Load(serviceId); ok {
		oe := old.(*Exposure)
		oe.Stop()
	}

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	_, err = getService(ctx, serviceId)
	if err != nil {
		return err
	}
	e := &Exposure{
		ServiceId: serviceId,
		AgentId:   agentId,
		Port:      port,
		ctx:       ctx,
		cancel:    cancel,
	}

	if e.lis, err = net.Listen("tcp4", ":"+port); err != nil {
		log.Error("failed to listen on port %s, %v", port, err)
		return err
	}
	log.Info("exposed. %+v", e)
	exposures.Store(serviceId, e)
	go e.ServeUsers()
	return nil
}

func DeleteExposure(serviceId string) error {
	if old, ok := exposures.Load(serviceId); ok {
		oe := old.(*Exposure)
		oe.Stop()
	}
	exposures.Delete(serviceId)
	return nil
}

func GetExposure(serviceId string) *Exposure {
	v, ok := exposures.Load(serviceId)
	if !ok {
		return nil
	}
	return v.(*Exposure)
}
