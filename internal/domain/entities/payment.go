package entities

import (
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

type Payment struct {
	OrgId          string            `json:"org_id"`
	Id             string            `json:"id"`
	OrderId        string            `json:"order_id"`
	SubscriptionId string            `json:"subscription_id"`
	Status         PaymentStatus     `json:"status"`
	Currency       string            `json:"currency"`
	Amount         int               `json:"amount"`
	PspFee         int               `json:"psp_fee"`
	PlatformFee    int               `json:"platform_fee"`
	NetAmount      int               `json:"net_amount"`
	Metadata       map[string]string `json:"metadata"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}
