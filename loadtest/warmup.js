// Warmup: 90 секунд лёгкой нагрузки, чтобы прогреть пулы, кэши и Go runtime
// до основных замеров. Цифры из warmup в отчёт не берём.
import http from "k6/http";
import { sleep } from "k6";
import { ADS_BASE, BILLING_BASE, randomSeedAdId, randomTitle, JSON_HEADERS } from "./lib/config.js";

export const options = {
  scenarios: {
    warmup_mix: {
      executor: "constant-arrival-rate",
      rate: 20,
      timeUnit: "1s",
      duration: "90s",
      preAllocatedVUs: 30,
      maxVUs: 80,
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.05"],
  },
};

export default function () {
  const r = Math.random();
  if (r < 0.8) {
    http.get(`${ADS_BASE}/api/v1/ads/${randomSeedAdId()}`);
  } else if (r < 0.95) {
    const cr = http.post(
      `${ADS_BASE}/api/v1/ads`,
      JSON.stringify({ title: randomTitle("warmup") }),
      { headers: JSON_HEADERS },
    );
    if (cr.status === 201) {
      const id = JSON.parse(cr.body).id;
      http.post(`${ADS_BASE}/api/v1/ads/${id}/publish`, null);
    }
  } else {
    http.post(`${BILLING_BASE}/api/v1/ads/${randomSeedAdId()}/promotions`, null);
  }
  sleep(0.05);
}
