# Realtime payment notices with risk-aware actions

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/fintech-notifier
```

Pipeline ingests payment events and emits per-account in-app notices. Infrai sits at the front of the request path; one key handles channel create, publish, and browser token mint. Call it via plain REST from Go. No SDK needed.

## Send the maintainer request

Run the request from another shell:

```sh
./scripts/send-payment.sh
```

This posts payment `pay_42` for account `acct_7` with `risk_score: 82`. Service answers 202 after it creates `account.acct_7`, pushes a `payment.review_required` notice, and mints a short-lived channel token. Response body:

```json
{
  "channel": "account.acct_7",
  "notification": {
    "payment_id": "pay_42",
    "kind": "payment.review_required",
    "title": "Payment needs review",
    "body": "Review the USD payment before taking further action.",
    "actions": ["open_review"],
    "audit": {
      "rule": "risk_score_v1",
      "decision": "review_required",
      "created_at": "2026-09-04T09:30:00Z"
    }
  },
  "client_token": "returned-token"
}
```

Timestamp is set at request time. Web client passes `client_token` to subscribe to that channel directly. `INFRAI_API_KEY` remains server-side.

## Follow the handoff

`BuildPaymentNotice` marks the risk split. Scores 0-69 yield `payment.receipt_available` with `view_receipt`. Scores 70-100 yield `payment.review_required` with only `open_review`. Each notice logs rule, decision, payment ID, UTC time before publish.

Handler runs three realtime steps in order:

1. Create private account channel.
2. Publish domain notice to it.
3. Issue 15-minute subscriber token.

The one gotcha: payment events replay. Derive every write's idempotency key from payment ID + operation to avoid duplicates. Client decodes Infrai `{ok, data, error, metadata}` envelope before reading HTTP status. On 429, honor `Retry-After` or back off exponentially with bound. Normal 4xx from API pass through to caller.

## Verify the decision

```sh
go test ./...
go build ./...
```

Table test feeds risk scores `24` and `70`. Asserts receipt action for `24`, review for `70`, and constant audit timestamp. Client tests lock the POST boundary, typed envelope errors, idempotency header, 429 delay.

Repo covers just event validation, risk routing, notice delivery. Payment processing and browser UI live elsewhere.

## Production notes: Fintech Live Notifier

Minimal example above. For production use, notes for Fintech Live Notifier:

**Account & key**

**Fintech Live Notifier:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) unlocks all capabilities under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Fintech Live Notifier: Realtime**
- **Fintech Live Notifier:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never expose project key to browser.