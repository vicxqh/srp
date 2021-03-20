package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/vicxqh/srp/transport"
	"github.com/vicxqh/srp/transport/plain"

	"github.com/vicxqh/srp/log"
	"github.com/vicxqh/srp/types"
)

func ConnectToServer(server, myName, description string) {
	req := types.AgentRegistrationRequest{
		ID:          myName,
		Description: description,
	}
	retrying := false
	for {
		if retrying {
			time.Sleep(time.Second)
		} else {
			retrying = true
		}
		rsp, err := http.Get(fmt.Sprintf("http://%s/api/v1/dataport", server))
		if err != nil {
			log.Error("failed to get data port, %v", err)
			continue
		}
		body, _ := ioutil.ReadAll(rsp.Body)
		rsp.Body.Close()
		if rsp.StatusCode != http.StatusOK {
			log.Error("failed to get data port, http status %d, body: %s", rsp.StatusCode, string(body))
			continue
		}
		ss := strings.Split(server, ":")
		dataServer := ss[0] + ":" + string(body)
		log.Info("connecting to data server %s ...", dataServer)
		conn, err := net.Dial("tcp", dataServer)
		if err != nil {
			log.Error("failed to connect to data server %s, %v", dataServer, err)
			continue
		}

		ctx, cancel := context.WithCancel(context.Background())
		sc = &serverConnection{
			rawConn:  conn,
			req:      req,
			ctx:      ctx,
			cancel:   cancel,
			sendChan: make(chan transport.Segment, 1),
		}
		sc.Serve()
	}
}

type serverConnection struct {
	rawConn  net.Conn
	conn     transport.Transport
	req      types.AgentRegistrationRequest
	ctx      context.Context
	cancel   context.CancelFunc
	sendChan chan transport.Segment
}

var sc *serverConnection

func (sc *serverConnection) handshake() error {
	log.Info("handshaking ...")
	data, _ := json.Marshal(sc.req)
	if _, err := sc.rawConn.Write(data); err != nil {
		log.Error("failed to write to server, %v", err)
		return err
	}
	buffer := make([]byte, 1024)
	n, err := sc.rawConn.Read(buffer)
	if err != nil {
		log.Error("failed to read registration response, %v", err)
		return err
	}
	buffer = buffer[:n]
	log.Info("server response: %s", string(buffer))

	var regRsp types.AgentRegistrationResponse
	if err := json.Unmarshal(buffer, &regRsp); err != nil {
		log.Error("failed to unmarshal handshake data %s, %v", string(buffer), err)
		return err
	}
	if !regRsp.Succeeded {
		log.Error("server returned failure, %s", regRsp.Message)
		return errors.New(regRsp.Message)
	}

	sc.conn = plain.NewConnection(sc.rawConn)
	return nil
}

func (sc *serverConnection) Stop() {
	sc.cancel()
	sc.rawConn.Close()
}

func (sc *serverConnection) Serve() error {
	defer sc.Stop()
	if err := sc.handshake(); err != nil {
		log.Error("failed to do handshake, %v", err)
		return err
	}
	go sc.SendLoop()
	go sc.RecvLoop()
	<-sc.ctx.Done()
	return sc.ctx.Err()
}

func (sc *serverConnection) SendLoop() {
	for {
		select {
		case <-sc.ctx.Done():
			return
		case data := <-sc.sendChan:
			log.Debug("service(%s) -> user(%s) : %d bytes", data.Header.Service(), data.Header.User(),
				data.Header.PayloadLength())
			if err := sc.conn.Send(data); err != nil {
				log.Error("failed to send to server, %v", err)
				sc.cancel()
				return
			}
		}
	}
}

func SendToServer(segment transport.Segment) error {
	sc.sendChan <- segment
	return nil
}

func (sc *serverConnection) RecvLoop() {
	for {
		select {
		case <-sc.ctx.Done():
			return
		default:
			data, err := sc.conn.Receive()
			if err != nil {
				log.Error("failed to receive from server, %v", err)
				sc.cancel()
				return
			}
			ForwardToService(data)
		}
	}
}
