#!/usr/bin/env bash
# Shared helpers for the failure scenarios. Requires bash, curl and docker.

MAIN_URL="${MAIN_URL:-http://localhost:8080}"
NOTIFICATION_URL="${NOTIFICATION_URL:-http://localhost:8082}"
RABBIT_URL="${RABBIT_URL:-http://localhost:15672}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@futhub.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin12345}"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUN_ID="$(date +%s)$RANDOM"

export MSYS_NO_PATHCONV=1

title() { printf '\n=== %s ===\n' "$*"; }
step() { printf '\n--> %s\n' "$*"; }

compose() { (cd "$PROJECT_ROOT" && docker compose "$@"); }

new_uuid() {
  if [ -r /proc/sys/kernel/random/uuid ]; then cat /proc/sys/kernel/random/uuid; return; fi
  powershell.exe -NoProfile -Command "[guid]::NewGuid().ToString()" | tr -d '\r'
}

json_field() { sed -n "s/.*\"$1\":\"\([^\"]*\)\".*/\1/p" | head -1; }

api() { # method path token [json body]
  local method="$1" path="$2" token="${3:-}" body="${4:-}"
  local args=(-s -X "$method" "$MAIN_URL$path" -H 'Content-Type: application/json')
  [ -n "$token" ] && args+=(-H "Authorization: Bearer $token")
  [ -n "$body" ] && args+=(-d "$body")
  curl "${args[@]}"
}

api_status() { # method path token [json body]
  local method="$1" path="$2" token="${3:-}" body="${4:-}"
  local args=(-s -o /dev/null -w '%{http_code}' -X "$method" "$MAIN_URL$path" -H 'Content-Type: application/json')
  [ -n "$token" ] && args+=(-H "Authorization: Bearer $token")
  [ -n "$body" ] && args+=(-d "$body")
  curl "${args[@]}"
}

login() { api POST /auth/login "" "{\"email\":\"$1\",\"password\":\"$2\"}" | json_field accessToken; }

admin_token() { login "$ADMIN_EMAIL" "$ADMIN_PASSWORD"; }

new_user() { # prefix -> prints "email token"
  local email="$1.$RUN_ID.$RANDOM@example.com"
  api POST /auth/register "" "{\"name\":\"Teste $1\",\"email\":\"$email\",\"password\":\"senha12345\",\"accountType\":\"USER\"}" > /dev/null
  echo "$email $(login "$email" senha12345)"
}

db() { # service sql
  local container
  container="$(docker ps -q --filter "name=instance-futhub-$1" | head -1)"
  docker exec -e PGPASSWORD=futhub123 "$container" psql -U futhub -h localhost -d "db_$1" -At -F ' | ' -c "$2"
}

rabbit() { curl -s -u guest:guest -H 'content-type: application/json' "$@"; }

queue_messages() { rabbit "$RABBIT_URL/api/queues/%2F/$1" | sed -n 's/.*"messages":\([0-9]*\).*/\1/p' | head -1; }

publish() { # routing_key envelope [amqp message_id]
  local properties='{}'
  [ -n "${3:-}" ] && properties="{\"message_id\":\"$3\"}"
  local payload
  payload="${2//\\/\\\\}"
  payload="${payload//\"/\\\"}"
  rabbit -X POST "$RABBIT_URL/api/exchanges/%2F/futhub.events/publish" \
    -d "{\"properties\":$properties,\"routing_key\":\"$1\",\"payload\":\"$payload\",\"payload_encoding\":\"string\"}" > /dev/null
}

envelope() { # type payload-json [message id]
  printf '{"messageId":"%s","correlationId":"%s","type":"%s","version":1,"occurredAt":"%s","payload":%s}' \
    "${3:-$(new_uuid)}" "$(new_uuid)" "$1" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$2"
}

wait_for() { # description seconds command...
  local description="$1" seconds="$2"; shift 2
  local waited=0
  until "$@" > /dev/null 2>&1; do
    sleep 1; waited=$((waited + 1))
    if [ "$waited" -ge "$seconds" ]; then echo "    timeout waiting for: $description"; return 1; fi
  done
  echo "    ok after ${waited}s: $description"
}

subscription_status() { api GET "/subscriptions/$2" "$1" | json_field status; }

payment_healthy_count() { docker ps --filter name=futhub-payment- --filter health=healthy -q | wc -l | tr -d ' '; }

wait_payment() { # replicas
  until [ "$(payment_healthy_count)" = "$1" ]; do
    sleep 1; waited_payment=$(( ${waited_payment:-0} + 1 ))
    if [ "$waited_payment" -ge 120 ]; then echo "    timeout waiting for $1 healthy payment replica(s)"; waited_payment=0; return 1; fi
  done
  waited_payment=0
  echo "    ok: $1 payment replica(s) healthy"
}

restore_defaults() {
  step "Restoring the default configuration of main and payment"
  compose up -d --no-deps --scale payment=2 main payment > /dev/null 2>&1
  wait_for "main healthy" 90 curl -sf "$MAIN_URL/health"
  wait_payment 2
}

result() { # label expected actual
  if [ "$2" = "$3" ]; then printf '  PASS  %s (%s)\n' "$1" "$3"; else printf '  FAIL  %s (expected %s, got %s)\n' "$1" "$2" "$3"; FAILED=1; fi
}
FAILED=0
finish() { if [ "$FAILED" = 0 ]; then printf '\nRESULT: PASS\n'; else printf '\nRESULT: FAIL\n'; exit 1; fi; }
