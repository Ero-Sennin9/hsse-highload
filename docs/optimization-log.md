# Optimization log

Шаблон по [ДЗ 3](https://github.com/RobertMaz/MIPT-Highload-2026/blob/main/homework/Assignment%2003%20-%20PoC.md).

Профиль трафика: **read-heavy** (~80% GET / ~15% write / ~5% promotion).
VM: 2 vCPU, 8 GB RAM, 72 GB HDD.

---

## Таблица прогресса

| Метрика         | NFR (ДЗ1)           | Iter 0 | Iter 1 | Iter 2 | Iter 3 |
|-----------------|---------------------|--------|--------|--------|--------|
| Latency p99 read  | < 500 мс          |        |        |        |        |
| Latency p99 write | < 1000 мс         |        |        |        |        |
| Max RPS (read)  | ≥ 100               |        |        |        |        |
| Max RPS (write) | ≥ 10                |        |        |        |        |
| Error rate      | < 1% (5 мин mix)    |        |        |        |        |
| 2x spike errors | < 5%                |        |        |        |        |
| CPU / RAM       | CPU 70–90% ИЛИ RAM 70–90% |   |        |        |        |
| Bottleneck      | —                   |        |        |        |        |
| Что сделали     | —                   | —      |        |        |        |
| NFR достигнут?  | —                   |        |        |        |        |

---

## Iteration 0 — Baseline

### RED метрики

#### 2.1. Read stress

| Ступень (RPS) | p50 мс | p95 мс | p99 мс | Error rate |
|--------------|--------|--------|--------|------------|
| 50           |        |        |        |            |
| 100          |        |        |        |            |
| 150          |        |        |        |            |
| 200          |        |        |        |            |
| 300          |        |        |        |            |
| 400          |        |        |        |            |

**Max RPS при p99 < 500 мс и error < 1%:** _TBD_

#### 2.2. Write stress

| Ступень (RPS) | p50 мс | p95 мс | p99 мс | Error rate |
|--------------|--------|--------|--------|------------|
| 10           |        |        |        |            |
| 30           |        |        |        |            |
| 50           |        |        |        |            |
| 80           |        |        |        |            |
| 120          |        |        |        |            |

**Max RPS при p99 < 1000 мс и error < 1%:** _TBD_

#### 2.3. Mixed sustained (5 мин)

| Endpoint    | p50 мс | p95 мс | p99 мс | Error rate |
|-------------|--------|--------|--------|------------|
| read (GET)  |        |        |        |            |
| create      |        |        |        |            |
| publish     |        |        |        |            |
| promotion   |        |        |        |            |

**Общий error rate за 5 мин:** _TBD_ (NFR: < 1%)

#### 2.4. Stress mixed — поиск потолка

| Total RPS | Read p99 мс | Write p99 мс | Error rate |
|-----------|-------------|--------------|------------|
| 50        |             |              |            |
| 100       |             |              |            |
| 200       |             |              |            |
| 300       |             |              |            |
| 450       |             |              |            |
| 600       |             |              |            |

**BASE_RPS (последняя устойчивая ступень):** _TBD_

#### 2.5. 2x Spike

| Фаза            | Total RPS | Error rate | p99 мс |
|-----------------|-----------|------------|--------|
| Pre-spike       |           |            |        |
| Spike (2×base)  |           |            |        |
| Recovery        |           |            |        |

**OOMKilled контейнеров:** _TBD_
**Время восстановления:** _TBD_ (NFR: < 1 мин)

---

### USE метрики (на пиковой ступени stress)

| Ресурс           | Utilization (avg/peak) | Saturation | Errors |
|------------------|----------------------|------------|--------|
| CPU ads-service  |                      | —          | —      |
| CPU billing-svc  |                      | —          | —      |
| CPU postgres     |                      | —          | —      |
| RAM ads-service  |                      | —          | —      |
| RAM billing-svc  |                      | —          | —      |
| RAM postgres     |                      | —          | —      |
| Disk I/O (HDD)   |                      | %util=     | —      |
| PG connections   |                      | queue=     | refused= |

---

### Bottleneck (iter-0)

_TBD — после замеров заполнить 1–3 абзаца по шаблону:_

Что стало узким местом и на основании каких данных (cpu% из docker-stats, await из iostat, queue из pg-connections и т.д.):

- ...

### Gap vs NFR

| NFR                        | Цель     | Факт    | Gap   |
|----------------------------|----------|---------|-------|
| Read p99                   | < 500 мс |         |       |
| Write p99                  | < 1000 мс|         |       |
| Max read RPS               | ≥ 100    |         |       |
| Max write RPS              | ≥ 10     |         |       |
| Error rate (5 мин)         | < 1%     |         |       |
| Spike error rate           | < 5%     |         |       |

---

## Iteration 1

_TBD_

### Что изменили

- ...

### Результаты

| Метрика    | До (iter-0) | После (iter-1) | Delta |
|------------|-------------|----------------|-------|
| Read p99   |             |                |       |
| Max RPS    |             |                |       |
| Error rate |             |                |       |
| CPU        |             |                |       |

---

## Iteration 2

_TBD_

### Что изменили

- ...

### Результаты

| Метрика    | До (iter-1) | После (iter-2) | Delta |
|------------|-------------|----------------|-------|
| Read p99   |             |                |       |
| Max RPS    |             |                |       |
| Error rate |             |                |       |
| CPU        |             |                |       |

---

## Iteration 3

_TBD_

### Что изменили

- ...

### Результаты

| Метрика    | До (iter-2) | После (iter-3) | Delta |
|------------|-------------|----------------|-------|
| Read p99   |             |                |       |
| Max RPS    |             |                |       |
| Error rate |             |                |       |
| CPU        |             |                |       |
