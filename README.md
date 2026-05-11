# Маркетплейс объявлений — кейс B (Highload)

Кейс: объявления, модерация, поиск, оплата продвижения, подписки (см. `docs/`).

## Задание 3 (PoC)

Требования к репозиторию и нагрузочному циклу: [ДЗ 3: PoC + оптимизация](https://github.com/RobertMaz/MIPT-Highload-2026/blob/main/homework/Assignment%2003%20-%20PoC.md), материалы курса: [Highload · ВШПИ × МТС 2026](https://robertmaz.github.io/MIPT-Highload-2026/#/).

### 1. Как запустить

Из корня репозитория (нужны Docker и Docker Compose v2):

```bash
docker compose up -d --build
```

Дождаться зелёных healthcheck’ов (первый старт PostgreSQL может занять ~30–60 с):

```bash
docker compose ps
```

Сервисы и порты на хосте:

| Сервис          | Порт | Healthcheck                    |
|-----------------|------|--------------------------------|
| `ads-service`   | 8081 | `GET http://localhost:8081/health` |
| `billing-service` | 8082 | `GET http://localhost:8082/health` |
| `postgres`      | (только сеть Docker) | `pg_isready` внутри контейнера |

Пересоздать БД с нуля (если менялась схема): `docker compose down -v && docker compose up -d --build`.

### 2. Как проверить (curl, happy path из ДЗ 2)

Создать объявление (статус `moderation_pending` по OpenAPI):

```bash
curl -sS -X POST http://localhost:8081/api/v1/ads \
  -H 'Content-Type: application/json' \
  -d '{"title":"Велосипед"}' | jq .
```

Ожидается **201** и JSON с полями `id`, `title`, `status` (`moderation_pending`).

Опубликовать (имитация успешной модерации → `published`):

```bash
export AD_ID='<uuid из ответа create>'
curl -sS -X POST "http://localhost:8081/api/v1/ads/${AD_ID}/publish" | jq .
```

Ожидается **200**, `status`: `published`.

Получить карточку:

```bash
curl -sS "http://localhost:8081/api/v1/ads/${AD_ID}" | jq .
```

Ожидается **200**.

Купить продвижение (billing проверяет объявление по HTTP в ads, пишет в БД):

```bash
curl -sS -X POST "http://localhost:8082/api/v1/ads/${AD_ID}/promotions" | jq .
```

Ожидается **201** и поля `id`, `adId`, `status`, `createdAt`.

Проверка health:

```bash
curl -sS http://localhost:8081/health && echo
curl -sS http://localhost:8082/health && echo
```

Ожидается **200** и `{"status":"ok"}` (включая успешный `Ping` к PostgreSQL).

### 3. Как запустить нагрузочный тест

На **ноутбуке** (не на VM приложения), при установленном [k6](https://k6.io/docs/):

```bash
export BASE_URL=http://<IP_VM>:8081
export BILLING_URL=http://<IP_VM>:8082
k6 run loadtest/smoke.js
```

Метрики: сводка k6 в консоли; на VM — `docker stats`, `htop`, `iostat` (см. ДЗ 3). Полный цикл baseline и iter 1–3 — в `docs/optimization-log.md`.

### 4. Паттерны (≥4: проектирование + устойчивость)

| Паттерн | Зачем | Где в коде |
|---------|--------|------------|
| **Ports & Adapters (чистая архитектура)** | Домен не зависит от БД/HTTP; тестируемость | `services/*/internal/domain/ports`, `application/usecases`, `infrastructure/adapters` |
| **CQRS (упрощённо)** | Запись объявлений в PostgreSQL; межсервисная **проверка** опубликованного объявления — read через HTTP в ads | Write: `services/ads-service/.../postgres/ads_repository.go`; read-side check: `services/billing-service/.../http/ads_catalog_adapter.go` |
| **Rate limiting** | Защита create от burst (429 при перегрузе) | `services/ads-service/internal/presentation/http/middleware.go` |
| **Timeouts** | Ограничение висящих исходящих вызовов | `services/billing-service/internal/di/providers.go` (`http.Client.Timeout`) |
| **Retry with exponential backoff** | Временные 5xx / сетевые сбои при вызове ads | `services/billing-service/internal/infrastructure/adapters/http/ads_catalog_adapter.go` (`getWithRetry`) |
| **Circuit breaker** | Не долбить упавший ads при каскадном сбое | `services/billing-service/.../ads_catalog_adapter.go` (`gobreaker`) |
| **Healthcheck + DB ping** | Readiness: сервис не «зелёный», пока БД недоступна | `services/*/internal/presentation/http/handlers.go` (`GET /health`), `docker-compose.yml` |

### 5. Итерации оптимизации

Шаблон таблицы и место для RED/USE и bottleneck — в [`docs/optimization-log.md`](docs/optimization-log.md). Git-теги `iter-0`, `iter-1`, … — по инструкции ДЗ 3 после замеров.

---

## Структура репозитория (ДЗ 3)

```
├── docker-compose.yml
├── deployments/postgres/init.sql
├── services/
│   ├── ads-service/
│   └── billing-service/
├── loadtest/
│   └── smoke.js
├── docs/
│   ├── optimization-log.md
│   └── ...
└── README.md
```

## Локальная разработка

```bash
make test    # unit-тесты
make wire    # пересборка Wire DI
```
