package models

import (
	"errors"
	"fmt"
	"strconv"
)

const NanoPerTON int64 = 1_000_000_000

var ErrNegativeAmount = errors.New("nano amount must be positive")

func ParsePositiveNano(value string) (int64, error) {
	nano, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid nano amount: %w", err)
	}
	if nano <= 0 {
		return 0, ErrNegativeAmount
	}
	return nano, nil
}
