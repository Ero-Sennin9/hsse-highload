// 2x spike: 2 мин на BASE_RPS → 1 мин на 2×BASE_RPS → 2 мин восстановление.
// BASE_RPS подставить из фактического max sustained RPS из stress.js.
// NFR: error rate < 5%, нет OOM/крашей контейнеров, восстановление < 1 мин.
//
// Запуск: BASE_RPS=200 k6 run loadtest/spike.js
import http from "k6/http";
import { check } from "k6";
import {
  ADS_BASE,
  BILLING_BASE,
  randomSeedAdId,
  randomCreatePayload,
  JSON_HEADERS,
} from "./lib/config.js";

const BASE_RPS  = parseInt(__ENV.BASE_RPS || "150", 10);
const SPIKE_RPS = BASE_RPS * 2;

const READ_FRAC  = 0.80;
const WRITE_FRAC = 0.15;
const PROMO_FRAC = 0.05;

const stages = [
  { target: BASE_RPS,  duration: "2m" },   // прогрев / pre-spike
  { target: SPIKE_RPS, duration: "1m" },   // сам спайк
  { target: BASE_RPS,  duration: "2m" },   // окно восстановления
  { target: 0,         duration: "30s" },
];

function rpsStages(frac) {
  return stages.map((s) => ({
    target: Math.max(1, Math.round(s.target * frac)),
    duration: s.duration,
  }));
}

export const options = {
  scenarios: {
    reads: {
      executor: "ramping-arrival-rate",
      exec: "doRead",
      startRate: Math.round(BASE_RPS * READ_FRAC),
      timeUnit: "1s",
      preAllocatedVUs: 100,
      maxVUs: 1000,
      stages: rpsStages(READ_FRAC),
    },
    writes: {
      executor: "ramping-arrival-rate",
      exec: "doWrite",
      startRate: Math.max(1, Math.round(BASE_RPS * WRITE_FRAC)),
      timeUnit: "1s",
      preAllocatedVUs: 40,
      maxVUs: 400,
      stages: rpsStages(WRITE_FRAC),
    },
    promotions: {
      executor: "ramping-arrival-rate",
      exec: "doPromotion",
      startRate: Math.max(1, Math.round(BASE_RPS * PROMO_FRAC)),
      timeUnit: "1s",
      preAllocatedVUs: 15,
      maxVUs: 150,
      stages: rpsStages(PROMO_FRAC),
    },
  },
  thresholds: {
    "http_req_failed": ["rate<0.05"],
  },
};

export function doRead() {
  const r = http.get(`${ADS_BASE}/api/v1/ads/${randomSeedAdId()}`, { tags: { op: "read" } });
  check(r, { "get 200": (x) => x.status === 200 });
}

export function doWrite() {
  const cr = http.post(
    `${ADS_BASE}/api/v1/ads`,
    JSON.stringify(randomCreatePayload("k6-sp")),
    { headers: JSON_HEADERS, tags: { op: "create" } },
  );
  if (!check(cr, { "create 201": (x) => x.status === 201 })) return;
  const id = JSON.parse(cr.body).id;
  const pr = http.post(`${ADS_BASE}/api/v1/ads/${id}/publish`, null, { tags: { op: "publish" } });
  check(pr, { "publish 200": (x) => x.status === 200 });
}

export function doPromotion() {
  const r = http.post(`${BILLING_BASE}/api/v1/ads/${randomSeedAdId()}/promotions`, null, {
    tags: { op: "promotion" },
  });
  check(r, { "promotion 201": (x) => x.status === 201 });
}
