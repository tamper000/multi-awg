# Docker Worker

HTTP API для управления AmneziaWG пирами.

## Запуск

Требуется Docker daemon и Go 1.26+.

```bash
cp .env.example .env  # настрой токен
go run ./cmd/worker/
```

Сервер стартует на `:9090`.

## Переменные окружения

| Переменная | По умолчанию | Описание |
|-----------|-------------|----------|
| `PORT` | `9090` | Порт HTTP API |
| `AUTH_TOKEN` | (пусто) | Токен авторизации. Если не задан — авторизация отключена |

## API

### Авторизация

Если задан `AUTH_TOKEN`, все запросы кроме `/api/health` требуют заголовок:

```
Authorization: Bearer <token>
```

### Health check

```bash
curl http://localhost:9090/api/health
```

```json
{"status": "ok"}
```

### Создать пир

```bash
curl -X POST http://localhost:9090/api/peers \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -d '{"name": "my-phone"}'
```

```json
{
  "name": "my-phone",
  "ip": "10.0.0.3",
  "public_key": "abc123...",
  "config": "[Interface]\nPrivateKey = ...\nAddress = 10.0.0.3/32\n..."
}
```

С кастомным DNS:

```bash
curl -X POST http://localhost:9090/api/peers \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -d '{"name": "laptop", "dns": "8.8.8.8"}'
```

Имя должно быть уникальным. При дубликате вернётся 409:

```json
{"error": "peer name already exists"}
```

### Список пиров

```bash
curl http://localhost:9090/api/peers \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

```json
[
  {"name": "my-phone", "ip": "10.0.0.3", "created_at": "2026-08-01T12:00:00Z"},
  {"name": "laptop", "ip": "10.0.0.4", "created_at": "2026-08-01T12:05:00Z"}
]
```

### Получить конфиг пира

```bash
curl http://localhost:9090/api/peers/my-phone/config \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

```json
{
  "config": "[Interface]\nPrivateKey = ...\nAddress = 10.0.0.3/32\nDNS = 1.1.1.1\nMTU = 1280\n\n[Peer]\nPublicKey = ...\nAllowedIPs = 0.0.0.0/0, ::/0\nPersistentKeepalive = 25\n"
}
```

### Удалить пир

```bash
curl -X DELETE http://localhost:9090/api/peers/my-phone \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

```json
{"status": "deleted"}
```

### Статистика всех пиров

```bash
curl http://localhost:9090/api/stats \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

```json
[
  {"name": "my-phone", "ip": "10.0.0.3", "received": 0, "sent": 0, "last_handshake": 0},
  {"name": "laptop", "ip": "10.0.0.4", "received": 257662, "sent": 2226272, "last_handshake": 1785763093}
]
```

`received`/`sent` — байты, `last_handshake` — unix-время последнего рукопожатия (0 = никогда). Данные из `awg show all dump`.

### Статистика одного пира

```bash
curl http://localhost:9090/api/peers/my-phone/stats \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

```json
{"name": "my-phone", "ip": "10.0.0.3", "received": 257662, "sent": 2226272, "last_handshake": 1785763093}
```

Если пир не найден — 404:

```json
{"error": "peer not found"}
```

## Файлы

- `amnezia-config/awg0.conf` — серверный конфиг AmneziaWG, сюда добавляются `[Peer]` секции
- `amnezia-config/peers.json` — метаданные пиров (имя, ключи, IP, дата создания)
