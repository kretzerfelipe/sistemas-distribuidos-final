#!/usr/bin/env bash
# F8: three payment replicas share the load; every subscription gets exactly one charge.
source "$(dirname "$0")/lib.sh"
trap restore_defaults EXIT

title "F8 - horizontal scale of payment"
step "Scaling payment to 3 replicas (failures off for a clean count)"
GATEWAY_FAILURE_RATE=0 compose up -d --no-deps --scale payment=3 payment > /dev/null 2>&1
wait_payment 3
sleep 25

step "Creating 30 subscriptions"
ids=(); tokens=()
for i in $(seq 1 30); do
  read -r email token < <(new_user f8)
  ids+=("$(api POST /subscriptions "$token" | json_field id)"); tokens+=("$token")
done
list="$(printf "'%s'," "${ids[@]}")"; list="${list%,}"
wait_for "all 30 charges are decided" 120 bash -c "[ \"\$(source '$PROJECT_ROOT/scripts/falhas/lib.sh'; db payment \"SELECT count(*) FROM charges WHERE subscription_id IN ($list) AND status <> 'PROCESSING'\")\" = 30 ]"
sleep 5

step "Evidence"
for replica in $(docker ps --format '{{.Names}}' | grep 'payment-'); do
  echo "    $replica handled $(docker logs "$replica" 2>&1 | grep -c 'charge approved') charge(s)"
done
db payment "SELECT status, count(*) FROM charges WHERE subscription_id IN ($list) GROUP BY status"
result "charges" "30" "$(db payment "SELECT count(*) FROM charges WHERE subscription_id IN ($list)")"
result "subscriptions with more than one charge" "0" "$(db payment "SELECT count(*) FROM (SELECT subscription_id FROM charges WHERE subscription_id IN ($list) GROUP BY subscription_id HAVING count(*) > 1) d")"
finish
