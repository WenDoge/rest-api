# Users REST API

REST API на Go для работы с пользователями: регистрация, вход, профиль, список пользователей, обновление и мягкое удаление.

## Быстрый старт

1. создать .env по шаблону .env.example

2. docker compose up
   go run ./cmd/api-server

либо можно запускать через Task (`go install github.com/go-task/task/v3/cmd/task@latest`).

Сервер слушает `http://localhost:8080`, API доступен под префиксом `/api/v1`.

Swagger UI: http://localhost:8080/swagger/index.html

## Структура проекта

```
cmd/api-server/       точка входа: конфиг, роутинг, graceful shutdown
internal/
  auth/               генерация и разбор JWT, хэширование токенов
  config/             загрузка конфигурации из .env
  database/           подключение к Postgres (pgxpool)
  handlers/           HTTP-хендлеры Gin + интерфейсы репозиториев
  middleware/         проверка access-токена
  migrations/         SQL-миграции goose
  models/             доменные модели
  repository/         SQL-запросы к БД
docs/                 сгенерированная Swagger-документация
```

## Тесты

```bash
go test ./...
```

Покрыты генерация и валидация токенов (`internal/auth`) и HTTP-хендлеры с моками репозиториев (`internal/handlers`).

## Swagger

Документация генерируется из аннотаций над хендлерами:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/api-server/main.go
```
