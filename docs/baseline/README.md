# Iteration 0 — Baseline runbook

Точная последовательность для снятия baseline на VM.
Итог — заполненная таблица RED+USE и анализ bottleneck в
[`docs/optimization-log.md`](../optimization-log.md) + git-тег `iter-0`.

Профиль трафика: **read-heavy** (~80% GET / ~15% write / ~5% promotion).

Целевые NFR (минимальная планка ДЗ 3):

| Класс операций | RPS    | p99      | Error rate |
|----------------|--------|----------|------------|
| Read           | ≥ 100  | < 500 мс | < 1%       |
| Write          | ≥ 10*  | < 1 с    | < 1%       |
| 2x spike       | —      | —        | < 5%       |

\* Write-планка снижена с 30 до 10 RPS на основании read-heavy профиля (разрешено ДЗ).

VM: 2 vCPU, 8 GB RAM, 72 GB HDD.

---

## 0. Pre-flight (один раз)

**На VM:**

```bash
git pull
docker compose down -v          # чистая БД
docker compose up -d --build
docker compose ps               # дождаться "healthy" у postgres / ads / billing
make seed                       # засеять 50 000 published ads
make seed-count                 # должно вернуть 50000
```

**На ноутбуке:**

```bash
# Установить k6 (macOS)
brew install k6
# Установить k6 (Ubuntu)
# см. README.md

export BASE_URL=http://<IP_VM>:8081
export BILLING_URL=http://<IP_VM>:8082

# smoke — проверить, что приложение и сеть до VM живые
k6 run --duration 20s --vus 5 loadtest/smoke.js
```

Если smoke зелёный — переходим к замерам.

---

## 1. Прогрев

> Цель: разогреть Go runtime / connection pool / PG shared buffers.

```bash
k6 run loadtest/warmup.js
```

90 секунд лёгкой нагрузки (~20 RPS микса). Цифры из warmup **не используем**.

---

## 2. Замеры baseline (последовательно)

Каждый прогон:
1. На VM в отдельной SSH-сессии запустить `collect-use-metrics.sh` **первым**.
2. Параллельно на ноутбуке запустить нужный сценарий k6.
3. По истечении k6 коллектор сам остановится по таймауту.

`<duration_sec>` должно быть ≥ длительности k6-сценария + 60 с запаса.

### 2.1. Read stress — потолок чтения (~12.5 мин)

| | |
|---|---|
| Ступени | 50 → 100 → 150 → 200 → 300 → 400 RPS, по 2 мин + 30 с rampdown |
| VM | `./scripts/collect-use-metrics.sh read 900` |
| Ноутбук | `k6 run --summary-export=docs/baseline/raw/read/k6-summary.json loadtest/read.js` |

Из отчёта берём: **max RPS при p99 < 500 мс и errors < 1%**, а также p50/p95/p99.

### 2.2. Write stress — потолок записи (~10.5 мин)

| | |
|---|---|
| Ступени | 10 → 30 → 50 → 80 → 120 RPS, по 2 мин |
| VM | `./scripts/collect-use-metrics.sh write 750` |
| Ноутбук | `k6 run --summary-export=docs/baseline/raw/write/k6-summary.json loadtest/write.js` |

> Каждая итерация = 2 HTTP-запроса (create + publish). Реальный RPS к API = ×2.

### 2.3. Mixed sustained — 5 мин на целевом RPS

| | |
|---|---|
| Параметры | `READ_RPS=100 WRITE_RPS=10 PROMO_RPS=5 DURATION=5m` |
| VM | `./scripts/collect-use-metrics.sh mixed 360` |
| Ноутбук | `READ_RPS=100 WRITE_RPS=10 PROMO_RPS=5 k6 run --summary-export=docs/baseline/raw/mixed/k6-summary.json loadtest/mixed.js` |

Берём строку **"Error rate < 1% при устойчивой нагрузке 5 мин"** из NFR.
Если на 100/10/5 деградация — снизить `READ_RPS` до потолка из 2.1 и зафиксировать gap.

### 2.4. Stress mixed — поиск общего потолка (~12.5 мин)

