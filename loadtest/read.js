// Чистый read-стресс: GET /api/v1/ads/{id} по случайным посеянным id.
// Ступени по 2 минуты, ramping-arrival-rate.
// Цель — найти max RPS, при котором p99 < 500 мс и error rate < 1% (NFR read).
import http from "k6/http";
import { check } from "k6";
import { ADS_BASE, randomSeedAdId } from "./lib/config.js";

export const options = {
  discardResponseBodies: true,
  scenarios: {
    read_stress: {
      executor: "ramping-arrival-rate",
      startRate: 50,
      timeUnit: "1s",
      preAllocatedVUs: 50,
      maxVUs: 400,
      stages: [
        { target: 50,  duration: "2m" },
        { target: 100, duration: "2m" },
        { target: 150, duration: "2m" },
        { target: 200, duration: "2m" },
        { target: 300, duration: "2m" },
        { target: 400, duration: "2m" },
        { target: 0,   duration: "30s" },
      ],
    },
  },
  thresholds: {
    "http_req_failed": ["rate<0.01"],
    "http_req_duration{expected_response:true}": ["p(99)<500"],
  },
};

export default function () {
  const res = http.get(`${ADS_BASE}/api/v1/ads/${randomSeedAdId()}`);
  check(res, { "get 200": (r) => r.status === 200 });
}
