package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/vicxqh/srp/transport/plain"

	"github.com/vicxqh/srp/transport"

	"github.com/vicxqh/srp/log"
	"github.com/vicxqh/srp/types"
)

var agents sync.Map

func addAgent(agent *agent) error {
	if agent.ID == "" {
		return errors.New("agent id is required")
	}
	if _, ok := agents.Load(agent.ID); ok {
		return ErrAlreadyExist
	}
	agents.Store(agent.ID, agent)
	return nil
}

func removeAgent(agent *agent) {
	if agent == nil {
		return
	}
	log.Info("removing agent %s", agent.ID)
	agents.Delete(agent.ID)
}

func listAgents() []types.Agent {
	var tagents []types.Agent
	agents.Range(func(key, value interface{}) bool {
		agent := value.(*agent)
		tagents = append(tagents, agent.Agent)
		return true
	})
	return tagents
}

func SendToAgent(agentId string, segment transport.Segment) error {
	agentV, ok := agents.Load(agentId)
	if !ok {
		return fmt.Errorf("agent %s not found", agentId)
	}
	agent := agentV.(*agent)
	return agent.Send(segment)
}

type agent struct {
	types.Agent
	conn     transport.Transport
	ctx      context.Context
	cancel   context.CancelFunc
	sendChan chan transport.Segment
}

func (a *agent) Send(segment transport.Segment) error {
	a.sendChan <- segment
	return nil
}

func (a *agent) sendLoop() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case data := <-a.sendChan:
			log.Debug("user(%s) -> service(%s) : %d bytes", data.Header.User(), data.Header.Service(),
				data.Header.PayloadLength())
			a.conn.Send(data)
		}
	}
}

func (a *agent) recvLoop() {
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			data, err := a.conn.Receive()
			if err != nil {
				log.Error("failed to receive from agent %s, %v", a.ID, err)
				a.cancel()
				return
			}
			ForwardToUser(data)
		}
	}
}

func (s *Server) AcceptAgents() {
	l, err := net.Listen("tcp4", fmt.Sprintf(":%d", s.DataPort()))
	if err != nil {
		log.Fatal("failed to listen on data port, %v", err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Error("unexpected error, %v", err)
			continue
		}
		go s.handleAgentConnection(conn)
	}
}

func (s *Server) handleAgentConnection(conn net.Conn) {
	defer conn.Close()
	log.Info("new agent connection from %s", conn.RemoteAddr().String())

	// registration handshake
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Error("failed to read registration, %v", err)
		return
	}
	buffer = buffer[:n]
	log.Info("agent meta: %s", string(buffer))

	var agentMeta types.Agent
	if err := json.Unmarshal(buffer, &agentMeta); err != nil {
		log.Error("illegal agent format, %s, %v", string(buffer), err)
		return
	}

	// register agent
	ctx, cancel := context.WithCancel(context.Background())
	agent := &agent{
		Agent:    agentMeta,
		conn:     plain.NewConnection(conn),
		ctx:      ctx,
		cancel:   cancel,
		sendChan: make(chan transport.Segment, 1),
	}
	if err := addAgent(agent); err != nil {
		log.Error("failed to add agent", err)
		rsp := types.AgentRegistrationResponse{
			Succeeded: false,
			Message:   err.Error(),
		}
		rspData, _ := json.Marshal(rsp)
		conn.Write(rspData)
		return
	}

	defer removeAgent(agent)

	rsp := types.AgentRegistrationResponse{
		Succeeded: true,
		Message:   "OK",
	}
	rspData, _ := json.Marshal(rsp)
	if _, err := conn.Write(rspData); err != nil {
		log.Error("failed to write registration response to agent, %v", err)
		return
	}

	go agent.sendLoop()
	go agent.recvLoop()

	<-agent.ctx.Done()
}
