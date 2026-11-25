#!/usr/bin/env bash
# F2: a payment replica dies while charging; the message goes to the other replica and there is a single charge.
source "$(dirname "$0")/lib.sh"
trap restore_defaults EXIT

title "F2 - a payment replica is killed in the middle of the charge"
step "Slow gateway (10 s) and a long payment timeout"
GATEWAY_LATENCY_MS=10000 GATEWAY_LATENCY_JITTER_MS=0 GATEWAY_TIMEOUT_MS=30000 GATEWAY_FAILURE_RATE=0 compose up -d --no-deps --scale payment=2 payment > /dev/null 2>&1
PAYMENT_TIMEOUT_SECONDS=120 compose up -d --no-deps main > /dev/null 2>&1
wait_payment 2
wait_for "main ready" 90 curl -sf "$MAIN_URL/health"

read -r email token < <(new_user f2)
id="$(api POST /subscriptions "$token" | json_field id)"
echo "    subscription $id"
sleep 4

busy=""
for replica in $(docker ps --format '{{.Names}}' | grep 'payment-'); do
  if docker logs --since 20s "$replica" 2>&1 | grep -q "message received"; then busy="$replica"; fi
done
step "Killing the replica that took the message: ${busy:-none found}"
[ -n "$busy" ] && docker kill "$busy" > /dev/null
wait_for "subscription becomes ACTIVE" 90 bash -c "[ \"\$(source '$PROJECT_ROOT/scripts/falhas/lib.sh'; subscription_status $token $id)\" = ACTIVE ]"

step "Evidence"
db main "SELECT id, status FROM subscriptions WHERE id = '$id'"
db payment "SELECT id, status, attempts FROM charges WHERE subscription_id = '$id'"
result "subscription status" "ACTIVE" "$(subscription_status "$token" "$id")"
result "charges for the subscription" "1" "$(db payment "SELECT count(*) FROM charges WHERE subscription_id = '$id'")"
[ -n "$busy" ] && docker start "$busy" > /dev/null
finish
