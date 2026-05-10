# Electronic Digital Signature Lab

Проект уже ужат до минимального контура под новое ТЗ:

- одно и то же `Go`-приложение запускается в двух ролях: `sender` и `recipient`
- `sender` принимает HTTP-запрос с `.docx` и адресом получателя
- приложение добавляет в документ визуальные метаданные, подписывает его, формирует зашифрованный пакет и отправляет его по SMTP
- приложение умеет расшифровать и проверить пакет по публичному ключу отправителя из самого пакета
- `DB`, `JWT`, `users`, старые web use-cases и frontend выведены из рабочего контура

Фоновый listener/webhook для `recipient` будет следующим этапом уже поверх этой очищенной базы.

## Docker Compose

```bash
cp .env.example .env
docker compose up --build
```

Поднимаются:

- `sender` на `http://localhost:8081`
- `recipient` на `http://localhost:8082`
- `mailpit` UI на `http://localhost:8025`

Оба контейнера `sender` и `recipient` запускают один и тот же образ из `app/Dockerfile`, но с разными `APP_ROLE`, `APP_EMAIL` и volume для данных.

## Локальный запуск

```bash
cp .env.example .env
set -a
source .env
set +a
docker compose up -d mailpit
cd app
go run ./cmd/server
```

Ключи создаются автоматически при первом запуске, если файлов еще нет.

## Текущие endpoints

### `GET /health`

Проверка живости приложения.

### `GET /api/v1/identity`

Возвращает:

- роль приложения
- email экземпляра
- публичный ключ
- отпечаток ключа
- используемые алгоритмы

### `POST /api/v1/documents/send`

`multipart/form-data`:

- `file`: `.docx`
- `recipient_email`: адрес корреспондента

Пример:

```bash
curl -X POST http://localhost:8081/api/v1/documents/send \
  -F 'recipient_email=recipient@example.com' \
  -F 'file=@./contract.docx;type=application/vnd.openxmlformats-officedocument.wordprocessingml.document'
```

### `POST /api/v1/packages/verify-decrypt`

Принимает JSON пакета как raw body или как multipart-файл `package`.

Пример:

```bash
curl -X POST http://localhost:8082/api/v1/packages/verify-decrypt \
  -H 'Content-Type: application/json' \
  --data-binary @./package.json
```

## Что уже удалено из рабочего контура

- Postgres
- GORM repositories
- регистрация и логин пользователей
- JWT middleware
- server-signed-message use cases
- маршруты `/auth`, `/users`, старые `/documents/:id/...`
- старые route tests под user-based flow

## Что будет следующим шагом

- listener/webhook режим для `recipient`
- автоматическая обработка входящей почты
- локальный audit/inbox/outbox слой
- замена demo key transport на схему шифрования именно для получателя
