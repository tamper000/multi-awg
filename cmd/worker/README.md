# Docker Worker

HTTP API для управления AmneziaWG пирами.

## Запуск

Требуется Docker daemon и Go 1.26+.

```bash
cp .env.example .env  # настрой токен
go run ./cmd/worker/
```

Сервер стартует на `:9090`.

Сервер слушает `:9090` (порт захардкожен).

## Переменные окружения

| Переменная | По умолчанию | Описание |
|-----------|-------------|----------|
| `AUTH_TOKEN` | (пусто) | Токен авторизации. Если не задан — авторизация отключена |
| `SERVER_ENDPOINT` | (пусто) | Адрес сервера (`host:port`) для генерации клиентских конфигов |
| `MIHOMO_TEMPLATE` | (пусто) | Путь к шаблону mihomo YAML для `.../sub` |

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

### Удалить пиры (батчем)

```bash
curl -X DELETE http://localhost:9090/api/peers \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -d '{"names": ["my-phone", "laptop"]}'
```

```json
{"status": "deleted"}
```

Если ни один из указанных пиров не найден — 404:

```json
{"error": "peer not found"}
```

### Заморозить / разморозить пиры

Пир остаётся в списке, но его `[Peer]`-секция убирается из активного конфига (трафик замораживается).

```bash
curl -X POST http://localhost:9090/api/peers/freeze \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -d '{"names": ["my-phone"]}'
```

```json
{"status": "frozen", "count": "1"}
```

Разморозка — тот же запрос на `/api/peers/unfreeze`, ответ `{"status": "unfrozen", "count": "1"}`. Если ни один пир не найден — 404.

### Подписка пира (конфиг + mihomo)

Отдаёт клиентский конфиг, vpn-ссылку и mihomo YAML (из `MIHOMO_TEMPLATE`):

```bash
curl http://localhost:9090/api/peers/my-phone/sub \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

```json
{
  "conf": "[Interface]\nPrivateKey = ...",
  "vpn_link": "amneziawg://...",
  "mihomo_yaml": "mixed-port: 7890\n..."
}
```

Если пир не найден — 404, при ошибке чтения mihomo-шаблона поле `mihomo_yaml` будет пустым.

### Принудительная синхронизация

Перечитывает конфиг и применяет изменения (`awg syncconf`):

```bash
curl -X POST http://localhost:9090/api/sync \
  -H "Authorization: Bearer $AUTH_TOKEN"
```

```json
{"status": "synced"}
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
