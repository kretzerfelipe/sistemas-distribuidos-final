#!/usr/bin/env bash
# F6: RabbitMQ goes down for 30 s; services reconnect by themselves and nothing is lost.
source "$(dirname "$0")/lib.sh"
cleanup() { docker start futhub-rabbitmq-1 > /dev/null 2>&1; restore_defaults; }
trap cleanup EXIT

title "F6 - the broker is down for 30 seconds"
step "Long payment timeout, so the subscription does not expire during the outage"
PAYMENT_TIMEOUT_SECONDS=180 compose up -d --no-deps main > /dev/null 2>&1
wait_for "main ready" 90 curl -sf "$MAIN_URL/health"

step "Stopping RabbitMQ"
docker stop futhub-rabbitmq-1 > /dev/null
sleep 6
echo "    main health: $(curl -s "$MAIN_URL/health")"
echo "    notification health: $(curl -s "$NOTIFICATION_URL/health")"

step "Creating a subscription while the broker is down (main must still answer)"
read -r email token < <(new_user f6)
status="$(api_status POST /subscriptions "$token")"
echo "    POST /subscriptions -> $status"
sleep 3
echo "    outbox rows waiting in main: $(db main "SELECT count(*) FROM outbox WHERE published_at IS NULL")"
sleep 20

step "Starting RabbitMQ again"
docker start futhub-rabbitmq-1 > /dev/null
id="$(api GET /subscriptions/mine "$token" | json_field id)"
wait_for "main reconnects" 90 bash -c "curl -s $MAIN_URL/health | grep -q '\"rabbitmq\":\"up\"'"
wait_for "notification reconnects" 90 bash -c "curl -s $NOTIFICATION_URL/health | grep -q '\"rabbitmq\":\"up\"'"
wait_for "the subscription is processed" 90 bash -c "[ \"\$(source '$PROJECT_ROOT/scripts/falhas/lib.sh'; subscription_status $token $id)\" = ACTIVE ]"

step "Evidence"
echo "    outbox rows waiting in main: $(db main "SELECT count(*) FROM outbox WHERE published_at IS NULL")"
result "request accepted while the broker was down" "202" "$status"
result "subscription status" "ACTIVE" "$(subscription_status "$token" "$id")"
result "outbox drained" "0" "$(db main "SELECT count(*) FROM outbox WHERE published_at IS NULL")"
finish
