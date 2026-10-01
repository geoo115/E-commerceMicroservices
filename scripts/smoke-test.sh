#!/usr/bin/env bash
# End-to-end smoke test against a running stack (docker compose up -d --wait).
# Exercises the full order saga plus the main security rules.
# Requirements: curl, jq, docker compose.
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
API="$BASE_URL/api/v1"
ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin-password-change-me}"
RUN_ID="$(date +%s)$RANDOM"

pass=0
green() { printf '\033[32m%s\033[0m\n' "$*"; }
red() { printf '\033[31m%s\033[0m\n' "$*"; }
step() { printf '\n\033[1m▶ %s\033[0m\n' "$*"; }
ok() { pass=$((pass + 1)); green "  ✔ $*"; }
fail() { red "  ✘ $*"; exit 1; }

# request METHOD PATH [JSON] [TOKEN] -> sets $STATUS and $BODY
request() {
  local method=$1 path=$2 data=${3:-} token=${4:-}
  local args=(-s -o /tmp/smoke_body -w '%{http_code}' -X "$method" "$API$path" -H 'Content-Type: application/json')
  [[ -n $token ]] && args+=(-H "Authorization: Bearer $token")
  [[ -n $data ]] && args+=(-d "$data")
  STATUS=$(curl "${args[@]}")
  BODY=$(cat /tmp/smoke_body)
}

expect_status() {
  local want=$1 what=$2
  [[ $STATUS == "$want" ]] || fail "$what: expected HTTP $want, got $STATUS: $BODY"
  ok "$what (HTTP $STATUS)"
}

# wait_for DESCRIPTION JQ_FILTER EXPECTED METHOD PATH [TOKEN]
wait_for() {
  local what=$1 filter=$2 want=$3 method=$4 path=$5 token=${6:-} got=""
  for _ in $(seq 1 60); do # up to 15s: CI runners are slower than a laptop
    request "$method" "$path" "" "$token"
    got=$(jq -r "$filter" <<<"$BODY")
    [[ $got == "$want" ]] && { ok "$what → $want"; return; }
    sleep 0.25
  done
  fail "$what: expected $want, last value $got ($BODY)"
}

signup_and_login() { # signup_and_login USERNAME -> echoes token
  local user=$1 email="$1@example.com" code
  request POST /auth/signup "{\"username\":\"$user\",\"email\":\"$email\",\"password\":\"password123\"}"
  [[ $STATUS == 201 ]] || fail "signup $user: $STATUS $BODY"
  code=$(docker compose exec -T redis redis-cli GET "verify:$email" | tr -d '\r')
  request POST /auth/verify-email "{\"email\":\"$email\",\"code\":\"$code\"}"
  [[ $STATUS == 200 ]] || fail "verify $user: $STATUS $BODY"
  request POST /auth/login "{\"username\":\"$user\",\"password\":\"password123\"}"
  [[ $STATUS == 200 ]] || fail "login $user: $STATUS $BODY"
  jq -r .access_token <<<"$BODY"
}

step "Health"
curl -sf "$BASE_URL/healthz" >/dev/null && ok "gateway is healthy"

step "Admin sets up the catalog"
request POST /auth/login "{\"username\":\"$ADMIN_USERNAME\",\"password\":\"$ADMIN_PASSWORD\"}"
expect_status 200 "admin login"
ADMIN=$(jq -r .access_token <<<"$BODY")

request POST /categories "{\"name\":\"Keyboards $RUN_ID\"}" "$ADMIN"
expect_status 201 "create category"
CATEGORY=$(jq -r .id <<<"$BODY")

request POST /products "{\"name\":\"Mechanical Keyboard\",\"price_cents\":12999,\"category_id\":$CATEGORY,\"initial_stock\":5}" "$ADMIN"
expect_status 201 "create product A (5 in stock)"
PRODUCT_A=$(jq -r .id <<<"$BODY")

request POST /products "{\"name\":\"Limited Keycaps\",\"price_cents\":4500,\"category_id\":$CATEGORY,\"initial_stock\":1}" "$ADMIN"
expect_status 201 "create product B (1 in stock)"
PRODUCT_B=$(jq -r .id <<<"$BODY")

wait_for "stock of A initialised via product.created event" .available 5 GET "/products/$PRODUCT_A"

step "Customer signs up"
request POST /auth/signup "{\"username\":\"unverified_$RUN_ID\",\"email\":\"unverified_$RUN_ID@example.com\",\"password\":\"password123\"}"
expect_status 201 "signup"
request POST /auth/login "{\"username\":\"unverified_$RUN_ID\",\"password\":\"password123\"}"
expect_status 409 "login before email verification is refused"
request POST /auth/login "{\"username\":\"unverified_$RUN_ID\",\"password\":\"wrong-password\"}"
expect_status 401 "wrong password"

ALICE=$(signup_and_login "alice_$RUN_ID")
ok "alice verified her email and logged in"
BOB=$(signup_and_login "bob_$RUN_ID")
ok "bob verified his email and logged in"

