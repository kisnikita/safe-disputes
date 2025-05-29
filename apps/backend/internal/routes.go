package internal

import (
	"github.com/kisnikita/safe-disputes/backend/internal/api"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
)

func (s Server) RegisterRoutes(repo *repository.Repository) {
	apiRouter := s.router.Group("/api/v1", api.Middleware())

	auth := apiRouter.Group("/auth")
	auth.POST("/telegram", api.TelegramAuth(repo, s.logger))

	users := apiRouter.Group("/users")
	users.GET("/me", api.GetMe(repo, s.logger))
	users.PATCH("", api.UpdateUser(repo, s.logger))
	users.GET("/top", api.GetTop(repo, s.logger))

	disputes := apiRouter.Group("/disputes")
	disputes.GET("", api.ListDisputes(repo, s.logger, s.msgService))
	disputes.POST("", api.CreateDispute(repo, s.logger, s.msgService))
	disputes.GET("/:id", api.GetDispute(repo, s.logger, s.msgService))
	disputes.GET("/:id/evidence", api.GetDisputeForEvidence(repo, s.logger, s.msgService))
	disputes.POST("/:id/accept", api.AcceptDispute(repo, s.logger, s.msgService))
	disputes.POST("/:id/reject", api.RejectDispute(repo, s.logger, s.msgService))
	disputes.POST("/:id/claim", api.ClaimDispute(repo, s.logger, s.msgService))
	disputes.POST("/:id/vote", api.VoteDispute(repo, s.logger, s.msgService))
	disputes.POST("/:id/evidence", api.EvidenceDispute(repo, s.logger, s.msgService))

	evidence := apiRouter.Group("/evidence")
	evidence.GET("", api.GetEvidencesByDispute(repo, s.logger, s.msgService))

	investigation := apiRouter.Group("/investigations")
	investigation.GET("", api.ListInvestigations(repo, s.logger, s.msgService))
	investigation.GET("/:id", api.GetInvestigation(repo, s.logger, s.msgService))
	investigation.POST("/:id/vote", api.VoteInvestigation(repo, s.logger, s.msgService))
}
