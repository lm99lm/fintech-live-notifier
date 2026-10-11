package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"example.com/fintech-live-notifier/notification"
)

type server struct {
	infrai *notification.InfraiClient
	now    func() time.Time
}

type deliveredNotice struct {
	Channel      string              `json:"channel"`
	Notification notification.Notice `json:"notification"`
	ClientToken  string              `json:"client_token"`
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	s := &server{infrai: &notification.InfraiClient{BaseURL: notification.DefaultBaseURL, APIKey: key}, now: time.Now}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /payment-events", s.paymentEvent)
	addr := envOr("LISTEN_ADDR", ":8080")
	log.Printf("fintech notifier listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (s *server) paymentEvent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var event notification.PaymentEvent
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payment event"})
		return
	}
	if event.PaymentID == "" || event.AccountID == "" || event.Currency == "" || event.Amount <= 0 || event.RiskScore < 0 || event.RiskScore > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payment_id, account_id, positive amount_minor, currency, and risk_score 0..100 are required"})
		return
	}
	channel := "account." + event.AccountID
	notice := notification.BuildPaymentNotice(event, s.now())
	if err := s.infrai.CreateChannel(r.Context(), channel, event.PaymentID); err != nil {
		s.writeInfraiError(w, err)
		return
	}
	if err := s.infrai.Publish(r.Context(), channel, event.AccountID, event.PaymentID, notice); err != nil {
		s.writeInfraiError(w, err)
		return
	}
	token, err := s.infrai.IssueToken(r.Context(), event.AccountID, channel, event.PaymentID)
	if err != nil {
		s.writeInfraiError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, deliveredNotice{Channel: channel, Notification: notice, ClientToken: token.Token})
}

func (s *server) writeInfraiError(w http.ResponseWriter, err error) {
	var rejected *notification.InfraiError
	if errors.As(err, &rejected) && rejected.HTTPStatus >= 400 && rejected.HTTPStatus < 500 {
		writeJSON(w, rejected.HTTPStatus, map[string]string{"error": rejected.Code, "message": rejected.Message})
		return
	}
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": "notification delivery failed"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
