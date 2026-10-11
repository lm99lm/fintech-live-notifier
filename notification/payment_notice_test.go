package notification

import (
	"testing"
	"time"
)

func TestBuildPaymentNotice(t *testing.T) {
	now := time.Date(2026, 9, 4, 9, 30, 0, 0, time.UTC)
	tests := []struct {
		name       string
		risk       int
		wantKind   string
		wantAction string
		decision   string
	}{
		{name: "settled payment exposes receipt", risk: 24, wantKind: "payment.receipt_available", wantAction: "view_receipt", decision: "receipt_available"},
		{name: "high risk payment requires review", risk: 70, wantKind: "payment.review_required", wantAction: "open_review", decision: "review_required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notice := BuildPaymentNotice(PaymentEvent{PaymentID: "pay_42", Currency: "USD", RiskScore: tt.risk}, now)
			if notice.Kind != tt.wantKind || notice.Actions[0] != tt.wantAction || notice.Audit.Decision != tt.decision {
				t.Fatalf("got kind=%q action=%q decision=%q", notice.Kind, notice.Actions[0], notice.Audit.Decision)
			}
			if notice.Audit.CreatedAt != "2026-09-04T09:30:00Z" {
				t.Fatalf("unexpected audit timestamp %q", notice.Audit.CreatedAt)
			}
		})
	}
}
