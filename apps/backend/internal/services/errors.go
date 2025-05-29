package services

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrUserNotFound  = fmt.Errorf("user %w", ErrNotFound)
	ErrMinimalAmount = errors.New("amount is less than opponent's minimum disputes amount")
	ErrUnready       = errors.New("not ready for disputes")
)
