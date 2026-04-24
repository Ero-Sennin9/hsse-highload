# Assignment 02: Architecture (Case B Marketplace)

## 1. Архитектурный стиль + обоснование

### Выбранный стиль

Гибрид `SBA + EDA + CQRS`:
- `SBA` (service-based): отдельные крупные сервисы по бизнес-областям;
- `EDA`: асинхронная интеграция через брокер событий;
- `CQRS` в поиске: write-модель в PostgreSQL, read-модель в Elasticsearch.

### Почему этот стиль подходит под требования

1. НФТ по поиску (`p99 < 200 ms`) требует выделенной read-модели и дешевого горизонтального масштабирования чтения.
2. НФТ по платежам (0% потерь) требует изоляции биллинга и надежной асинхронной обработки (`outbox + idempotency`).
3. НФТ по модерации (`p95 < 5s`) требует фоновой обработки без блокировки user flow.
4. НФТ по media (`p99 thumbnail < 150 ms`) требует хранения в S3 и доставки через CDN.
5. Рост до 2x-3x MAU поддерживается scale-out за счет stateless API и асинхронных контуров.

### Основные trade-off

- Плюсы: масштабируемость, отказоустойчивость, независимое развитие подсистем.
- Минусы: eventual consistency между write/read-путями, сложнее эксплуатация и observability.

## 2. Компоненты (C4 L1 + L2 с описаниями)

Диаграммы (Mermaid исходники):
- `docs/diagrams/c4-l1-system-context.mmd`
- `docs/diagrams/c4-l2-container.mmd`

### Список компонентов (5–10)

| Компонент | Назначение | Технология | Коммуникация |
|---|---|---|---|
| API Gateway | Единая точка входа, auth/rate limit/routing | Nginx/Kong | sync HTTP |
| Ads Service | CRUD объявлений, статусы, публикация | Go + Gin/Fiber | sync HTTP + async events |
| Moderation Service | Авто/ручная модерация объявлений | Go/Python worker | async (Kafka) |
| Search Service | Поиск и фильтрация, read API | Go + Elasticsearch client | sync HTTP |
| Billing Service | Оплата продвижения, idempotency, outbox | Go + PostgreSQL | sync HTTP + async events |
| Subscription/Notification Service | Подписки и email рассылка | Go + worker + SMTP provider | async (Kafka) |
| PostgreSQL | Source of truth для транзакционных данных | PostgreSQL 16 | sync DB |
| Elasticsearch | Полнотекстовый индекс и фильтры | Elasticsearch 8 | sync query, async indexing |
| Kafka | Интеграция и события домена | Apache Kafka | async pub/sub |
| S3 + CDN | Хранение и быстрая доставка фото | MinIO/S3 + CDN | sync upload, CDN delivery |

## 3. Sequence diagrams (3 сценария)

Диаграммы:
- `docs/diagrams/sequence-happy-path.mmd`
- `docs/diagrams/sequence-error-path.mmd`
- `docs/diagrams/sequence-async-path.mmd`

### 3.1 Happy path (создание и публикация объявления)

Поток:
1. Клиент создает объявление.
2. Ads Service пишет в PostgreSQL со статусом `moderation_pending`.
3. Ads Service публикует `ad.created` в Kafka.
4. Moderation Service проверяет и публикует `ad.approved`.
5. Ads Service переводит объявление в `published`.
6. Indexer обновляет Elasticsearch, объявление появляется в поиске.

### 3.2 Сценарий с ошибкой (таймаут search backend)

Поток:
1. Клиент делает запрос поиска.
2. Search Service получает timeout от Elasticsearch.
3. Срабатывает circuit breaker.
4. Возвращается `503` с корректным error payload (fail fast).
5. API Gateway не блокирует другие пользовательские сценарии.

### 3.3 Асинхронный сценарий (платеж + активация продвижения)

Поток:
1. Клиент вызывает оплату продвижения с `Idempotency-Key`.
2. Billing Service создает payment + promotion (`pending_publication`) в одной транзакции.
3. Outbox publisher публикует `promotion.paid`.
4. После события `ad.published` Ads Service публикует `promotion.activate.requested`.
5. Billing Service активирует продвижение и публикует `promotion.activated`.
6. Search/Notification обновляют выдачу и уведомляют пользователя.