step "Authorization rules"
request GET /cart
expect_status 401 "cart without token"
request POST /products "{\"name\":\"x\",\"price_cents\":1,\"category_id\":$CATEGORY}" "$ALICE"
expect_status 403 "customer cannot create products"

step "Cart → checkout (happy path)"
request POST /cart/items "{\"product_id\":$PRODUCT_A,\"quantity\":2}" "$ALICE"
expect_status 200 "add 2 × A to cart"
request POST /cart/checkout "" "$ALICE"
expect_status 201 "checkout"
ORDER=$(jq -r .id <<<"$BODY")
[[ $(jq -r .total_cents <<<"$BODY") == 25998 ]] && ok "total priced from the catalog (25998)" || fail "wrong total: $BODY"
request GET /cart "" "$ALICE"
[[ $(jq '.items | length' <<<"$BODY") == 0 ]] && ok "cart emptied after checkout" || fail "cart not empty: $BODY"

wait_for "order confirmed after inventory.reserved" .status confirmed GET "/orders/$ORDER" "$ALICE"
request GET "/inventory/$PRODUCT_A" "" "$ADMIN"
[[ $(jq -c '[.available,.reserved]' <<<"$BODY") == "[3,2]" ]] && ok "2 units reserved, 3 available" || fail "unexpected stock: $BODY"

request GET "/orders/$ORDER" "" "$BOB"
expect_status 404 "bob cannot see alice's order"
request POST "/orders/$ORDER/payment" '{"method":"card","card_last_four":"4242"}' "$BOB"
expect_status 403 "bob cannot pay alice's order"

request POST "/orders/$ORDER/payment" '{"method":"card","card_last_four":"4242"}' "$ALICE"
expect_status 201 "payment succeeded"
[[ $(jq -r .amount_cents <<<"$BODY") == 25998 ]] && ok "charged the order total" || fail "wrong amount: $BODY"
wait_for "order paid after payment.succeeded" .status paid GET "/orders/$ORDER" "$ALICE"
request POST "/orders/$ORDER/payment" '{"method":"card","card_last_four":"4242"}' "$ALICE"
expect_status 409 "second payment is rejected"
wait_for "reserved units committed as sold" .reserved 0 GET "/inventory/$PRODUCT_A" "$ADMIN"

request PATCH "/orders/$ORDER/status" '{"status":"shipped"}' "$ADMIN"
expect_status 200 "admin marks the order shipped"

step "Client-supplied prices are ignored"
request POST /orders "{\"items\":[{\"product_id\":$PRODUCT_A,\"quantity\":1,\"price\":1}]}" "$ALICE"
expect_status 201 "order with a tampered price field"
TAMPERED=$(jq -r .id <<<"$BODY")
[[ $(jq -r .total_cents <<<"$BODY") == 12999 ]] && ok "total is the catalog price (12999), not 1" || fail "price tampering: $BODY"

step "Compensation: customer cancels a confirmed order"
wait_for "tampered-price order confirmed" .status confirmed GET "/orders/$TAMPERED" "$ALICE"
request POST "/orders/$TAMPERED/cancel" '{"reason":"changed my mind"}' "$ALICE"
expect_status 200 "cancel order"
wait_for "stock released by order.cancelled" .available 3 GET "/inventory/$PRODUCT_A" "$ADMIN"

step "Compensation: inventory rejects an order"
request POST /orders "{\"items\":[{\"product_id\":$PRODUCT_B,\"quantity\":2}]}" "$BOB"
expect_status 201 "order 2 × B (only 1 in stock)"
REJECTED=$(jq -r .id <<<"$BODY")
wait_for "order cancelled after inventory.rejected" .status cancelled GET "/orders/$REJECTED" "$BOB"
request GET "/orders/$REJECTED" "" "$BOB"
ok "reason: $(jq -r .cancellation_reason <<<"$BODY")"

step "Declined payment"
request POST /orders "{\"items\":[{\"product_id\":$PRODUCT_B,\"quantity\":1}]}" "$BOB"
DECLINED=$(jq -r .id <<<"$BODY")
wait_for "order confirmed" .status confirmed GET "/orders/$DECLINED" "$BOB"
request POST "/orders/$DECLINED/payment" '{"method":"card","card_last_four":"0002"}' "$BOB"
expect_status 402 "card ending 0002 is declined"
request POST "/orders/$DECLINED/payment" '{"method":"paypal"}' "$BOB"
expect_status 201 "retry with another method succeeds"

step "Reviews update the product rating (review.created event)"
request POST "/products/$PRODUCT_A/reviews" '{"rating":4,"comment":"Great switches"}' "$ALICE"
expect_status 201 "alice reviews product A"
REVIEW=$(jq -r .id <<<"$BODY")
request POST "/products/$PRODUCT_A/reviews" '{"rating":5}' "$ALICE"
expect_status 409 "one review per user and product"
wait_for "rating aggregated by product-service" .rating_count 1 GET "/products/$PRODUCT_A"
request DELETE "/reviews/$REVIEW" "" "$BOB"
expect_status 403 "bob cannot delete alice's review"

echo
green "All $pass checks passed."
