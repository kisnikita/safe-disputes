package models

import "time"

type User struct {
	ID                  string    `db:"id" json:"id"`
	Username            string    `db:"username" json:"username"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	RefreshToken        string    `db:"refresh_token" json:"refresh_token"`
	NotificationEnabled bool      `db:"notification_enabled" json:"notification_enabled"`
	DisputeReadiness    bool      `db:"dispute_readiness" json:"dispute_readiness"`
}

func NewUser(username string) *User {
	return &User{
		ID:                  uuid(),
		Username:            username,
		CreatedAt:           time.Now(),
		RefreshToken:        "",
		NotificationEnabled: false,
		DisputeReadiness:    false,
	}
}