| | |
|---|---|
| Ступени | 50 → 100 → 200 → 300 → 450 → 600 RPS суммарно, по 2 мин |
| VM | `./scripts/collect-use-metrics.sh stress 900` |
| Ноутбук | `k6 run --summary-export=docs/baseline/raw/stress/k6-summary.json loadtest/stress.js` |

Берём **последнюю ступень до деградации** — это `BASE_RPS` для spike.

### 2.5. 2x Spike (~5.5 мин)

Подставить `<X>` = max sustained RPS из 2.4.

| | |
|---|---|
| Стадии | base 2 мин → 2×base 1 мин → base 2 мин (recovery) |
| VM | `./scripts/collect-use-metrics.sh spike 360` |
| Ноутбук | `BASE_RPS=<X> k6 run --summary-export=docs/baseline/raw/spike/k6-summary.json loadtest/spike.js` |

Проверяем: error rate < 5%, нет OOMKilled, восстановление за ≤ 1 мин.

---

## 3. Забрать сырые данные с VM на ноутбук

```bash
mkdir -p docs/baseline/raw
rsync -av student@<IP_VM>:~/hsse-highload/docs/baseline/raw/ docs/baseline/raw/
```

Итоговая структура:

```
docs/baseline/raw/
├── read/
│   ├── k6-summary.json
│   └── <TS>/{meta.txt,docker-stats.log,vmstat.log,iostat.log,free.log,pg-connections.log,...}
├── write/...
├── mixed/...
├── stress/...
└── spike/...
```

---

## 4. Заполнить отчёт

Полезные команды для вытаскивания цифр из k6-summary:

```bash
# Общая сводка (RPS, error rate, p50/p95/p99)
jq '{
  rps_avg:      .metrics.http_reqs.rate,
  failed_rate:  .metrics.http_req_failed.value,
  p50_ms:       .metrics.http_req_duration.med,
  p95_ms:       .metrics.http_req_duration["p(95)"],
  p99_ms:       .metrics.http_req_duration["p(99)"],
  max_ms:       .metrics.http_req_duration.max
}' docs/baseline/raw/read/k6-summary.json

# Per-endpoint (по тегу op=read/create/publish/promotion)
jq '.metrics | with_entries(select(.key|test("http_req_duration\\{op:")))' \
   docs/baseline/raw/mixed/k6-summary.json
```

Для USE-метрик:

```bash
# CPU по контейнерам
grep -E "ads-service|billing-service|marketplace-postgres" \
  docs/baseline/raw/stress/<TS>/docker-stats.log | less

# Disk saturation (HDD!)
less docs/baseline/raw/stress/<TS>/iostat.log

# CPU всей VM
less docs/baseline/raw/stress/<TS>/vmstat.log

# PG-соединения
less docs/baseline/raw/stress/<TS>/pg-connections.log

# OOM / crash
tail -n 30 docs/baseline/raw/stress/<TS>/meta.txt
```

---

## 5. Где смотреть какие метрики

| Метрика (RED/USE)              | Источник                                                        |
|-------------------------------|----------------------------------------------------------------|
| **R**ate (RPS)                | k6 summary, `http_reqs.rate`                                   |
| **E**rrors (%)                | k6 summary, `http_req_failed.value`                            |
| **D**uration p50/p95/p99      | k6 summary, `http_req_duration`                                |
| **U**tilization CPU/RAM (cont.)| `docker-stats.log`                                             |
| **U**tilization CPU/RAM (VM)  | `vmstat.log` (us/sy/id), `free.log`                            |
| **S**aturation disk I/O       | `iostat.log` (`%util`, `await`), `vmstat` (`bi`/`bo`)          |
| **S**aturation PG connections | `pg-connections.log`                                           |
| **E**rrors (OOM / crash)      | `meta.txt` финальный блок (`OOMKilled`, `ExitCode`)            |
| **E**rrors (unreachable)      | `ads-health.log` / `billing-health.log`                        |

---

## 6. Зафиксировать в git

```bash
git add docs/baseline docs/optimization-log.md
git commit -m "iter-0: baseline measurements (read/write/mixed/stress/spike)"
git tag iter-0
git push && git push --tags
```
