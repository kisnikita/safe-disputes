package internal

import (
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/internal/transport/api"
)

func (s Server) RegisterRoutes(repo *repository.Repository) {
	apiRouter := s.router.Group("/apiRouter/v1")
	auth := apiRouter.Group("/auth")
	auth.POST("/telegram", api.TelegramAuth(repo, s.logger))
}
