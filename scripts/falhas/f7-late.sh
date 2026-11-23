#!/usr/bin/env bash
# F7: the approval arrives after the subscription expired; the charge must be refunded.
source "$(dirname "$0")/lib.sh"
trap restore_defaults EXIT

title "F7 - late approval"
step "Gateway slower (12 s) than the payment timeout (5 s)"
GATEWAY_LATENCY_MS=12000 GATEWAY_LATENCY_JITTER_MS=0 GATEWAY_TIMEOUT_MS=30000 GATEWAY_FAILURE_RATE=0 compose up -d --no-deps --scale payment=2 payment > /dev/null 2>&1
PAYMENT_TIMEOUT_SECONDS=5 compose up -d --no-deps main > /dev/null 2>&1
wait_payment 2
wait_for "main ready" 90 curl -sf "$MAIN_URL/health"

read -r email token < <(new_user f7)
id="$(api POST /subscriptions "$token" | json_field id)"
echo "    subscription $id"
wait_for "subscription expires" 30 bash -c "[ \"\$(source '$PROJECT_ROOT/scripts/falhas/lib.sh'; subscription_status $token $id)\" = EXPIRED ]"
wait_for "the late charge is refunded" 90 bash -c "[ \"\$(source '$PROJECT_ROOT/scripts/falhas/lib.sh'; db payment \"SELECT status FROM charges WHERE subscription_id = '$id'\")\" = REFUNDED ]"
sleep 3

step "Evidence"
db main "SELECT id, status, reason FROM subscriptions WHERE id = '$id'"
db payment "SELECT id, status, refund_pending FROM charges WHERE subscription_id = '$id'"
result "subscription status" "EXPIRED" "$(subscription_status "$token" "$id")"
result "charge status" "REFUNDED" "$(db payment "SELECT status FROM charges WHERE subscription_id = '$id'")"
result "user is not premium" "EXPIRED" "$(api GET /subscriptions/mine "$token" | json_field status)"
finish
