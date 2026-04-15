package services

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound             = errors.New("not found")
	ErrUserNotFound         = fmt.Errorf("user %w", ErrNotFound)
	ErrMinimalAmount        = errors.New("amount is less than opponent's minimum disputes amount")
	ErrUnready              = errors.New("not ready for disputes")
	ErrSelfOpponent         = errors.New("creator and opponent must be different")
	ErrInvalidBOC           = errors.New("invalid transaction boc")
	ErrTxFailed             = errors.New("transaction failed")
	ErrTxNotFinalized       = errors.New("transaction not finalized in time")
	ErrTxMonitorUnavailable = errors.New("transaction monitor unavailable")
)
