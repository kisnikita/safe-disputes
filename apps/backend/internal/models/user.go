package models

import (
	"github.com/google/uuid"
)

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
