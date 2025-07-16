package entities

import "time"

type Payment struct {
	ErrorEnum string
	IsDefault bool
	ID        *string  `json:"correlationId"`
	Amount    *float64 `json:"amount"`
}

type PaymentToSend struct {
	ID          string  `json:"correlationId"`
	Amount      float64 `json:"amount"`
	RequestedAt string  `json:"requestedAt"` //2025-07-15T12:34:56.000Z
}

func (p *Payment) ToPaymentToSend() *PaymentToSend {
	t := time.Now().UTC()

	return &PaymentToSend{
		ID:          *p.ID,
		Amount:      *p.Amount,
		RequestedAt: t.Format("2006-01-02T15:04:05.000Z"),
	}
}

type PaymentSummary struct {
	Default  Summary `json:"default"`
	Fallback Summary `json:"fallback"`
}

type Summary struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

func (p *PaymentToSend) GetUnixTimestamp() int64 {
	t, err := time.Parse("2006-01-02T15:04:05.000Z", p.RequestedAt)
	if err != nil {
		return 0
	}
	return t.Unix()
}
