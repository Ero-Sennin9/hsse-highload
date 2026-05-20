# Deployment Document: Marketplace объявлений (Case B, Highload)

> Документ описывает целевое (production-ready) развёртывание системы. Текущий PoC (`docker-compose.yml`) — это redux-вариант одной AZ для нагрузочного тестирования; production-топология ниже расширяет PoC до полноценной отказоустойчивой схемы.

## 1. Архитектура

### 1.1. Описание развёртывания

Назначение: маркетплейс объявлений с модерацией, полнотекстовым поиском, оплатой продвижения и подписками на поисковые запросы (см. `docs/requirements.md`, `docs/architecture.md`). Целевая аудитория — 20 млн MAU (рост до 40–60 млн в горизонте 1–3 лет).

Нагрузка (производная от capacity estimation и НФТ):

- Read-heavy профиль: ~80% GET (карточки + поиск) / ~15% write (создание/публикация) / ~5% promotion.
- Целевая «полка»: **~3000 RPS** на чтение (поиск + карточки), **~400 RPS** на запись, пик ~6000 RPS на чтение (2× spike по НФТ).
- SLO (см. `docs/requirements.md`):
  - НФТ-001 поиск: p99 < 200 мс;
  - НФТ-002 платежи: 0% потерь, exactly-once;
  - НФТ-003 каталог/поиск: 99.99% uptime (≤ ~53 мин даунтайма в год);
  - НФТ-004 автомодерация: p95 < 5 с;
  - НФТ-005 media: thumbnail p99 < 150 мс, до 10k параллельных upload;
  - НФТ-007 scale-out до 2×–3× MAU без переписывания ядра.

Развёртывание: облако (managed K8s, multi-AZ), две зоны доступности **AZ-A** и **AZ-B** (active/active). Все сервисы — stateless, реплики раскатываются как Deployment’ы на K8s; stateful (PG, ES, Kafka, Redis) — managed-инстансы с синхронной/полусинхронной репликацией между AZ.

Сервисы (Deployment diagram: `deployment-diagram-skeleton.drawio` → PNG):

- **API Gateway** ×2 в каждой AZ — Nginx/Kong: TLS termination, JWT auth, rate limiting, routing. Stateless, public zone.
- **Ads Service** ×3 в каждой AZ — CRUD объявлений, FSM статусов, публикация. Stateless, private zone. Пишет в PostgreSQL (primary) + публикует события в Kafka через transactional outbox.
- **Search Service** ×3 в каждой AZ — `GET /search/ads`: фасетный полнотекст с фильтрами. Stateless, private zone. Читает только из Elasticsearch.
- **Indexer** ×2 (1 в каждой AZ) — фоновый воркер: читает `ad.*` события из Kafka, обновляет индексы Elasticsearch. Stateless, private zone.
- **Moderation Service** ×2 в каждой AZ — авто-скоринг текстов (regex / stop-words / spam-rules), публикует `ad.approved` / `ad.rejected`. Stateless, private zone.
- **Billing Service** ×2 в каждой AZ — оплата продвижения, idempotency-key, transactional outbox, активация продвижения по `ad.published`. Stateless, private zone. PostgreSQL (та же кластерная пара, отдельная схема).
- **Subscription/Notification Service** ×2 в каждой AZ — матчинг новых объявлений по сохранённым подпискам, отправка email через внешний провайдер. Stateless, private zone.
- **Media Service** ×2 в каждой AZ — приём upload’ов фото, валидация, нарезка thumbnail, запись в S3. Stateless, private zone (upload через API GW).
- **PostgreSQL** — managed кластер, **primary в AZ-A** + **sync standby в AZ-B** + 2 async read-replicas (по 1 в каждой AZ). Stateful, private zone. Для ads/payments/promotions/subscriptions (write source of truth).
- **Elasticsearch** — кластер 3 master + 6 data nodes (3 в каждой AZ), `number_of_replicas=1` на индексах. Stateful, private zone. Read-only для пользователя.
- **Kafka** — 3 брокера (по 1–2 в каждой AZ), `min.insync.replicas=2`, `acks=all` для outbox-топиков. Stateful, private zone.
- **Redis** — managed, 1 primary + 1 replica (по AZ), cache горячих карточек + token bucket для rate-limit. Stateful, private zone.
- **S3 (object storage)** — managed multi-AZ (например, S3 Standard / managed MinIO multi-site). Stateful, private zone (signed-URL раздача).
- **CDN** — внешний провайдер (CloudFront/Cloudflare), кеширует thumbnail’ы и публичные карточки. External, public zone (edge).
- **SMTP/Email provider** — external, выход через NAT GW.
- **Payment provider (эквайер)** — external, выход через NAT GW, вебхуки приходят на отдельный publicly exposed путь `api.example.com/webhooks/payments` с подписанным payload.

