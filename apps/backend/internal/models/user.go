package models

import (
	"time"

	"github.com/google/uuid"
)

type User_old struct {
	ID                   uuid.UUID `db:"id" json:"id"`
	Username             string    `db:"username" json:"username"`
	PhotoUrl             *string   `db:"photo_url" json:"photoUrl"`
	ChatID               int64     `db:"chat_id" json:"chatID"`
	CreatedAt            time.Time `db:"created_at" json:"createdAt"`
	NotificationEnabled  bool      `db:"notification_enabled" json:"notificationEnabled"`
	DisputeReadiness     bool      `db:"dispute_readiness" json:"disputeReadiness"`
	MinimumDisputeAmount int       `db:"minimum_dispute_amount" json:"minimumDisputeAmount"`
	Rating               int       `db:"rating" json:"rating"`
}

type UserUpdateOpts struct {
	Username             string `json:"username"`
	NotificationEnabled  *bool  `json:"notificationEnabled"`
	DisputeReadiness     *bool  `json:"disputeReadiness"`
	MinimumDisputeAmount *int   `json:"minimumDisputeAmount"`
	Rating               *int   `json:"rating"`
}

func NewUser(username string, photoUrl *string) User {
	return User{
		ID:       uuid.New(),
		Username: username,
		PhotoUrl: photoUrl,
	}
}
