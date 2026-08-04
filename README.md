# multi-awg

Админ-панель для своего AmneziaWG VPN-сервера. Один клик — и готовый набор инструментов для управления сервером.

## Возможности

- **awg-proxy** — маскирует ответы сервера под QUIC, SIP, STUN, DNS и т.д. (обход DPI)
- **mihomo-ядро** — форвардинг трафика с AmneziaWG, поддержка каскадных подключений и настройка для работы в RU-регионах
- **удобный frontend** — панель управления (написан ИИ)
- **поддержка AmneziaWG 3.0**

## Архитектура

```
Frontend ──HTTP──► Backend (:8080) ──HTTP──► Worker (:9090)
                                                  │
                                            awg-proxy + amneziawg + mihomo
```

## Настройка

### 1. Конфиг AmneziaWG

Настройте `amnezia-config/awg0.conf`. В проекте уже лежит пример. Поддерживается AWG 3.0.

### 2. awg-proxy

Настройте `awg-proxy/proxy.toml` — здесь конфигурируется имитация протокола (`imitate_protocol`: `quic`, `sip`, `dns`, `stun`, `auto`).

### 3. mihomo

Настройте `mihomo-config/config.yaml` — используется при форвардинге трафика с AmneziaWG.

В этой же папке настройте `mihomo-config/example.yaml` — пример конфига, который выдаётся в подписке пользователям.

> **Важно:** значения `nameConfig`, `private_key`, `server_ip`, `server_port`, `client_ip`, `public_key` менять не нужно — это захардкоженные плейсхолдеры, которые подменяются автоматически.

### 4. Переменные окружения

- `.env.server` — backend (см. `.env.server.example`): `DB_PATH`, `WORKER_URL`, `WORKER_TOKEN`, `JWT_SECRET`, `MAX_CONFIGS`
- `.env.worker` — worker (см. `.env.worker.example`): `AUTH_TOKEN`, `SERVER_ENDPOINT`, `MIHOMO_TEMPLATE`

### 5. Константные переменные
- Название awg0.conf, переменная ListenPort
- В proxy.toml переменные listen и backend
- В config.yaml переменная device

## Запуск через docker-compose

```bash
docker compose up
```

Поднимает стек: mihomo, amneziawg, awg-proxy, worker и backend (`server`).

## Локальный запуск без Docker

- Backend: `go run ./cmd/server/`
- Worker: `go run ./cmd/worker/`

## Полезные ссылки

Список приложений для импорта подписки clash — в `TODO.md`.