Адресация:

- Публичный вход: `api.example.com` → Global LB (anycast VIP) → API Gateway в обеих AZ. CDN: `cdn.example.com`.
- Service-to-service: K8s DNS, `ads.svc.cluster.local:8081`, `billing.svc.cluster.local:8082`, `search.svc.cluster.local:8083`, `media.svc.cluster.local:8084`, `moderation.svc.cluster.local:8085`, `notify.svc.cluster.local:8086`.
- Storage / infra:
  - `pg-rw.svc.cluster.local:5432` (primary), `pg-ro.svc.cluster.local:5432` (round-robin между read-replicas);
  - `es.svc.cluster.local:9200`;
  - `kafka-bootstrap.svc.cluster.local:9093` (TLS);
  - `redis.svc.cluster.local:6379`;
  - `s3.internal.example.com:443` (VPC endpoint).
- CIDR: VPC `10.0.0.0/16`. AZ-A public `10.0.1.0/24`, AZ-A private `10.0.11.0/24`. AZ-B public `10.0.2.0/24`, AZ-B private `10.0.12.0/24`.
- Static IPs указываем только там, где они реально фиксированы: NAT GW per-AZ (egress к внешним провайдерам), LB VIP, S3 VPC endpoint. Эфемерные IP подов на диаграмме не отображаем.
- Порты на стрелках: `:443` (TLS снаружи), `:5432` (PG), `:9200` (ES), `:9093` (Kafka TLS), `:6379` (Redis), `:8081–:8086` (internal HTTP).

---

### 1.2. Стратегия деплоя

Стратегия выбирается по «профилю» сервиса: насколько ему нужен zero-downtime, насколько критичен и насколько безопасно вернуть назад.

| Сервис                      | Стратегия           | Почему именно она                                                                                                                                                         |
|----------------------------|---------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| API Gateway                | **Rolling (maxSurge=1, maxUnavailable=0)** | Точка входа, нельзя терять трафик. Stateless, поддерживает graceful drain через preStop + connection draining на LB. Канари не нужно — изменения редкие и инфраструктурные. |
| Ads Service                | **Canary 5% → 25% → 100%** | Касается write-пути + статусной FSM + outbox. Любой регресс ломает публикацию и платежи. Канари 10–15 мин на ступень с автокатом по error-rate / p99. |
| Search Service             | **Canary 10% → 50% → 100%** | Critical path для НФТ-001 (p99 < 200 мс). Изменения в query/маппинге легко роняют латентность; нужна возможность быстро вернуться по метрике p99. |
| Indexer                    | **Rolling**         | Background-консьюмер Kafka; consumer-group обеспечивает плавную переподписку. Лаг — единственный риск, мониторим отдельно. |
| Moderation Service         | **Rolling**         | Асинхронный, off critical-path с т.з. пользователя; небольшой рост лага очереди допустим. |
| Billing Service            | **Blue/Green с ручным cutover** | НФТ-002 «0% потерь платежей». Двойной outbox/idempotency-чувствительный путь — безопаснее держать обе версии запущенными, прогревать на shadow-traffic и переключать LB атомарно. Откат — мгновенный обратный переключатель. |
| Subscription/Notification  | **Rolling**         | Допускает кратковременный лаг (НФТ-006 p95 < 5 мин), идемпотентность по `subscription_id + ad_id`. |
| Media Service              | **Rolling**         | Stateless, тяжёлый трафик upload’а лучше не «дёргать» одновременно; `maxUnavailable=0` обязательно. |
| Indexer ES mapping changes | **Reindex по новому индексу + alias switch** | Меняем индекс не in-place: создаём `ads_v{N+1}`, дозаливаем данные, переключаем alias `ads`. Откат — переключение alias обратно. |
| PostgreSQL / Kafka / ES    | **Managed, rolling по нодам кластера** | Версии — отдельный change window, не привязан к релизу приложения. |

#### Zero-downtime контракт (для stateless-сервисов)

Все сервисы выставляют:

- **readinessProbe** — `GET /ready`: возвращает 200 только когда:
  - подключение к PostgreSQL/Kafka/ES/Redis установлено и pinged;
  - инициализированы внутренние кэши (для Ads/Search — прогрев из БД/ES);
  - rate-limiter и outbox publisher запущены.
  - Параметры: `initialDelaySeconds=5`, `periodSeconds=2`, `failureThreshold=3`. Сервис не получает трафик от Service/LB, пока ready != 200.
