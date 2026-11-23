#!/usr/bin/env bash
# F5: 100 simultaneous enrollments for a championship with 10 slots: exactly 10 succeed.
source "$(dirname "$0")/lib.sh"

title "F5 - race for the last slots of a championship"
admin="$(admin_token)"
championship="$(api POST /championships "$admin" "{\"name\":\"F5 $RUN_ID\",\"slotsTotal\":10}" | json_field id)"
echo "    championship $championship"

step "Creating 100 teams"
teams=()
for i in $(seq 1 100); do
  teams+=("$(api POST /teams "$admin" "{\"name\":\"F5 $RUN_ID T$i\",\"city\":\"X\"}" | json_field id)")
done

step "Enrolling all of them at the same time"
results="$(mktemp -d)"
index=0
for team in "${teams[@]}"; do
  index=$((index + 1))
  ( printf '%s\n' "$(api_status POST "/championships/$championship/teams" "$admin" "{\"teamId\":\"$team\"}")" > "$results/$index" ) &
done
wait

created="$(cat "$results"/* | grep -c '^201$')"; full="$(cat "$results"/* | grep -c '^409$')"
rm -rf "$results"
step "Evidence"
db main "SELECT slots_total, slots_taken FROM championships WHERE id = '$championship'"
db main "SELECT count(*) AS enrollments FROM championship_teams WHERE championship_id = '$championship'" 2>/dev/null
result "responses 201" "10" "$created"
result "responses 409" "90" "$full"
result "slots taken" "10" "$(db main "SELECT slots_taken FROM championships WHERE id = '$championship'")"
finish
