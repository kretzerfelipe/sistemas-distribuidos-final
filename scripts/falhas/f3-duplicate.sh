#!/usr/bin/env bash
# F3: the same message (same messageId) is published 5 times and must have a single effect.
source "$(dirname "$0")/lib.sh"

title "F3 - duplicated message"
subscription_id="$(new_uuid)"; user_id="$(new_uuid)"; message_id="$(new_uuid)"
body="$(envelope PaymentRequested "{\"subscriptionId\":\"$subscription_id\",\"userId\":\"$user_id\",\"amountCents\":1990}" "$message_id")"

step "Publishing the same PaymentRequested 5 times"
for i in 1 2 3 4 5; do publish payment.requested "$body" "$message_id"; done
wait_for "the charge exists" 60 bash -c "[ \"\$(source '$PROJECT_ROOT/scripts/falhas/lib.sh'; db payment \"SELECT count(*) FROM charges WHERE subscription_id = '$subscription_id'\")\" = 1 ]"
sleep 8

step "Evidence"
db payment "SELECT id, status, attempts FROM charges WHERE subscription_id = '$subscription_id'"
db payment "SELECT count(*) AS processed_messages FROM processed_messages WHERE message_id = '$message_id'"
for replica in $(docker ps --format '{{.Names}}' | grep 'payment-'); do
  echo "    $replica: $(docker logs --since 60s "$replica" 2>&1 | grep -c 'duplicate message ignored') duplicate(s) ignored"
done
result "charges" "1" "$(db payment "SELECT count(*) FROM charges WHERE subscription_id = '$subscription_id'")"
result "processed message rows" "1" "$(db payment "SELECT count(*) FROM processed_messages WHERE message_id = '$message_id'")"
result "approvals published" "1" "$(db payment "SELECT count(*) FROM outbox WHERE type = 'PaymentApproved' AND payload->>'subscriptionId' = '$subscription_id'")"
finish
