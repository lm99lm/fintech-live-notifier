#!/bin/sh
set -eu

curl --fail-with-body \
  --request POST \
  --header 'Content-Type: application/json' \
  --data '{"payment_id":"pay_42","account_id":"acct_7","amount_minor":12500,"currency":"USD","risk_score":82}' \
  "${NOTIFIER_URL:-http://localhost:8080}/payment-events"