- **livenessProbe** — `GET /live`: тривиальный «процесс жив», без проверки внешних зависимостей. Параметры: `initialDelaySeconds=15`, `periodSeconds=10`, `failureThreshold=6`. Цель — рестартить только зависший процесс, а не каскадить рестарты при сбое PG.
- **startupProbe** — для сервисов с прогревом (Ads, Search) — `failureThreshold=30, periodSeconds=2`, чтобы не убить под во время холодного старта.
- **preStop hook + graceful shutdown**:
  1. preStop: `sleep 5` (даёт K8s endpoints/LB время убрать под из rotation),
  2. SIGTERM → сервис переходит в режим «no new requests» (readiness = 503), дорабатывает in-flight за `terminationGracePeriodSeconds=30`,
  3. Outbox/Kafka consumer’ы закрывают сессии (`consumer.Close()` с коммитом текущих оффсетов), HTTP-сервер делает `http.Server.Shutdown(ctx)`.
- **PDB (PodDisruptionBudget)** — `minAvailable: N-1` для каждого сервиса (минимум одна реплика всегда в AZ).
- **Connection draining** на API Gateway — 30 с.

Чек-лист zero-downtime для каждого rolling-релиза:

- [ ] readiness и liveness разделены и не зависят друг от друга;
- [ ] preStop sleep ≥ интервал readiness ×2;
- [ ] `terminationGracePeriodSeconds` ≥ p99 longest request + outbox flush;
- [ ] нет «общего» рестарта (не катимся одновременно во всех AZ);
- [ ] на canary автокат по `error_rate > 1%` или `p99 latency > 2× baseline` на 5 мин окно.

#### Миграции БД (Expand/Contract)

Применяется **Expand/Contract** ко всем изменениям схемы `ads/payments/promotions/subscriptions` — для совместимости двух версий приложения во время rolling/canary.

Фазы:

1. **Expand** (выкатывается отдельным релизом до релиза приложения):
   - добавляем nullable-колонки / новые индексы CONCURRENTLY / новые таблицы;
   - двойная запись (старое и новое поле) делается в коде новой версии;
   - в этой фазе старая версия приложения продолжает работать без изменений.
2. **Migrate / Backfill**: фоновый job-ом дозаполняем новые поля для существующих строк (батчами, с rate-limit, чтобы не уронить PG).
3. **Code switch**: выкатываем приложение, которое читает уже из новой колонки/таблицы (canary).
4. **Contract** (отдельный релиз, через 1–2 недели стабильной работы): убираем старые колонки / удаляем legacy-индексы / снимаем dual-write.

Конкретика:

- Миграции выполняются отдельным K8s Job (sidecar нельзя — должен отработать один раз). Никаких миграций в init-контейнере сервиса.
- `ALTER TABLE ... ADD COLUMN` с `DEFAULT` запрещён без `NULL` (long lock в PG < 11; в нашем PG 16 безопасный default, но всё равно делаем через nullable + backfill для гомогенности).
- `CREATE INDEX` — только `CONCURRENTLY` на горячих таблицах (`ads`, `payments`).
- Любой `DROP COLUMN / DROP INDEX` — только в фазе Contract, после подтверждения, что новая версия везде и нет роллбэка.
- Для **Billing Service** запрещены destructive миграции в одном релизе с кодом — только Expand → wait → Contract отдельным деплоем (НФТ-002).
- Elasticsearch: меняем не «таблицу», а индекс — см. выше «Reindex + alias switch».
- Kafka: новые поля в событиях — только optional + schema registry с backward-compatible compatibility level.

Когда **не применимо**: чистые добавления данных без изменения схемы (insert-only) — обычный релиз; смены runtime-конфигов (через ConfigMap + rolling) — Expand/Contract не нужен.

---

## 2. Observability

### Алерты (4) — только Golden Signals на critical path

Critical path для НФТ-001/003: пользователь → API GW → Search Service → Elasticsearch. Critical path для НФТ-002: пользователь → API GW → Billing Service → PostgreSQL.