## 4. API (3–5 endpoint'ов)

OpenAPI спецификация:
- `docs/api/openapi.yaml`

Формат:
- OpenAPI `3.0.3`
- URI versioning: `/api/v1/...`

Реализованные операции:
1. `POST /api/v1/ads` — создание объявления;
2. `GET /api/v1/ads/{adId}` — получение карточки объявления;
3. `GET /api/v1/search/ads` — полнотекстовый поиск с фильтрами;
4. `POST /api/v1/ads/{adId}/promotions` — оплата продвижения;
5. `POST /api/v1/subscriptions` — подписка на поисковый запрос.

Во всех методах определены:
- request/response payload;
- коды ошибок (`400/401/404/409/429/503`);
- контракт версионирования;
- idempotency на платежной операции через `Idempotency-Key`.

## 5. Выбор БД + модель данных

### 5.1 Выбор хранилищ

| Хранилище | Почему выбрано | Паттерн доступа |
|---|---|---|
| PostgreSQL | ACID, транзакции, надежный write-path (ads/payments) | OLTP, point reads/writes |
| Elasticsearch | Быстрый полнотекст + фильтры + сортировка | read-heavy search |
| Redis | Кэш горячих карточек и ограничителей | hot key-value reads |
| S3/MinIO | Дешевое и масштабируемое хранение media | binary objects |
| Kafka | Надежная асинхронная интеграция | event streaming |

### 5.2 Модель данных (3–5 сущностей)

#### 1) `ads`
- PK: `id (uuid)`.
- Поля: `seller_id`, `title`, `description`, `category`, `region`, `price`, `status`, `published_at`, `created_at`.
- Индексы:
  - `idx_ads_seller_created_at (seller_id, created_at desc)` для ЛК;
  - `idx_ads_status_published_at (status, published_at desc)` для listing;
  - `idx_ads_category_region_price (category, region, price)` для фильтров.
- Объем: десятки миллионов активных записей, рост линейно с MAU.

#### 2) `ad_media`
- PK: `id (uuid)`.
- FK: `ad_id -> ads.id`.
- Поля: `storage_key`, `thumbnail_key`, `position`, `created_at`.
- Индексы:
  - `idx_ad_media_ad_id_position (ad_id, position)`.
- Объем: до `8 * ads` записей.

#### 3) `payments`
- PK: `id (uuid)`.
- Поля: `ad_id`, `user_id`, `idempotency_key`, `amount`, `currency`, `status`, `provider_tx_id`, `created_at`.
- Ограничения:
  - `uniq_payments_idempotency_key`.
- Индексы:
  - `idx_payments_user_created_at (user_id, created_at desc)`;
  - `idx_payments_ad_id (ad_id)`.
- Объем: пропорционален количеству покупок продвижения.

#### 4) `promotions`
- PK: `id (uuid)`.
- FK: `payment_id -> payments.id`, `ad_id -> ads.id`.
- Поля: `type`, `status`, `starts_at`, `expires_at`, `created_at`.
- Индексы:
  - `idx_promotions_ad_status (ad_id, status)`;
  - `idx_promotions_expires_at (expires_at)`.

#### 5) `subscriptions`
- PK: `id (uuid)`.
- Поля: `user_id`, `query_json`, `channel`, `status`, `created_at`.
- Индексы:
  - `idx_subscriptions_user_status (user_id, status)`.

### 5.3 Ключевые partition/sharding идеи

- `ads`: горизонтальное шардирование по `hash(id)` при выходе за лимиты single-writer.
- Kafka topics: партиционирование по `ad_id` для локального порядка событий по объявлению.
- Elasticsearch: индексы по времени и rollover для контролируемого retention.

## 6. ADR (Architecture Decision Records)

Файлы ADR:
- `docs/adr/ADR-001-architecture-style.md`
- `docs/adr/ADR-002-search-cqrs-elasticsearch.md`
- `docs/adr/ADR-003-payments-outbox-idempotency.md`

Каждый ADR содержит:
- context;
- alternatives (2+);
- decision;
- consequences.
