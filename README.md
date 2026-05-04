# Практическая работа №4 (семестр 2)

## Выполнила: Сорокина К.С., ЭФМО-01-25

## Тема: Метрики приложения, Prometheus, Grafana и интеграция с HTTP сервисом

### Цель:

Научиться собирать и визуализировать метрики сервиса: трафик, ошибки, задержки, активные запросы.

## Технологии

- **Go** — язык реализации
- **Prometheus** — сбор и хранение метрик
- **Grafana** — визуализация метрик
- **Docker Compose** — запуск Prometheus и Grafana
- **client_golang** — библиотека Prometheus для Go

## Структура проекта

```
PR4_sem2/
├── go.mod
├── go.sum
├── proto/
├── shared/
│   ├── logger/
│   │   └── logger.go
│   └── middleware/
│       ├── accesslog.go
│       ├── metrics.go         ← middleware для сбора метрик
│       └── requestid.go
├── services/
│   ├── auth/
│   │   ├── cmd/auth/main.go
│   │   └── internal/grpc/server.go
│   └── tasks/
│       ├── cmd/tasks/main.go  ← подключены метрики и /metrics endpoint
│       └── internal/handler.go
└── deploy/
    └── monitoring/
        ├── docker-compose.yml ← Prometheus + Grafana
        └── prometheus.yml     ← конфигурация scrape
```

## Метрики

В сервис tasks добавлены 3 группы метрик:

| Метрика | Тип | Labels | Описание |
|---|---|---|---|
| `http_requests_total` | Counter | method, route, status | Общее количество запросов |
| `http_request_duration_seconds` | Histogram | method, route | Длительность запросов |
| `http_in_flight_requests` | Gauge | — | Активные запросы прямо сейчас |

### Почему именно эти метрики:
- **Counter** для запросов — только растёт, удобно считать RPS через `rate()`
- **Histogram** для длительности — позволяет считать перцентили (p95, p99)
- **Gauge** для активных запросов — показывает текущую нагрузку в реальном времени

## Endpoint /metrics

Сервис tasks отдаёт метрики в формате Prometheus по адресу `GET /metrics`.

### Пример вывода:

```
# HELP http_requests_total Общее количество HTTP запросов
# TYPE http_requests_total counter
http_requests_total{method="GET",route="/tasks",status="200"} 13
http_requests_total{method="GET",route="/tasks",status="401"} 8

# HELP http_request_duration_seconds Длительность HTTP запросов в секундах
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",route="/tasks",le="0.01"} 21
http_request_duration_seconds_bucket{method="GET",route="/tasks",le="0.05"} 21
http_request_duration_seconds_bucket{method="GET",route="/tasks",le="+Inf"} 21

# HELP http_in_flight_requests Количество активных запросов в данный момент
# TYPE http_in_flight_requests gauge
http_in_flight_requests 1
```

## Конфигурация

### deploy/monitoring/prometheus.yml

```yaml
global:
  scrape_interval: 5s

scrape_configs:
  - job_name: tasks
    static_configs:
      - targets:
          - host.docker.internal:8082
```

Prometheus опрашивает сервис tasks каждые 5 секунд. `host.docker.internal` позволяет Docker-контейнеру обращаться к локальному сервису.

### deploy/monitoring/docker-compose.yml

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    extra_hosts:
      - "host.docker.internal:host-gateway"

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    depends_on:
      - prometheus
```

## Инструкция запуска

### 1. Запустить auth сервис

```bash
go run ./services/auth/cmd/auth
```
![](https://github.com/krrristina/PR4_sem2/blob/main/screenshots/запуск%20auth.png)
### 2. Запустить tasks сервис

```bash
go run ./services/tasks/cmd/tasks
```
![](https://github.com/krrristina/PR4_sem2/blob/main/screenshots/запуск%20tasks.png)
### 3. Запустить Prometheus и Grafana

```bash
cd deploy/monitoring
docker compose up -d
```
![](https://github.com/krrristina/PR4_sem2/blob/main/screenshots/containers%20created.png)

### 4. Проверить метрики напрямую

```bash
curl http://localhost:8082/metrics
```
![](https://github.com/krrristina/PR4_sem2/blob/main/screenshots/metrics.png)

![](https://github.com/krrristina/PR4_sem2/blob/main/screenshots/metrics%202.png)

### 5. Открыть интерфейсы

| Сервис | Адрес | Логин/Пароль |
|---|---|---|
| Prometheus | http://localhost:9090 | — |
| Grafana | http://localhost:3000 | admin / admin |

## Проверка через Postman

### Успешные запросы

- Method: `GET`
- URL: `http://localhost:8082/tasks`
- Headers: `Authorization: Bearer my-test-token`

