# Накопительная система лояльности «Гофермарт»

![Go](https://img.shields.io/badge/Go-1.23-blue?logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-orange?logo=postgresql)
![Docker](https://img.shields.io/badge/Docker-24.0-blue?logo=docker)
![JWT](https://img.shields.io/badge/JWT-authentication-yellow?logo=jsonwebtokens)
![Chi](https://img.shields.io/badge/Chi-v5-green)

## Описание

Система представляет собой HTTP API для накопительной системы лояльности «Гофермарт». Позволяет пользователям регистрироваться, загружать номера заказов, получать начисления баллов лояльности и совершать списания баллов в счёт оплаты новых заказов.

## Основные функции

- **Регистрация пользователей** — создание аккаунта с логином и паролем.
- **Аутентификация** — вход в систему с использованием JWT-токенов.
- **Загрузка номеров заказов** — передача номеров заказов для расчёта начислений.
- **Баланс пользователя** — просмотр текущего количества баллов и суммы списанных баллов.
- **Списание баллов** — запрос на списание баллов в счёт оплаты нового заказа.
- **История выводов** — получение информации о всех списаниях.

## Технологический стек

- **Go 1.23** — основной язык программирования.
- **chi/v5** — HTTP-роутер.
- **PostgreSQL** — база данных для хранения информации о пользователях, заказах и балансах.
- **golang-jwt/jwt/v5** — аутентификация и авторизация пользователей.
- **zap** — логирование.
- **Docker** — контейнеризация приложения.

## Принцип работы

1. Пользователь регистрируется в системе лояльности «Гофермарт».
2. Пользователь совершает покупку в интернет-магазине.
3. Пользователь передаёт номер заказа в систему лояльности.
4. Система связывает номер заказа с пользователем и сверяет с системой расчёта баллов лояльности.
5. При наличии положительного расчёта баллов лояльности производится начисление на счёт пользователя.
6. Пользователь списывает доступные баллы для частичной или полной оплаты последующих заказов.

## Требования

- Docker и Docker Compose
- Go 1.23+
- PostgreSQL 13+

## Инструкция по запуску

### 1. Подготовка

Склонируйте репозиторий:
```bash
git clone https://github.com/GagarinRu/go-musthave-diploma-tpl.git
cd go-musthave-diploma-tpl
```

Создайте `.env` из шаблона `.env_example` и задайте параметры БД (и при необходимости `ACCRUAL_PORT`, `SERVER_PORT`):

```bash
cp .env_example .env
```

### 2. Запуск с использованием Docker Compose

Поднимаются PostgreSQL, сервис начислений **accrual** (внутри контейнера порт `8080`, с хоста обычно `8081`) и приложение **gophermart_app**:

```bash
docker compose up --build -d
```

### 3. Запуск без Docker

Нужны PostgreSQL и бинарник **accrual** из `cmd/accrual/` (слушает `:8080`). Пример:

```bash
go run ./cmd/gophermart -a :8080 -d "postgres://user:password@localhost:5432/gophermart" -r http://localhost:8080
```

## API Endpoints

| Метод | Эндпоинт | Описание |
|-------|----------|-----------|
| POST | /api/user/register | Регистрация пользователя |
| POST | /api/user/login | Аутентификация пользователя |
| POST | /api/user/orders | Загрузка номера заказа |
| GET | /api/user/orders | Получение списка заказов |
| GET | /api/user/balance | Получение баланса |
| POST | /api/user/balance/withdraw | Списание баллов |
| GET | /api/user/withdrawals | История выводов |

## Пример использования

### Регистрация пользователя

```bash
curl -X POST http://localhost:8080/api/user/register \
  -H "Content-Type: application/json" \
  -d '{"login": "user@example.com", "password": "password123"}'
```

### Загрузка номера заказа

```bash
curl -X POST http://localhost:8080/api/user/orders \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: text/plain" \
  -d '9278923470'
```

### Получение баланса

```bash
curl -X GET http://localhost:8080/api/user/balance \
  -H "Authorization: Bearer <token>"
```

Пример ответа:
```json
{
  "current": 500.5,
  "withdrawn": 42
}
```

## Структура проекта

- `cmd/gophermart/main.go` — точка входа приложения.
- `internal/handler/` — HTTP-обработчики.
- `internal/service/` — бизнес-логика.
- `internal/repository/postgres/` — работа с PostgreSQL.
- `internal/config/` — конфигурация приложения.
- `internal/entity/` — сущности предметной области.
- `internal/dto/` — объекты передачи данных.
- `migrations/` — миграции базы данных.
- `make.ps1` — сборка, моки, юнит-тесты и интеграция через Docker (`docker-it`).

## Тестирование

- **Юнит-тесты:** `go test ./...` или в Windows: `.\make.ps1 tests` (пакет `internal/service` с моками).
- **Интеграционные тесты** (Docker, сервис `integration_tests` в `docker-compose.yml`): `.\make.ps1 docker-it` — требуется доступный Docker и поднятая БД (`docker compose up -d gophermart_db` или полный compose).

## Авторы

Gagarin: https://github.com/GagarinRu