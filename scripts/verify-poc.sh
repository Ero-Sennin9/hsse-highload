#!/usr/bin/env bash
# PoC verify: health → create (full body) → photo → publish → get → promotion.
#   ./scripts/verify-poc.sh
#   START_COMPOSE=1 ./scripts/verify-poc.sh
#   SAVE_OUTPUT=1 ./scripts/verify-poc.sh   # пишет scripts/verify-poc.output.txt
set -euo pipefail

ADS_URL="${ADS_URL:-http://127.0.0.1:8081}"
BILLING_URL="${BILLING_URL:-http://127.0.0.1:8082}"
MINIO_URL="${MINIO_URL:-http://127.0.0.1:9000}"
HEALTH_RETRIES="${HEALTH_RETRIES:-90}"
HEALTH_INTERVAL="${HEALTH_INTERVAL:-2}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ "${SAVE_OUTPUT:-0}" == "1" ]]; then
  exec > >(tee "${ROOT}/scripts/verify-poc.output.txt") 2>&1
fi

fail() { echo "FAIL: $*" >&2; exit 1; }
ok()   { echo "    OK: $*"; }

json_get() {
  local json="$1" expr="$2"
  if command -v jq >/dev/null 2>&1; then
    echo "${json}" | jq -r "${expr}"
    return
  fi
  echo "${json}" | python3 -c '
import json, sys
d = json.load(sys.stdin)
e = sys.argv[1]
if e == ".id": print(d.get("id", ""))
elif e == ".status": print(d.get("status", ""))
elif e == ".title": print(d.get("title", ""))
elif e == ".category": print(d.get("category", ""))
elif e == ".region": print(d.get("region", ""))
elif e == ".price": print(d.get("price", ""))
elif e == ".url": print(d.get("url", ""))
elif e == ".position": print(d.get("position", ""))
elif e == ".contentType": print(d.get("contentType", ""))
elif e == ".adId": print(d.get("adId", ""))
elif e == ".description | length": print(len(d.get("description", "")))
elif e == ".photos | length": print(len(d.get("photos", [])))
elif e == ".description[:10]": print(d.get("description", "")[:10])
else: raise SystemExit(f"unsupported expr without jq: {e}")
' "${expr}"
}

assert_eq() {
  local name="$1" got="$2" want="$3"
  [[ "${got}" == "${want}" ]] || fail "${name}: got '${got}', want '${want}'"
}

assert_nonempty() {
  local name="$1" val="$2"
  [[ -n "${val}" && "${val}" != "null" ]] || fail "${name} is empty"
}

http_code() {
  curl -sS -o /dev/null -w "%{http_code}" "$@"
}

if [[ "${START_COMPOSE:-0}" == "1" ]]; then
  echo "==> docker compose up -d --build"
  docker compose up -d --build
  echo "    (при смене схемы: docker compose down -v && docker compose up -d --build)"
fi

wait_http() {
  local name="$1" url="$2"
  local i=0
  echo "==> wait ${name}: ${url}"
  while (( i < HEALTH_RETRIES )); do
    if curl -sf --max-time 3 "${url}" >/dev/null 2>&1; then
      ok "${name}"
      return 0
    fi
    ((i++)) || true
    sleep "${HEALTH_INTERVAL}"
  done
  fail "${name} timeout ($((HEALTH_RETRIES * HEALTH_INTERVAL))s)"
}

wait_http "ads /health" "${ADS_URL}/health"
wait_http "billing /health" "${BILLING_URL}/health"

# --- create ---
CREATE_BODY="$(python3 -c '
import json
print(json.dumps({
  "title": "Verify bike Pro",
  "description": "load-test " + "x" * 480,
  "category": "transport",
  "region": "moscow",
  "price": 19900,
}))
')"

echo "==> POST ${ADS_URL}/api/v1/ads"
CREATE_JSON="$(curl -sfS -X POST "${ADS_URL}/api/v1/ads" \
  -H 'Content-Type: application/json' \
  -d "${CREATE_BODY}")"
echo "${CREATE_JSON}" | (command -v jq >/dev/null && jq . || cat)

AD_ID="$(json_get "${CREATE_JSON}" '.id')"
assert_nonempty "ad id" "${AD_ID}"
assert_eq "create status" "$(json_get "${CREATE_JSON}" '.status')" "moderation_pending"
assert_eq "create title" "$(json_get "${CREATE_JSON}" '.title')" "Verify bike Pro"
assert_eq "create category" "$(json_get "${CREATE_JSON}" '.category')" "transport"
assert_eq "create region" "$(json_get "${CREATE_JSON}" '.region')" "moscow"
assert_eq "create price" "$(json_get "${CREATE_JSON}" '.price')" "19900"
DESC_LEN="$(json_get "${CREATE_JSON}" '.description | length')"
[[ "${DESC_LEN}" -ge 20 ]] || fail "description too short (${DESC_LEN})"
ok "create ad_id=${AD_ID}"

