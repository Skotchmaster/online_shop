# Online Shop

Учебный backend интернет-магазина на Go в формате микросервисов.

Проект показывает практический подход: отдельные сервисы по доменам, единая точка входа через gateway, отдельные БД и миграции, единый запуск через Docker Compose.

## Содержание

1. [Что реализовано](#что-реализовано)
2. [Архитектура](#архитектура)
3. [Структура репозитория](#структура-репозитория)
4. [Поток запроса](#поток-запроса)
5. [Запуск проекта](#запуск-проекта)
6. [Конфигурация .env](#конфигурация-env)
7. [API (через gateway)](#api-через-gateway)
8. [Безопасность](#безопасность)
9. [Почему выдерживает большую нагрузку](#почему-выдерживает-большую-нагрузку)
10. [TODO (план на будущее)](#todo-план-на-будущее)

## Что реализовано

- аутентификация и сессии через JWT cookies (`accessToken` + `refreshToken`);
- регистрация, login, refresh, logout;
- роли пользователей (`user`, `admin`) и проверка прав доступа;
- каталог товаров: список, карточка, поиск, admin CRUD;
- корзина: добавить товар, удалить одну позицию, очистить полностью;
- заказы: создание, просмотр, отмена, смена статуса (admin);
- CSRF middleware для mutating-запросов;
- единый docker-compose для локального запуска всех сервисов.

## Архитектура

Сервисы:

- `gateway` - единая точка входа (`http://localhost:8080`), reverse proxy на внутренние сервисы;
- `services/auth` - регистрация, login, refresh/logout, роли, хранение refresh-токенов;
- `services/catalog` - товары и поиск;
- `services/cart` - корзина пользователя;
- `services/order` - заказы и переходы статусов;
- `pkg` - общий код для всех сервисов (middleware, jwt, config, db, logging, util, auth client).

## Структура репозитория

```text
.
├── docker-compose.yml                        # оркестрация всех сервисов, БД и миграций
├── go.work                                   # связывает Go-модули в одном workspace
├── .env.example                              # шаблон переменных окружения
├── gateway/                                  # входная точка API и маршрутизация в сервисы
│   ├── cmd/
│   │   └── gateway/
│   │       └── main.go                       # bootstrap gateway (server, config, middleware)
│   ├── internal/
│   │   ├── config/                           # чтение env и валидация обязательных настроек
│   │   ├── httpserver/                       # proxy и регистрация внешних маршрутов
│   │   └── middleware/                       # JWT/secure headers/logging/recover
│   ├── Dockerfile                            # образ gateway
│   └── go.mod                                # модуль gateway
├── services/                                 # доменные микросервисы
│   ├── auth/                                 # users, login, refresh, logout, роли
│   │   ├── cmd/auth/main.go
│   │   ├── db/migrations/                    # SQL схема auth сервиса
│   │   ├── internal/
│   │   │   ├── config/
│   │   │   ├── httpserver/                   # auth handlers и роутинг
│   │   │   ├── middleware/                   # service-level auth middleware
│   │   │   ├── models/                       # модели пользователей и токенов
│   │   │   ├── repo/                         # доступ к auth БД
│   │   │   ├── service/                      # бизнес-логика auth
│   │   │   └── transport/                    # request/response DTO
│   │   ├── Dockerfile                        # образ auth
│   │   └── go.mod                            # модуль auth
│   ├── cart/                                 # корзина пользователя
│   │   ├── cmd/cart/main.go
│   │   ├── db/migrations/                    # SQL схема cart сервиса
│   │   ├── internal/
│   │   │   ├── config/
│   │   │   ├── httpserver/                   # cart handlers и роутинг
│   │   │   ├── models/                       # модели корзины
│   │   │   ├── repo/                         # доступ к cart БД
│   │   │   ├── service/                      # бизнес-логика cart
│   │   │   └── transport/                    # request/response DTO
│   │   ├── Dockerfile                        # образ cart
│   │   └── go.mod                            # модуль cart
│   ├── catalog/                              # каталог товаров и поиск
│   │   ├── cmd/catalog/main.go
│   │   ├── db/migrations/                    # SQL схема catalog сервиса
│   │   ├── internal/
│   │   │   ├── config/
│   │   │   ├── httpserver/                   # catalog handlers и роутинг
│   │   │   ├── models/                       # модели товаров
│   │   │   ├── repo/                         # доступ к catalog БД
│   │   │   ├── service/                      # бизнес-логика catalog
│   │   │   └── transport/                    # request/response DTO
│   │   ├── Dockerfile                        # образ catalog
│   │   └── go.mod                            # модуль catalog
│   └── order/                                # заказы и переходы статусов
│       ├── cmd/order/main.go
│       ├── db/migrations/                    # SQL схема order сервиса
│       ├── internal/
│       │   ├── config/
│       │   ├── httpserver/                   # order handlers и роутинг
│       │   ├── models/                       # модели заказов и items
│       │   ├── repo/                         # доступ к order БД
│       │   ├── service/                      # бизнес-логика order
│       │   └── transport/                    # request/response DTO
│       ├── Dockerfile                        # образ order
│       └── go.mod                            # модуль order
└── pkg/                                      # общий переиспользуемый код
    ├── authclient/                           # HTTP клиент к auth (refresh/validation)
    ├── config/                               # общие env/config helper-функции
    ├── db/                                   # открытие и настройка подключения к БД
    ├── hash/                                 # хеширование паролей
    ├── jwt/                                  # cookie helpers и JWT utility
    ├── logging/                              # инициализация slog логера
    ├── middleware/                           # общие middleware
    │   ├── auth/                             # auto-refresh middleware
    │   ├── csrf/                             # CSRF защита
    │   └── logging/                          # request logging middleware
    ├── tokens/                               # типы claims и парсинг JWT
    └── util/                                 # вспомогательные функции
```

## Поток запроса

1. Клиент отправляет запрос в `gateway`.
2. Gateway прогоняет common middleware (recover/request-id/logger/secure headers).
3. Для защищенных путей gateway проверяет `accessToken` cookie и достает claims (user_id, role).
4. Для mutating запросов CSRF middleware проверяет `Origin` + `X-CSRF-Token`.
5. Gateway проксирует запрос в нужный сервис (`auth` / `catalog` / `cart` / `order`).
6. Сервис выполняет бизнес-логику и обращается только к своей БД.
7. Если в auth нужен refresh, сервис/ middleware обновляет токены и возвращает новые cookies.
8. Ответ возвращается клиенту через gateway с единым внешним API.

## Запуск проекта

1) Скопируй env:

```bash
cp .env.example .env
```

2) Подними окружение:

```bash
docker compose up --build -d
```

3) Проверь состояние:

```bash
docker compose ps
docker compose logs -f gateway
```

4) Остановить:

```bash
docker compose down
```

## Конфигурация `.env`

Проект использует один корневой `.env`.

```env
JWT_SECRET=change_me_access_secret                                                           # секрет подписи access токенов
REFRESH_SECRET=change_me_refresh_secret                                                      # секрет подписи refresh токенов

DB_USER=postgres                                                                             # пользователь PostgreSQL
DB_PASSWORD=postgres                                                                         # пароль PostgreSQL (секрет)

AUTH_DB_NAME=auth_db                                                                         # имя БД сервиса auth
CART_DB_NAME=cart_db                                                                         # имя БД сервиса cart
CATALOG_DB_NAME=catalog_db                                                                   # имя БД сервиса catalog
ORDER_DB_NAME=order_db                                                                       # имя БД сервиса order

AUTH_DATABASE_URL=postgres://postgres:postgres@auth-db:5432/auth_db?sslmode=disable          # подключение auth -> auth-db
CART_DATABASE_URL=postgres://postgres:postgres@cart-db:5432/cart_db?sslmode=disable          # подключение cart -> cart-db
CATALOG_DATABASE_URL=postgres://postgres:postgres@catalog-db:5432/catalog_db?sslmode=disable # подключение catalog -> catalog-db
ORDER_DATABASE_URL=postgres://postgres:postgres@order-db:5432/order_db?sslmode=disable       # подключение order -> order-db

AUTH_BIND_ADDR=:8080                                                                         # адрес запуска auth HTTP сервера
AUTH_INTERNAL_URL=http://auth:8080                                                           # внутренний URL auth для сервисов
CATALOG_INTERNAL_URL=http://catalog:8080                                                     # внутренний URL catalog для gateway
CART_INTERNAL_URL=http://cart:8080                                                           # внутренний URL cart для gateway
ORDER_INTERNAL_URL=http://order:8080                                                         # внутренний URL order для gateway
GATEWAY_ADDR=:8080                                                                           # адрес запуска gateway
```

## API (через gateway)

База:

```text
http://localhost:8080
```

Auth:

- `POST /api/v1/auth/register` - регистрирует нового пользователя.
- `POST /api/v1/auth/login` - выдает `accessToken` и `refreshToken` cookies.
- `POST /api/v1/auth/refresh` - обновляет пару токенов по refresh cookie.
- `POST /api/v1/auth/logout` - очищает auth cookies и завершает сессию.

Catalog:

- `GET /api/v1/catalog/products` - возвращает список товаров с пагинацией.
- `GET /api/v1/catalog/products/:id` - возвращает карточку товара по id.
- `GET /api/v1/catalog/products/search?q=...&page=1&size=10` - ищет товары по текстовому запросу.
- `POST /api/v1/catalog/products` (admin) - создает новый товар.
- `PATCH /api/v1/catalog/products/:id` (admin) - обновляет поля товара.
- `DELETE /api/v1/catalog/products/:id` (admin) - удаляет товар.

Cart:

- `GET /api/v1/cart` - возвращает текущую корзину пользователя.
- `POST /api/v1/cart` - добавляет товар в корзину.
- `DELETE /api/v1/cart/items` - удаляет одну позицию из корзины.
- `DELETE /api/v1/cart` - очищает корзину полностью.

Orders:

- `GET /api/v1/orders` - список заказов текущего пользователя.
- `GET /api/v1/orders/:id` - детали конкретного заказа.
- `POST /api/v1/orders` - создает заказ.
- `POST /api/v1/orders/:id/cancel` - отменяет заказ пользователя.
- `PATCH /api/v1/orders/:id` (admin) - меняет статус заказа.

Health:

- `GET /health/live` - liveness check.
- `GET /health/ready` - readiness check.

## Безопасность

В проекте используется несколько слоев защиты.

JWT и роли:

- access token хранится в `accessToken` cookie;
- refresh token хранится в `refreshToken` cookie;
- gateway проверяет access token на защищенных маршрутах;
- роль из claims (`user`/`admin`) используется для ограничения admin-операций.

Middleware:

- при login пользователь получает пару токенов;
- при `POST /api/v1/auth/refresh` auth сервис выдает новую пару токенов;
- refresh токены ротируются и хранятся в БД в хешированном виде;
- в проекте есть auto-refresh middleware: если access token истек, middleware пытается обновить токены по refresh и продолжить запрос без повторного логина.

CSRF:

- для mutating-запросов (`POST/PUT/PATCH/DELETE`) проверяется CSRF токен;
- токен передается через cookie/header (`XSRF-TOKEN` + `X-CSRF-Token`);
- включена проверка same-origin;
- для технических/публичных путей используется `SkipPaths`.

## Производительность

В проекте есть технические решения для стабильной работы под ростом трафика.

Декомпозиция и изоляция:

- backend разделен на отдельные сервисы (`auth`, `catalog`, `cart`, `order`) с собственными БД;
- нагрузка по доменам изолируется: тяжелые запросы поиска в `catalog` меньше влияют на `auth` и `cart`;

Пулы соединений и эффективная работа с БД:

- для БД настроен connection pool (`MaxOpenConns`, `MaxIdleConns`);
- в GORM включен `PrepareStmt`, что снижает накладные расходы на повторяющиеся SQL-запросы;
- при старте есть `PingContext` с таймаутом, чтобы сервис не принимал трафик с неготовой БД.

Оптимизация запросов и ограничение тяжелых операций:

- пагинация и лимиты (`size` ограничен до 100) не дают одному запросу читать слишком много данных;
- в миграциях добавлены индексы под частые сценарии (`cart_items`, `orders`, `order_items`, `refresh_tokens`);
- для каталога есть GIN/TRGM/FTS индексы и `tsvector`-триггер для быстрого полнотекстового поиска.

Устойчивость сети и HTTP-слоя:

- в gateway и сервисах настроены `ReadTimeout`, `WriteTimeout`, `ReadHeaderTimeout`;
- в reverse proxy и auth HTTP-клиенте включены keep-alive и пулы idle-коннектов;
- это снижает риск зависаний медленных соединений и уменьшает overhead на частых внутренних вызовах.

Доступность в Docker-окружении:

- у PostgreSQL-сервисов включены healthcheck'и;
- миграции стартуют только после готовности БД;
- для основных сервисов включен `restart: unless-stopped`, что улучшает восстановление после сбоев.

## TODO (план на будущее)

- [ ] Добавить внутренние gRPC связи: `order -> catalog` для `POST /api/v1/orders`, `POST /api/v1/orders/:id/cancel`, `PATCH /api/v1/orders/:id`.
- [ ] Добавить внутренние gRPC связи: `cart -> catalog` для `POST /api/v1/cart`.
- [ ] Опционально перейти на gRPC для auth-интеграций: `gateway/order/cart/catalog -> auth`.
- [ ] Добавить unit/integration/e2e тесты для ключевых бизнес-сценариев.
- [ ] Добавить метрики и трассировку для наблюдаемости.
