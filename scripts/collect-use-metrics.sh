#!/usr/bin/env bash
# Параллельный сбор USE-метрик на VM во время прогона k6 с ноутбука.
#
# Запуск (на VM, в корне репозитория):
#   ./scripts/collect-use-metrics.sh <scenario> <duration_seconds>
#
# Примеры:
#   ./scripts/collect-use-metrics.sh read  900   # 15 минут
#   ./scripts/collect-use-metrics.sh mixed 360   # 6 минут
#   ./scripts/collect-use-metrics.sh stress 900
#   ./scripts/collect-use-metrics.sh spike  360
#
# Результаты пишутся в docs/baseline/raw/<scenario>/<timestamp>/:
#   meta.txt           — параметры запуска, uname, лимиты compose
#   docker-stats.log   — docker stats --no-stream, 1 снимок / 5 с
#   vmstat.log         — vmstat 5, утилизация CPU/IO/memory
#   iostat.log         — iostat -xz 5, насыщение диска (HDD!)
#   free.log           — free -m, RAM, 1 раз / 10 с
#   pg-connections.log — pg_stat_activity по состояниям, 1 раз / 10 с
#   ads-health.log     — HTTP-код /health ads, 1 раз / 5 с
#   billing-health.log — то же для billing
#
# Скрипт завершает все фоновые джобы по истечении duration.
set -euo pipefail

SCENARIO="${1:-unnamed}"
DURATION="${2:-600}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="${ROOT}/docs/baseline/raw/${SCENARIO}/${TS}"
mkdir -p "${OUT}"

echo "==> Сбор USE-метрик: scenario=${SCENARIO} duration=${DURATION}s"
echo "==> Каталог результатов: ${OUT}"

# ---- meta ----
{
  echo "scenario: ${SCENARIO}"
  echo "duration_sec: ${DURATION}"
  echo "started_utc: ${TS}"
  echo "host: $(hostname)"
  echo "uname: $(uname -a)"
  echo
  echo "--- docker compose ps ---"
  docker compose ps || true
  echo
  echo "--- docker compose config (deploy.resources) ---"
  docker compose config 2>/dev/null | grep -E "(services|cpus|memory|container_name):" || true
} > "${OUT}/meta.txt"

# ---- background collectors ----
pids=()

# docker stats: снимок каждые 5 с
(
  end=$(( $(date +%s) + DURATION ))
  while [ "$(date +%s)" -lt "${end}" ]; do
    {
      date -u +%Y-%m-%dT%H:%M:%SZ
      docker stats --no-stream --format \
        "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}\t{{.PIDs}}"
      echo
    } >> "${OUT}/docker-stats.log" 2>&1
    sleep 5
  done
) & pids+=($!)

# vmstat: 1 строка / 5 с
(
  vmstat 5 $(( DURATION / 5 + 1 )) >> "${OUT}/vmstat.log" 2>&1
) & pids+=($!)

# iostat (sysstat) — best effort
if command -v iostat >/dev/null 2>&1; then
  (
    iostat -xz 5 $(( DURATION / 5 + 1 )) >> "${OUT}/iostat.log" 2>&1
  ) & pids+=($!)
else
  echo "iostat не установлен. Используйте: sudo apt-get install -y sysstat" \
    > "${OUT}/iostat.log"
fi

# free -m: 1 раз / 10 с
(
  end=$(( $(date +%s) + DURATION ))
  while [ "$(date +%s)" -lt "${end}" ]; do
    {
      date -u +%Y-%m-%dT%H:%M:%SZ
      free -m
      echo
    } >> "${OUT}/free.log" 2>&1
    sleep 10
  done
) & pids+=($!)

# pg connections: 1 раз / 10 с
(
  end=$(( $(date +%s) + DURATION ))
  while [ "$(date +%s)" -lt "${end}" ]; do
    {
      date -u +%Y-%m-%dT%H:%M:%SZ
      docker compose exec -T postgres psql -U app -d marketplace -c \
        "SELECT state, count(*) FROM pg_stat_activity WHERE datname='marketplace' GROUP BY state ORDER BY 2 DESC;" \
        2>&1 || true
      echo
    } >> "${OUT}/pg-connections.log"
    sleep 10
  done
) & pids+=($!)

# health-проверки: фиксируют моменты, когда сервис перестаёт отвечать
(
  end=$(( $(date +%s) + DURATION ))
  while [ "$(date +%s)" -lt "${end}" ]; do
    ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    ads_code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 http://127.0.0.1:8081/health || echo CURLFAIL)"
    bil_code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 http://127.0.0.1:8082/health || echo CURLFAIL)"
    echo "${ts} ${ads_code}" >> "${OUT}/ads-health.log"
    echo "${ts} ${bil_code}" >> "${OUT}/billing-health.log"
    sleep 5
  done
) & pids+=($!)

trap 'echo "==> Прерывание, останавливаю коллекторы..."; for p in "${pids[@]}"; do kill "$p" 2>/dev/null || true; done; wait 2>/dev/null || true; exit 130' INT TERM

echo "==> Коллекторы запущены (PIDs: ${pids[*]}). Жду ${DURATION} секунд..."
sleep "${DURATION}"

# Финальный снимок после теста
{
  echo
  echo "--- final docker stats ---"
  docker stats --no-stream
  echo
  echo "--- final docker compose ps ---"
  docker compose ps
  echo
  echo "--- container OOM/exit codes ---"
  docker inspect \
    --format='{{.Name}} OOMKilled={{.State.OOMKilled}} ExitCode={{.State.ExitCode}} Status={{.State.Status}}' \
    $(docker compose ps -q) 2>/dev/null || true
} >> "${OUT}/meta.txt"

echo "==> Останавливаю коллекторы..."
for p in "${pids[@]}"; do
  kill "$p" 2>/dev/null || true
done
wait 2>/dev/null || true

echo
echo "=== Файлы в ${OUT}: ==="
ls -lh "${OUT}"
