# ADR-003: Exactly-once payment workflow via idempotency + outbox

## Status

Accepted

## Context

Требование бизнеса: не терять платежи за продвижение, даже если модерация/публикация задерживается.
НФТ: 0% потерь платежных транзакций.

Критичный риск — двойные списания при ретраях клиента и потеря событий при падении между DB commit и publish.

## Alternatives

1. **Synchronous direct activation only**
   - Pros: простая схема.
   - Cons: ломается при длительной модерации, выше риск потерь/таймаутов.

2. **2PC между сервисами**
   - Pros: формально сильная консистентность.
   - Cons: высокая сложность и latency, плохо масштабируется в микросервисах.

3. **Idempotency key + transactional outbox + async activation**
   - Pros: надежный платежный контур, контролируемая согласованность, лучше под highload.
   - Cons: сложнее реализация и мониторинг.

## Decision

Выбираем `idempotency + outbox`:
- обязательный `Idempotency-Key` для API оплаты;
- атомарная запись `payments/promotions/outbox` в одной транзакции PostgreSQL;
- отдельный publisher в Kafka;
- активация продвижения только после `ad.published`.

## Consequences

### Positive
- Нет потерь событий между транзакцией и публикацией.
- Защита от двойного списания при повторах.
- Четкий жизненный цикл `pending_publication -> activated`.

### Negative
- Нужен отдельный outbox процесс и DLQ для ошибок.
- Усложняется tracing сквозного user flow.
