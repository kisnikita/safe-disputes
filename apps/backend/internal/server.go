package internal

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kisnikita/safe-disputes/backend/pkg/log"
)

type Server struct {
	logger log.Logger
	router *gin.Engine
	srv    *http.Server
}

func NewServer(logger log.Logger) *Server {
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"POST", "GET", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	return &Server{
		logger: logger,
		router: r,
		srv: &http.Server{
			Addr:    os.Getenv("PORT"),
			Handler: r,
		},
	}
}

func (s Server) StartServer() {
	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Fatal("listen error", zap.Error(err))
	}
}
