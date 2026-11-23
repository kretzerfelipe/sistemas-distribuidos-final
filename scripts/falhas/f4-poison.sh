#!/usr/bin/env bash
# F4: a message that breaks the contract goes to the dead letter queue and the other messages keep flowing.
source "$(dirname "$0")/lib.sh"

title "F4 - poison message"
payment_before="$(queue_messages payment.dlq)"; main_before="$(queue_messages main.dlq)"; notification_before="$(queue_messages notification.dlq)"

step "Publishing invalid messages without the AMQP message_id property (as the management UI would)"
publish payment.requested '{"messageId":"x","type":"PaymentRequested"}'
publish payment.approved '{"nope":true}'
publish match.started 'not json at all'

step "Waiting for the dead letter queues"
wait_for "payment.dlq grows" 60 bash -c "[ \"\$(source '$PROJECT_ROOT/scripts/falhas/lib.sh'; queue_messages payment.dlq)\" -gt $payment_before ]"
wait_for "main.dlq grows" 60 bash -c "[ \"\$(source '$PROJECT_ROOT/scripts/falhas/lib.sh'; queue_messages main.dlq)\" -gt $main_before ]"
wait_for "notification.dlq grows" 60 bash -c "[ \"\$(source '$PROJECT_ROOT/scripts/falhas/lib.sh'; queue_messages notification.dlq)\" -gt $notification_before ]"

step "Other messages still flow: a new subscription is processed"
read -r email token < <(new_user f4)
id="$(api POST /subscriptions "$token" | json_field id)"
wait_for "subscription leaves PENDING" 60 bash -c "[ \"\$(source '$PROJECT_ROOT/scripts/falhas/lib.sh'; subscription_status $token $id)\" != PENDING ]"

step "Evidence"
echo "    payment.dlq=$(queue_messages payment.dlq) main.dlq=$(queue_messages main.dlq) notification.dlq=$(queue_messages notification.dlq)"
echo "    payment.q=$(queue_messages payment.q) main.q=$(queue_messages main.q) notification.q=$(queue_messages notification.q)"
result "poison left payment.q" "0" "$(queue_messages payment.q)"
result "poison left main.q" "0" "$(queue_messages main.q)"
result "subscription processed after the poison" "true" "$([ "$(subscription_status "$token" "$id")" != PENDING ] && echo true || echo false)"
finish