# --- promotion before publish (expect conflict) ---
echo "==> POST promotion before publish (expect 409)"
CODE="$(http_code -X POST "${BILLING_URL}/api/v1/ads/${AD_ID}/promotions")"
[[ "${CODE}" == "409" || "${CODE}" == "404" ]] || fail "promotion before publish: HTTP ${CODE}, want 409 or 404"
ok "promotion blocked before publish (HTTP ${CODE})"

# --- photo upload ---
PHOTO_FILE="$(mktemp "${TMPDIR:-/tmp}/verify-poc-XXXXXX.png")"
trap 'rm -f "${PHOTO_FILE}"' EXIT

python3 -c "
import struct, zlib, sys
path = sys.argv[1]
def chunk(t, d):
    return struct.pack('>I', len(d)) + t + d + struct.pack('>I', zlib.crc32(t + d) & 0xffffffff)
raw = b'\x00' * 15 + b'\xff' * 15
img = chunk(b'IHDR', struct.pack('>IIBBBBB', 1, 1, 8, 2, 0, 0, 0))
img += chunk(b'IDAT', zlib.compress(raw)) + chunk(b'IEND', b'')
open(path, 'wb').write(b'\x89PNG\r\n\x1a\n' + img)
" "${PHOTO_FILE}"

echo "==> POST ${ADS_URL}/api/v1/ads/${AD_ID}/photos"
PHOTO_JSON="$(curl -sfS -X POST "${ADS_URL}/api/v1/ads/${AD_ID}/photos" \
  -F "file=@${PHOTO_FILE};type=image/png")"
echo "${PHOTO_JSON}" | (command -v jq >/dev/null && jq . || cat)

PHOTO_URL="$(json_get "${PHOTO_JSON}" '.url')"
assert_nonempty "photo url" "${PHOTO_URL}"
assert_eq "photo position" "$(json_get "${PHOTO_JSON}" '.position')" "0"
assert_eq "photo contentType" "$(json_get "${PHOTO_JSON}" '.contentType')" "image/png"
ok "photo uploaded"

if [[ "${CHECK_MINIO:-1}" == "1" ]]; then
  echo "==> GET photo from MinIO (${PHOTO_URL})"
  CODE="$(http_code "${PHOTO_URL}")"
  [[ "${CODE}" == "200" ]] || fail "MinIO photo URL HTTP ${CODE} (set CHECK_MINIO=0 to skip)"
  ok "photo reachable via MinIO"
fi

# --- publish ---
echo "==> POST ${ADS_URL}/api/v1/ads/${AD_ID}/publish"
PUBLISH_JSON="$(curl -sfS -X POST "${ADS_URL}/api/v1/ads/${AD_ID}/publish")"
echo "${PUBLISH_JSON}" | (command -v jq >/dev/null && jq . || cat)
assert_eq "publish status" "$(json_get "${PUBLISH_JSON}" '.status')" "published"
ok "published"

# --- get full card ---
echo "==> GET ${ADS_URL}/api/v1/ads/${AD_ID}"
GET_JSON="$(curl -sfS "${ADS_URL}/api/v1/ads/${AD_ID}")"
echo "${GET_JSON}" | (command -v jq >/dev/null && jq . || cat)

assert_eq "get status" "$(json_get "${GET_JSON}" '.status')" "published"
assert_eq "get description prefix" "$(json_get "${GET_JSON}" '.description[:10]')" "load-test "
PHOTOS_COUNT="$(json_get "${GET_JSON}" '.photos | length')"
[[ "${PHOTOS_COUNT}" -ge 1 ]] || fail "expected at least 1 photo in GET, got ${PHOTOS_COUNT}"
ok "GET card with ${PHOTOS_COUNT} photo(s)"

# --- promotion after publish ---
echo "==> POST ${BILLING_URL}/api/v1/ads/${AD_ID}/promotions"
PROMO_JSON="$(curl -sfS -X POST "${BILLING_URL}/api/v1/ads/${AD_ID}/promotions")"
echo "${PROMO_JSON}" | (command -v jq >/dev/null && jq . || cat)

PROMO_STATUS="$(json_get "${PROMO_JSON}" '.status')"
assert_nonempty "promotion status" "${PROMO_STATUS}"
assert_eq "promotion adId" "$(json_get "${PROMO_JSON}" '.adId')" "${AD_ID}"
ok "promotion status=${PROMO_STATUS}"

echo ""
echo "=== PASS: health → create → photo → publish → get → promotion ==="
