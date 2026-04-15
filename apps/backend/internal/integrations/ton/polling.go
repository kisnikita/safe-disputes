package ton

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kisnikita/safe-disputes/backend/internal/services"
	tonapi "github.com/tonkeeper/tonapi-go"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"go.uber.org/zap"
)

func (m TonAPIMonitor) WaitForSuccess(ctx context.Context, boc string) error {
	msgHash, err := normalizedExternalMessageHash(boc)
	if err != nil {
		return fmt.Errorf("%w: %v", services.ErrInvalidBOC, err)
	}

	pollCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	m.logger.Info("start TON tx tracking", zap.String("msg_hash", msgHash))

	for {
		trace, err := m.client.GetTrace(pollCtx, tonapi.GetTraceParams{TraceID: msgHash})
		if err != nil {
			if isNotFound(err) {
				if err = waitWithContext(pollCtx, m.pollInterval); err != nil {
					return mapWaitError(err)
				}
				continue
			}
			return mapWaitError(err)
		}

		// Finalized trace is considered immutable (included in masterchain finality path).
		if !isFinalized(*trace) {
			if err = waitWithContext(pollCtx, m.pollInterval); err != nil {
				return mapWaitError(err)
			}
			continue
		}

		if reason := firstTraceFailure(*trace); reason != "" {
			return fmt.Errorf("%w: %s", services.ErrTxFailed, reason)
		}

		m.logger.Info("TON tx confirmed", zap.String("msg_hash", msgHash))
		return nil
	}
}

func normalizedExternalMessageHash(boc string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(boc)
	if err != nil {
		return "", err
	}

	msgCell, err := cell.FromBOC(raw)
	if err != nil {
		return "", fmt.Errorf("invalid boc payload: %w", err)
	}

	var msg tlb.Message
	if err = tlb.LoadFromCell(&msg, msgCell.BeginParse()); err != nil {
		return "", fmt.Errorf("failed to parse message from boc: %w", err)
	}
	if msg.MsgType != tlb.MsgTypeExternalIn {
		return "", fmt.Errorf("boc does not contain external inbound message")
	}

	return hex.EncodeToString(msg.AsExternalIn().NormalizedHash()), nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}

	var statusErr *tonapi.ErrorStatusCode
	if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusNotFound {
		return true
	}

	return strings.Contains(err.Error(), "code 404:")
}

func waitWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func mapWaitError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return services.ErrTxNotFinalized
	}
	return fmt.Errorf("%w: %v", services.ErrTxMonitorUnavailable, err)
}

func isFinalized(trace tonapi.Trace) bool {
	if emulated, ok := trace.Emulated.Get(); ok && emulated {
		return false
	}

	for _, msg := range trace.Transaction.OutMsgs {
		if msg.MsgType == "int_msg" {
			return false
		}
	}

	for _, child := range trace.Children {
		if !isFinalized(child) {
			return false
		}
	}
	return true
}

func firstTraceFailure(trace tonapi.Trace) string {
	if reason := transactionFailureReason(trace.Transaction); reason != "" {
		return reason
	}

	for _, child := range trace.Children {
		if reason := firstTraceFailure(child); reason != "" {
			return reason
		}
	}
	return ""
}

func transactionFailureReason(tx tonapi.Transaction) string {
	computeExit := "n/a"
	if compute, ok := tx.ComputePhase.Get(); ok {
		if exitCode, ok := compute.ExitCode.Get(); ok {
			computeExit = fmt.Sprintf("%d", exitCode)
		}
	}

	actionCode := "n/a"
	actionSuccess := "n/a"
	if action, ok := tx.ActionPhase.Get(); ok {
		actionCode = fmt.Sprintf("%d", action.ResultCode)
		actionSuccess = fmt.Sprintf("%t", action.Success)
	}

	computeSuccess := "n/a"
	if compute, ok := tx.ComputePhase.Get(); ok {
		if success, ok := compute.Success.Get(); ok {
			computeSuccess = fmt.Sprintf("%t", success)
		}
	}

	hasFailure :=
		!tx.Success ||
			tx.Aborted ||
			(computeSuccess == "false") ||
			(actionSuccess == "false") ||
			(actionCode != "n/a" && actionCode != "0")

	if !hasFailure {
		return ""
	}

	return fmt.Sprintf(
		"tx_hash=%s account=%s success=%t aborted=%t compute_success=%s compute_exit=%s action_success=%s action_result=%s",
		tx.Hash,
		tx.Account.Address,
		tx.Success,
		tx.Aborted,
		computeSuccess,
		computeExit,
		actionSuccess,
		actionCode,
	)
}
