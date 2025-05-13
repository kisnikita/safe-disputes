package services

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrUserNotFound = fmt.Errorf("user %w", ErrNotFound)
)
