package internal

import (
	"io/ioutil"
	"net/http"
	"net/http/httputil"

	"github.com/vicxqh/srp/types"

	"github.com/vicxqh/srp/log"

	"github.com/gin-gonic/gin"
)

func (s *Server) GetDataPort(c *gin.Context) {
	c.String(http.StatusOK, "%d", s.DataPort())
}

func (s *Server) ListServices(c *gin.Context) {
	services, err := listServices(c)
	if err != nil {
		log.Error("failed to list services, %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, services)
}

func (s *Server) GetService(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		log.Error("empty id")
		c.Status(http.StatusBadRequest)
		return
	}
	svc, err := getService(c, id)
	if err != nil {
		log.Error("failed to get service %s, %v", id, err)
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, svc)
}

func (s *Server) UpdateService(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		log.Error("empty id")
		c.Status(http.StatusBadRequest)
		return
	}
	var svc types.Service
	err := c.BindJSON(&svc)
	if err != nil {
		c.Status(http.StatusBadRequest)
		log.Error("failed to bind http body as an instance of Service, %v", err)
		b, err := c.Request.GetBody()
		if err != nil {
			log.Error("failed to get another copy of body, %v", err)
			return
		}
		defer b.Close()
		data, _ := ioutil.ReadAll(b)
		log.Info("received http body: %s", string(data))
		return
	}
	err = updateService(c, id, svc)
	if err != nil {
		log.Error("failed to update service, %v", err)
		if err == ErrNotFound {
			c.String(http.StatusBadRequest, "not found")
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}
}

func (s *Server) CreateService(c *gin.Context) {
	var svc types.Service
	err := c.BindJSON(&svc)
	if err != nil {
		c.JSON(http.StatusBadRequest, "body should be a service")
		log.Error("failed to bind http body as an instance of Service, %v", err)
		b, err := c.Request.GetBody()
		if err != nil {
			log.Error("failed to get another copy of body, %v", err)
			return
		}
		defer b.Close()
		data, _ := ioutil.ReadAll(b)
		log.Info("received http body: %s", string(data))
		return
	}
	err = createService(c, svc)
	if err != nil {
		log.Error("failed to create service, %v", err)
		if err == ErrAlreadyExist {
			c.String(http.StatusBadRequest, "already existed")
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}
}

func (s *Server) DeleteService(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		log.Error("empty id")
		c.Status(http.StatusBadRequest)
		return
	}
	err := deleteService(c, id)
	if err != nil {
		log.Error("failed to delete service, %v", id, err)
		if err == ErrNotFound {
			c.String(http.StatusBadRequest, "not found")
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}
	DeleteExposure(id)
}

func (s *Server) ListAgents(c *gin.Context) {
	c.JSON(http.StatusOK, listAgents())
}

func (s *Server) ExposeService(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		log.Error("empty service id")
		c.Status(http.StatusBadRequest)
		return
	}
	agentId := c.Query("agent")
	port := c.Query("port")
	if agentId == "" || port == "" {
		c.String(http.StatusBadRequest, "required parameters in query: agent, port")
		return
	}
	err := NewExposure(id, agentId, port)
	if err != nil {
		log.Error("failed to create new exposure, %v", err)
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Status(http.StatusOK)
}

func (s *Server) StopExposingService(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		log.Error("empty service id")
		c.Status(http.StatusBadRequest)
		return
	}
	DeleteExposure(id)
	c.Status(http.StatusOK)
}

func (s *Server) httpHandler() http.Handler {
	router := gin.Default()
	g := router.Group("/api/v1")

	// log every request and response
	g.Use(func(c *gin.Context) {
		dump, _ := httputil.DumpRequest(c.Request, true)
		log.Info("%s", string(dump))
		c.Next()
		log.Info("response: %d", c.Writer.Status())
	})

	gin.Logger()

	g.GET("dataport", s.GetDataPort)

	g.GET("services", s.ListServices)
	g.POST("services", s.CreateService)
	g.GET("services/:id", s.GetService)
	g.PUT("services/:id", s.UpdateService)
	g.DELETE("services/:id", s.DeleteService)
	g.GET("agents", s.ListAgents)
	g.PUT("services/:id/exposure", s.ExposeService)
	g.DELETE("services/:id/exposure", s.StopExposingService)

	return router
}
