// Общие утилиты для всех k6-сценариев baseline (Iteration 0).
// BASE_URL / BILLING_URL — переопределяются переменными окружения.
// SEED_N — сколько объявлений посеяно в БД через deployments/postgres/seed.sql.

export const ADS_BASE = __ENV.BASE_URL || "http://127.0.0.1:8081";
export const BILLING_BASE = __ENV.BILLING_URL || "http://127.0.0.1:8082";
export const SEED_N = parseInt(__ENV.SEED_N || "50000", 10);

const HEX = "0123456789abcdef";

// UUID вида '00000000-0000-0000-0000-XXXXXXXXXXXX' по индексу i (как в seed.sql).
export function seedAdId(i) {
  let hex = i.toString(16);
  while (hex.length < 12) hex = "0" + hex;
  return "00000000-0000-0000-0000-" + hex;
}

export function randomSeedAdId() {
  const i = 1 + Math.floor(Math.random() * SEED_N);
  return seedAdId(i);
}

export function randomTitle(prefix) {
  let s = (prefix || "k6") + "-";
  for (let i = 0; i < 8; i++) {
    s += HEX[Math.floor(Math.random() * 16)];
  }
  return s;
}

const CATEGORIES = ["transport", "electronics", "realty", "clothing", "other"];
const REGIONS    = ["moscow", "spb", "other"];

export function randomCreatePayload(prefix) {
  const title = randomTitle(prefix);
  return {
    title:       title + " объявление",
    description: "Тестовое объявление для нагрузочного теста k6, prefix=" + title + ", данные синтетические.",
    category:    CATEGORIES[Math.floor(Math.random() * CATEGORIES.length)],
    region:      REGIONS[Math.floor(Math.random() * REGIONS.length)],
    price:       Math.floor(Math.random() * 100000),
  };
}

export const JSON_HEADERS = { "Content-Type": "application/json" };