| Сигнал     | Метрика                                                                                          | Порог                                                | На что                       |
|------------|--------------------------------------------------------------------------------------------------|------------------------------------------------------|------------------------------|
| Latency    | `histogram_quantile(0.99, sum by (le)(rate(http_request_duration_seconds_bucket{route="GET /api/v1/search/ads"}[5m])))` | **> 200 мс 5 мин подряд** (НФТ-001) | Search Service               |
| Errors     | `sum(rate(http_requests_total{service="billing",code=~"5.."}[5m])) / sum(rate(http_requests_total{service="billing"}[5m]))` | **> 1% 5 мин подряд** (НФТ-002, write-path платежей) | Billing Service              |
| Traffic    | `sum(rate(http_requests_total{route=~"GET /api/v1/(search/ads|ads/.*)"}[1m]))` отклонение от 7-дневного baseline | **±50% от baseline 10 мин подряд** (drop = инцидент, всплеск = capacity warning) | Каталог/поиск (API GW)      |
| Saturation | `pg_stat_activity_count{state="active"} / pg_settings_max_connections` **или** `kafka_consumer_group_lag{group="indexer"}` | **PG > 80% 5 мин** ИЛИ **lag > 100k сообщений 10 мин** | PostgreSQL / Indexer (Kafka) |

Принципы:

- Все алерты с `for: 5m` (кроме traffic — 10m), чтобы не звенеть на флапах.
- Алерт без runbook = не алерт; ссылка на runbook в annotation.
- Алерт «latency search» бьёт раньше, чем алерт «error rate каталога», потому что деградация поиска первой ломает UX.

### Дашборды (3 уровня)

1. **Overview** (1 экран, для on-call и руководства):
   - 4 Golden Signals по системе в целом: общий RPS, общий 5xx-rate, p99 latency endpoint поиска, общая сатурация (CPU / PG conn / Kafka lag).
   - SLO burn-rate (1h / 6h windows) для НФТ-001 и НФТ-003.
   - Health-плитка по сервисам (зелёный/жёлтый/красный по readiness).
   - Прямой ответ на вопрос «нам сейчас плохо?» за 5 секунд.
2. **Service-level** (по дашборду на сервис, фокус — RED):
   - **R**ate: RPS по route × status code.
   - **E**rrors: error-rate по route, топ-10 ошибок (по `error.kind`).
   - **D**uration: p50/p95/p99 latency по route, heatmap.
   - Дополнительно по сервису: для Ads — outbox lag, для Search — query latency vs ES query latency (где теряем), для Billing — idempotency-key collisions, для Indexer — Kafka consumer lag.
3. **Diagnostic** (USE по ресурсам + трейсы):
   - **U**tilization / **S**aturation / **E**rrors по узлам и контейнерам: CPU, RAM, disk I/O, network, PG connections, Kafka broker disk, ES heap, Redis evictions.
   - Per-node graphs для разбора «почему у нас p99 уехал».
   - Встроенная панель с trace search (Tempo/Jaeger) по `trace_id` из логов — клик в алерт открывает соответствующие трейсы.

### Логи

- **Формат**: JSON в stdout (контейнеры → агрегатор Loki/ELK). Обоснование: structured logging без парсинга, единый стандарт, простой grep по полям, дешёвая фильтрация в pipeline.
- **Обязательные поля**: `timestamp` (RFC3339 UTC), `level` (`debug|info|warn|error`), `service` (`ads|billing|search|...`), `service_version` (git-sha), `env` (`prod|stage`), `trace_id`, `span_id`, `msg`, `http.method`, `http.route`, `http.status`, `http.duration_ms`, `user_id` (если был JWT), `ad_id` / `payment_id` / `idempotency_key` (если применимо для записи), `error.kind` и `error.stack` (для уровня `error`).
- **Что логируем**:
  - входящий HTTP-запрос (после middleware): метод, route, статус, длительность, размер ответа;
  - исходящий вызов в зависимость (PG/ES/Kafka/S3/external): имя зависимости, длительность, статус, retry-count, circuit-breaker state;
  - переход состояния домена (FSM ads, status платежа, активация promotion);
  - ошибки уровня `error` со стеком и контекстом (`error.kind` machine-readable);
  - старт/остановка сервиса и подключение к зависимостям.
- **НЕ логируем**:
  - PII: email, телефон, ФИО, адреса — только хешированные/маскированные (`user_id` ок, `email` — нет);
  - payment PAN, CVV, токены эквайера, JWT целиком (только `user_id` и `kid`);
  - тело request/response с пользовательским контентом (текст объявления) — только размеры и категории;
  - секреты, пароли, S3 credentials, `Authorization` header, query string с токенами;
  - бинарный media-контент.
- **Retention**: hot 7 дней (всё), warm 30 дней (info+), cold 90 дней (warn+). Audit-логи биллинга — отдельный bucket 1 год (НФТ-002, регуляторика).
