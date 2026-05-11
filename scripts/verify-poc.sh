#!/usr/bin/env bash
# PoC: health → create → publish → get → promotion. START_COMPOSE=1 поднимает compose.
set -euo pipefail

ADS_URL="${ADS_URL:-http://127.0.0.1:8081}"
BILLING_URL="${BILLING_URL:-http://127.0.0.1:8082}"
HEALTH_RETRIES="${HEALTH_RETRIES:-60}"
HEALTH_INTERVAL="${HEALTH_INTERVAL:-2}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ "${START_COMPOSE:-0}" == "1" ]]; then
  echo "==> docker compose up -d --build"
  docker compose up -d --build
fi

wait_http() {
  local name="$1" url="$2"
  local i=0
  echo "==> Ждём ${name}: ${url}"
  while (( i < HEALTH_RETRIES )); do
    if curl -sf --max-time 3 "${url}" >/dev/null 2>&1; then
      echo "    OK (${name})"
      return 0
    fi
    ((i++)) || true
    sleep "${HEALTH_INTERVAL}"
  done
  echo "    TIMEOUT: ${name} не ответил за $((HEALTH_RETRIES * HEALTH_INTERVAL)) с" >&2
  return 1
}

wait_http "ads health" "${ADS_URL}/health"
wait_http "billing health" "${BILLING_URL}/health"

echo "==> POST ${ADS_URL}/api/v1/ads"
CREATE_JSON="$(curl -sfS -X POST "${ADS_URL}/api/v1/ads" \
  -H 'Content-Type: application/json' \
  -d '{"title":"verify-script"}')"
echo "${CREATE_JSON}" | (command -v jq >/dev/null && jq . || cat)

if command -v jq >/dev/null 2>&1; then
  AD_ID="$(echo "${CREATE_JSON}" | jq -r '.id')"
else
  AD_ID="$(echo "${CREATE_JSON}" | python3 -c 'import sys,json; print(json.load(sys.stdin)["id"])')"
fi
if [[ -z "${AD_ID}" || "${AD_ID}" == "null" ]]; then
  echo "Не удалось извлечь id объявления" >&2
  exit 1
fi
echo "    ad_id=${AD_ID}"

echo "==> POST publish"
PUBLISH_JSON="$(curl -sfS -X POST "${ADS_URL}/api/v1/ads/${AD_ID}/publish")"
echo "${PUBLISH_JSON}" | (command -v jq >/dev/null && jq . || cat)

if command -v jq >/dev/null 2>&1; then
  STATUS="$(echo "${PUBLISH_JSON}" | jq -r '.status // empty')"
else
  STATUS="$(echo "${PUBLISH_JSON}" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("status",""))')"
fi
if [[ "${STATUS}" != "published" ]]; then
  echo "Ожидался status=published, получено: ${STATUS}" >&2
  exit 1
fi

echo "==> GET ad"
curl -sfS "${ADS_URL}/api/v1/ads/${AD_ID}" | (command -v jq >/dev/null && jq . || cat)

echo "==> POST promotion"
PROMO_JSON="$(curl -sfS -X POST "${BILLING_URL}/api/v1/ads/${AD_ID}/promotions")"
echo "${PROMO_JSON}" | (command -v jq >/dev/null && jq . || cat)

if command -v jq >/dev/null 2>&1; then
  PROMO_STATUS="$(echo "${PROMO_JSON}" | jq -r '.status // empty')"
else
  PROMO_STATUS="$(echo "${PROMO_JSON}" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("status",""))')"
fi
if [[ -z "${PROMO_STATUS}" ]]; then
  echo "Нет поля status в ответе promotion" >&2
  exit 1
fi

echo ""
echo "=== Всё ок: цепочка прошла (create → publish → get → promotion) ==="
