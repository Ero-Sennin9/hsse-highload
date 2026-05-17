// Чистый write-стресс: POST /api/v1/ads + POST /api/v1/ads/{id}/publish.
// Каждая итерация = одно "пользовательское" создание объявления (2 запроса).
// Ступени по 2 минуты. read-heavy профиль => целевая планка write = 10 RPS
// (создаваний/сек), но ищем потолок для отчёта.
import http from "k6/http";
import { check } from "k6";
import { ADS_BASE, randomCreatePayload, JSON_HEADERS } from "./lib/config.js";

export const options = {
  scenarios: {
    write_stress: {
      executor: "ramping-arrival-rate",
      startRate: 10,
      timeUnit: "1s",
      preAllocatedVUs: 30,
      maxVUs: 200,
      stages: [
        { target: 10,  duration: "2m" },
        { target: 30,  duration: "2m" },
        { target: 50,  duration: "2m" },
        { target: 80,  duration: "2m" },
        { target: 120, duration: "2m" },
        { target: 0,   duration: "30s" },
      ],
    },
  },
  thresholds: {
    "http_req_failed": ["rate<0.01"],
    "http_req_duration{expected_response:true}": ["p(99)<1000"],
  },
};

export default function () {
  const cr = http.post(
    `${ADS_BASE}/api/v1/ads`,
    JSON.stringify(randomCreatePayload("k6-w")),
    { headers: JSON_HEADERS, tags: { op: "create" } },
  );
  if (!check(cr, { "create 201": (r) => r.status === 201 })) {
    return;
  }
  const id = JSON.parse(cr.body).id;
  const pr = http.post(`${ADS_BASE}/api/v1/ads/${id}/publish`, null, { tags: { op: "publish" } });
  check(pr, { "publish 200": (r) => r.status === 200 });
}
