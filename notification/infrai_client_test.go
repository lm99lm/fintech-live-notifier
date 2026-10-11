package notification

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestPublishDecodesBusinessErrorBeforeStatus(t *testing.T) {
	client := &InfraiClient{
		BaseURL: "https://example.test",
		APIKey:  "test-key",
		HTTP: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost || req.Header.Get("Idempotency-Key") != "pay_42:publish" {
				t.Fatalf("unexpected request method=%s key=%s", req.Method, req.Header.Get("Idempotency-Key"))
			}
			return &http.Response{StatusCode: 400, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":false,"data":null,"error":{"code":"BAD_PAYMENT","message":"payment rejected"},"metadata":{}}`))}, nil
		})},
	}
	err := client.Publish(context.Background(), "account.acct_7", "acct_7", "pay_42", Notice{})
	got, ok := err.(*InfraiError)
	if !ok || got.Code != "BAD_PAYMENT" || got.HTTPStatus != 400 {
		t.Fatalf("expected typed business error, got %#v", err)
	}
}

func TestCreateChannelRetriesRateLimit(t *testing.T) {
	attempts := 0
	var slept time.Duration
	client := &InfraiClient{
		BaseURL: "https://example.test",
		APIKey:  "test-key",
		HTTP: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				header := make(http.Header)
				header.Set("Retry-After", "2")
				return &http.Response{StatusCode: 429, Header: header, Body: io.NopCloser(strings.NewReader(`{"ok":false,"data":null,"error":{"code":"RATE_LIMITED","message":"retry later"},"metadata":{}}`))}, nil
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true,"data":{},"error":null,"metadata":{}}`))}, nil
		})},
		Sleep: func(_ context.Context, delay time.Duration) error { slept = delay; return nil },
	}
	if err := client.CreateChannel(context.Background(), "account.acct_7", "pay_42"); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || slept != 2*time.Second {
		t.Fatalf("attempts=%d slept=%s", attempts, slept)
	}
}
