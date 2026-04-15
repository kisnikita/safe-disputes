package ton

import (
	"fmt"
	"strings"
	"time"

	tonapi "github.com/tonkeeper/tonapi-go"

	"github.com/kisnikita/safe-disputes/backend/pkg/log"
)

type MonitorConfig struct {
	Token        string
	Network      string
	PollInterval time.Duration
	Timeout      time.Duration
}

type TonAPIMonitor struct {
	logger       log.Logger
	client       *tonapi.Client
	pollInterval time.Duration
	timeout      time.Duration
}

func NewTonAPIMonitor(logger log.Logger, cfg MonitorConfig) (TonAPIMonitor, error) {
	if logger == nil {
		return TonAPIMonitor{}, fmt.Errorf("logger is nil")
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 1500 * time.Millisecond
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	serverURL := tonapi.TestnetTonApiURL
	if strings.EqualFold(cfg.Network, "mainnet") {
		serverURL = tonapi.TonApiURL
	}

	client, err := tonapi.NewClient(serverURL, tonapi.WithToken(cfg.Token))
	if err != nil {
		return TonAPIMonitor{}, fmt.Errorf("failed to init tonapi client: %w", err)
	}

	return TonAPIMonitor{
		logger:       logger,
		client:       client,
		pollInterval: cfg.PollInterval,
		timeout:      cfg.Timeout,
	}, nil
}
