package payments

import (
	"payloop/internal/domain/common"
	"time"
)

type PaymentStatus string

const (
	PaymentStatusPending       PaymentStatus = "pending"
	PaymentStatusFailed        PaymentStatus = "failed"
	PaymentStatusSucceeded     PaymentStatus = "succeeded"
	PaymentStatusRefunded      PaymentStatus = "refunded"
	PaymentStatusPartialRefund PaymentStatus = "partial_refund"
	PaymentStatusCancelled     PaymentStatus = "cancelled"
	PaymentStatusExpired       PaymentStatus = "expired"
	PaymentStatusFraudulent    PaymentStatus = "fraudulent"
)

type ChargeResult struct {
	Psp         common.Gateway `json:"psp"`
	Amount      int64          `json:"amount"`
	Status      PaymentStatus  `json:"status"`
	ErrorReason string         `json:"error_reason"`
	ErrorCode   string         `json:"error_code"`
	Currency    string         `json:"currency"`
	PspId       string         `json:"psp_id"`
	Reference   string         `json:"reference"`
	ProcessedAt time.Time      `json:"processed_at"`
	RawData     string         `json:"raw_data"`
}
