package http

import (
	"auth-system/internal/config"

	"github.com/gin-gonic/gin"
)

type Server struct {
	engine *gin.Engine
	config *config.Config
}

func NewServer(cfg *config.Config) *Server {
	engine := gin.Default()
	return &Server{
		engine: engine,
		config: cfg,
	}
}

func (s *Server) GetEngine() *gin.Engine {
	return s.engine
}