![](https://github.com/krrristina/PR4_sem2/blob/main/screenshots/все%20работает.png)

### Запросы с ошибкой

- Method: `GET`
- URL: `http://localhost:8082/tasks`
- Headers: `Authorization: Bearer invalid-token`

![](https://github.com/krrristina/PR4_sem2/blob/main/screenshots/unauthorized.png)

## Prometheus Targets

Prometheus успешно собирает метрики с сервиса tasks — статус **UP**.

![](https://github.com/krrristina/PR4_sem2/blob/main/screenshots/prometheys%20работает.png)

## Grafana Dashboard — Tasks metrics

### График 1 — RPS (запросы в секунду)

**Запрос:** `rate(http_requests_total[1m])`

Показывает количество запросов в секунду с разбивкой по статусу ответа (200, 401).

![](https://github.com/krrristina/PR4_sem2/blob/main/screenshots/график%201.png)

### График 2 — Ошибки 4xx/5xx

**Запрос:** `rate(http_requests_total{status=~"4..|5.."}[1m])`

Показывает только ошибочные запросы — удобно для алертинга.

![](https://github.com/krrristina/PR4_sem2/blob/main/screenshots/график%202%20(запросы%20с%20неверным%20токеном).png)

### График 3 — Latency p95

**Запрос:** `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[1m]))`

Показывает время ответа, в которое укладываются 95% запросов.

![](https://github.com/krrristina/PR4_sem2/blob/main/screenshots/Latency%20p95.png)

## Контрольные вопросы

### Вопрос 1. Чем метрики отличаются от логов и зачем нужны оба подхода?

Логи — это подробная запись каждого события (что именно произошло, с какими данными). Метрики — это числовые показатели за период времени (сколько раз, как быстро). Логи нужны для детальной диагностики конкретной ошибки, метрики — для мониторинга общего состояния системы и построения алертов. Вместе они дают полную картину: метрики сигнализируют что что-то пошло не так, а логи объясняют почему.

### Вопрос 2. Чем Counter отличается от Gauge?

Counter — только растёт (или сбрасывается при рестарте). Используется для подсчёта событий: запросов, ошибок, байт. Gauge — может расти и убывать, отражает текущее состояние: количество активных подключений, использование памяти, температуру. Для Counter используют `rate()` чтобы получить скорость изменения, для Gauge берут значение напрямую.

### Вопрос 3. Почему latency нужно измерять histogram, а не просто средним значением?

Среднее значение скрывает выбросы. Например, если 99 запросов выполняются за 10мс, а 1 за 10 секунд — среднее будет около 110мс, что не отражает реальную картину. Histogram позволяет считать перцентили: p95 показывает время, в которое укладываются 95% запросов, p99 — 99%. Это честнее показывает пользовательский опыт, так как именно медленные запросы создают проблемы.

### Вопрос 4. Что такое labels и почему опасна высокая кардинальность?

Labels — это теги которые добавляют контекст к метрике (например method, route, status). Кардинальность — это количество уникальных комбинаций значений labels. Высокая кардинальность опасна тем, что Prometheus хранит отдельный временной ряд для каждой комбинации. Например, если в label записывать user_id — при миллионе пользователей получится миллион рядов, что перегрузит память и диск.

### Вопрос 5. Зачем нужны p95/p99 и почему среднее может "врать"?

p95 означает что 95% запросов выполняются быстрее этого значения. p99 — 99% быстрее. Среднее "врёт" потому что на него сильно влияют редкие но большие выбросы, и оно не показывает что происходит с медленными запросами. Если у вас p95 = 100мс, а p99 = 5 секунд — значит каждый сотый пользователь ждёт 5 секунд, хотя среднее может выглядеть вполне нормально.
