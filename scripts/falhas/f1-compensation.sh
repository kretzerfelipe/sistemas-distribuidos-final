#!/usr/bin/env bash
# F1: the payment is rejected, so the subscription must be canceled and the user stays out of premium.
source "$(dirname "$0")/lib.sh"
trap restore_defaults EXIT

title "F1 - compensation after a rejected payment"
step "Starting payment with GATEWAY_FAILURE_RATE=1.0"
GATEWAY_FAILURE_RATE=1.0 GATEWAY_MAX_ATTEMPTS=1 compose up -d --no-deps payment > /dev/null 2>&1
wait_payment 2

step "Creating a user and a subscription"
read -r email token < <(new_user f1)
id="$(api POST /subscriptions "$token" | json_field id)"
echo "    subscription $id"
wait_for "subscription leaves PENDING" 60 bash -c "[ \"\$(source '$PROJECT_ROOT/scripts/falhas/lib.sh'; subscription_status $token $id)\" != PENDING ]"

step "Evidence"
db main "SELECT id, status, reason FROM subscriptions WHERE id = '$id'"
user_id="$(db main "SELECT user_id FROM subscriptions WHERE id = '$id'")"
sleep 2
db notification "SELECT type, message FROM notifications WHERE user_id = '$user_id'"
db notification "SELECT user_id, active FROM premium_users WHERE user_id = '$user_id'"

result "subscription status" "CANCELED" "$(subscription_status "$token" "$id")"
result "cancellation reason" "PAYMENT_REJECTED" "$(db main "SELECT reason FROM subscriptions WHERE id = '$id'")"
result "user is not premium" "false" "$(api GET /subscriptions/mine "$token" | sed -n 's/.*"premium":\([a-z]*\).*/\1/p')"
finish
