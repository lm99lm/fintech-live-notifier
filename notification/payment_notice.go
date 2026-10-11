package notification

import (
	"fmt"
	"time"
)

type PaymentEvent struct {
	PaymentID string `json:"payment_id"`
	AccountID string `json:"account_id"`
	Amount    int64  `json:"amount_minor"`
	Currency  string `json:"currency"`
	RiskScore int    `json:"risk_score"`
}

type Notice struct {
	PaymentID string   `json:"payment_id"`
	Kind      string   `json:"kind"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Actions   []string `json:"actions"`
	Audit     Audit    `json:"audit"`
}

type Audit struct {
	Rule      string `json:"rule"`
	Decision  string `json:"decision"`
	CreatedAt string `json:"created_at"`
}

func BuildPaymentNotice(event PaymentEvent, now time.Time) Notice {
	notice := Notice{
		PaymentID: event.PaymentID,
		Kind:      "payment.receipt_available",
		Title:     "Payment complete",
		Body:      fmt.Sprintf("Your %s payment is complete.", event.Currency),
		Actions:   []string{"view_receipt"},
		Audit: Audit{
			Rule:      "risk_score_v1",
			Decision:  "receipt_available",
			CreatedAt: now.UTC().Format(time.RFC3339),
		},
	}
	if event.RiskScore >= 70 {
		notice.Kind = "payment.review_required"
		notice.Title = "Payment needs review"
		notice.Body = fmt.Sprintf("Review the %s payment before taking further action.", event.Currency)
		notice.Actions = []string{"open_review"}
		notice.Audit.Decision = "review_required"
	}
	return notice
}
