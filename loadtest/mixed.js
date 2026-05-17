// Mixed read-heavy профиль (80% read / 15% write / 5% promotion).
// Основной сценарий "длительной устойчивой нагрузки на целевом RPS".
// Цель: 100 read RPS + 10 write RPS + 5 promotion RPS в течение 5 минут.
// Именно этот прогон закрывает строку "Error rate < 1% при устойчивой нагрузке" из NFR.
import http from "k6/http";
import { check } from "k6";
import {
  ADS_BASE,
  BILLING_BASE,
  randomSeedAdId,
  randomCreatePayload,
  JSON_HEADERS,
} from "./lib/config.js";

const READ_RPS  = parseInt(__ENV.READ_RPS  || "100", 10);
const WRITE_RPS = parseInt(__ENV.WRITE_RPS || "10",  10);
const PROMO_RPS = parseInt(__ENV.PROMO_RPS || "5",   10);
const DURATION  = __ENV.DURATION || "5m";

export const options = {
  scenarios: {
    reads: {
      executor: "constant-arrival-rate",
      exec: "doRead",
      rate: READ_RPS,
      timeUnit: "1s",
      duration: DURATION,
      preAllocatedVUs: Math.max(50, READ_RPS),
      maxVUs: Math.max(200, READ_RPS * 4),
    },
    writes: {
      executor: "constant-arrival-rate",
      exec: "doWrite",
      rate: WRITE_RPS,
      timeUnit: "1s",
      duration: DURATION,
      preAllocatedVUs: Math.max(20, WRITE_RPS * 2),
      maxVUs: Math.max(60, WRITE_RPS * 6),
    },
    promotions: {
      executor: "constant-arrival-rate",
      exec: "doPromotion",
      rate: PROMO_RPS,
      timeUnit: "1s",
      duration: DURATION,
      preAllocatedVUs: Math.max(10, PROMO_RPS * 2),
      maxVUs: Math.max(30, PROMO_RPS * 6),
    },
  },
  thresholds: {
    "http_req_failed": ["rate<0.01"],
    "http_req_duration{op:read}":      ["p(99)<500"],
    "http_req_duration{op:create}":    ["p(99)<1000"],
    "http_req_duration{op:publish}":   ["p(99)<1000"],
    "http_req_duration{op:promotion}": ["p(99)<1000"],
  },
};

export function doRead() {
  const r = http.get(`${ADS_BASE}/api/v1/ads/${randomSeedAdId()}`, { tags: { op: "read" } });
  check(r, { "get 200": (x) => x.status === 200 });
}

export function doWrite() {
  const cr = http.post(
    `${ADS_BASE}/api/v1/ads`,
    JSON.stringify(randomCreatePayload("k6-m")),
    { headers: JSON_HEADERS, tags: { op: "create" } },
  );
  if (!check(cr, { "create 201": (x) => x.status === 201 })) return;
  const id = JSON.parse(cr.body).id;
  const pr = http.post(`${ADS_BASE}/api/v1/ads/${id}/publish`, null, { tags: { op: "publish" } });
  check(pr, { "publish 200": (x) => x.status === 200 });
}

export function doPromotion() {
  // Используем посеянный published id, чтобы billing не валился на "not published".
  const id = randomSeedAdId();
  const r = http.post(`${BILLING_BASE}/api/v1/ads/${id}/promotions`, null, {
    tags: { op: "promotion" },
  });
  check(r, { "promotion 201": (x) => x.status === 201 });
}
