// BASE_URL, BILLING_URL — хост VM (k6 с ноутбука)
import http from "k6/http";
import { check, sleep } from "k6";

const adsBase = __ENV.BASE_URL || "http://127.0.0.1:8081";
const billingBase = __ENV.BILLING_URL || "http://127.0.0.1:8082";
const description = "k6 smoke " + "x".repeat(500);

export const options = {
  vus: 3,
  duration: "45s",
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: ["p(99)<2000"],
  },
};

export default function () {
  const create = http.post(
    `${adsBase}/api/v1/ads`,
    JSON.stringify({
      title: "k6 listing item",
      description: description,
      category: "electronics",
      region: "spb",
      price: 42000,
    }),
    { headers: { "Content-Type": "application/json" } },
  );
  check(create, { "create 201": (r) => r.status === 201 });
  if (create.status !== 201) {
    sleep(1);
    return;
  }
  const id = JSON.parse(create.body).id;

  const publish = http.post(`${adsBase}/api/v1/ads/${id}/publish`, null);
  check(publish, { "publish 200": (r) => r.status === 200 });

  const get = http.get(`${adsBase}/api/v1/ads/${id}`);
  check(get, { "get 200": (r) => r.status === 200 });

  const promo = http.post(`${billingBase}/api/v1/ads/${id}/promotions`, null);
  check(promo, { "promotion 201": (r) => r.status === 201 });

  sleep(0.3);
}
