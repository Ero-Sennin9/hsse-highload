# ADR-002: Search read-model via Elasticsearch (CQRS)

## Status

Accepted

## Context

Главный пользовательский сценарий — поиск объявлений с фильтрами и сортировкой.
НФТ: `p99 < 200 ms` на пиковых нагрузках.

Прямой поиск в PostgreSQL по большим объемам с полнотекстом и фильтрами приведет к дорогому scale-up и росту latency.

## Alternatives

1. **PostgreSQL only (GIN/FTS + индексы)**
   - Pros: один источник, проще консистентность.
   - Cons: дорогое масштабирование чтения, слабее full-text ranking и faceting.

2. **External search engine (Elasticsearch) as read-model**
   - Pros: быстрый full-text, фильтры, горизонтальный scale-out.
   - Cons: eventual consistency, отдельный operational контур.

3. **OpenSearch/ClickHouse full-text approach**
   - Pros: тоже решает read-heavy.
   - Cons: дополнительные риски миграции и экспертизы команды сейчас.

## Decision

Используем CQRS:
- write-модель: PostgreSQL (source of truth);
- read-модель: Elasticsearch;
- синхронизация через события `ad.published`, `ad.updated`, `ad.archived`.

## Consequences

### Positive
- Выполнение SLO по задержке поиска.
- Гибкая сортировка/релевантность и быстрые фильтры.
- Поиск масштабируется отдельно от транзакционного контура.

### Negative
- Появляется lag индексации (допустимый staleness 2-3 минуты).
- Нужны idempotent indexer и восстановление индекса после сбоев.
