package internal

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vicxqh/srp/log"
)

type Server struct {
	httpPort int
	dataPort int
}

func NewServer(http, data int) *Server {
	gin.SetMode(gin.ReleaseMode)
	return &Server{
		httpPort: http,
		dataPort: data,
	}
}

func (s *Server) DataPort() int {
	return s.dataPort
}

func (s *Server) Run() error {
	InitDB()
	defer CloseDB()

	go s.AcceptAgents()

	return s.serveHttp()
}

func (s *Server) serveHttp() error {
	router := s.httpHandler()
	httpAddr := fmt.Sprintf(":%d", s.httpPort)
	log.Info("starting http service on %s", httpAddr)
	return http.ListenAndServe(httpAddr, router)
}
