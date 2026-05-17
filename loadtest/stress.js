// Stress в read-heavy профиле: ступенчато поднимаем суммарный RPS до отказа,
// сохраняя соотношение ~80% read / 15% write / 5% promotion.
// Ступени по 2 минуты (RED+USE фиксируются на каждой ступени).
//
// Базовый уровень = 50 RPS суммарно, потолок = 600 RPS суммарно.
// Из этого прогона берём BASE_RPS для spike.js.
import http from "k6/http";
import { check } from "k6";
import {
  ADS_BASE,
  BILLING_BASE,
  randomSeedAdId,
  randomCreatePayload,
  JSON_HEADERS,
} from "./lib/config.js";

const READ_FRAC  = 0.80;
const WRITE_FRAC = 0.15;
const PROMO_FRAC = 0.05;

const stages = [
  { target: 100,  duration: "2m" },
  { target: 300,  duration: "2m" },
  { target: 600,  duration: "2m" },
  { target: 900,  duration: "2m" },
  { target: 1200, duration: "2m" },
  { target: 1500, duration: "2m" },
  { target: 0,    duration: "30s" },
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
      startRate: Math.round(100 * READ_FRAC),
      timeUnit: "1s",
      preAllocatedVUs: 200,
      maxVUs: 2000,
      stages: rpsStages(READ_FRAC),
    },
    writes: {
      executor: "ramping-arrival-rate",
      exec: "doWrite",
      startRate: Math.max(1, Math.round(100 * WRITE_FRAC)),
      timeUnit: "1s",
      preAllocatedVUs: 50,
      maxVUs: 800,
      stages: rpsStages(WRITE_FRAC),
    },
    promotions: {
      executor: "ramping-arrival-rate",
      exec: "doPromotion",
      startRate: Math.max(1, Math.round(100 * PROMO_FRAC)),
      timeUnit: "1s",
      preAllocatedVUs: 20,
      maxVUs: 200,
      stages: rpsStages(PROMO_FRAC),
    },
  },
  thresholds: {
    // Мягкий порог для stress — сам k6 не падает, анализируем вручную.
    "http_req_failed": ["rate<0.50"],
  },
};

export function doRead() {
  const r = http.get(`${ADS_BASE}/api/v1/ads/${randomSeedAdId()}`, { tags: { op: "read" } });
  check(r, { "get 200": (x) => x.status === 200 });
}

export function doWrite() {
  const cr = http.post(
    `${ADS_BASE}/api/v1/ads`,
    JSON.stringify(randomCreatePayload("k6-s")),
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
