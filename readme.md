# Online Shop

Тестовый проект интернет-магазина на Go с использованием JWT, Docker, Kafka, PostgreSQL и нейросети для интеллектуального подбора товаров. Цель проекта — продемонстрировать настройку полного бэкенд-стека, включая аутентификацию, работу с БД, ML-инструменты и асинхронные события.

---

---

## Содержание

1. [Описание проекта](#описание-проекта)
2. [Стек технологий](#стек-технологий)
3. [Структура репозитория](#структура-репозитория)
4. [Установка и запуск](#установка-и-запуск)
5. [Настройка окружения](#настройка-окружения)
6. [Docker и Docker Compose](#docker-и-docker-compose)
7. [API Endpoints](#api-endpoints)
8. [Аутентификация и JWT](#аутентификация-и-jwt)
9. [Kafka-события](#kafka-события)
10. [Работа с базой данных](#работа-с-базой-данных)

---

## Описание проекта

Проект представляет собой простую демонстрацию бэкенд-сервиса для интернет-магазина. Включает в себя:

* Регистрацию и аутентификацию пользователей
* Выдачу и проверку JWT (access и refresh токены с версионированием ключей)
* CRUD-операции над товарами (только для админов)
* Управление корзиной пользователя
* Нейросеть для интеллектуального подбора товаров на основе предпочтений пользователя
* Асинхронную отправку событий в Kafka
* Хранение данных в PostgreSQL через GORM
* Запуск в Docker-контейнере и оркестрацию через Docker Compose

## Стек технологий

* Язык: Go 1.24
* Веб-фреймворк: [Echo](https://echo.labstack.com/)
* ORM: [GORM](https://gorm.io/) + PostgreSQL
* Аутентификация: JWT (HS256) с ключевым хранилищем для роутинга версий ключей (`kid`)
* Сообщения: Apache Kafka (Confluent Platform)
* Контейнеризация: Docker + Docker Compose
* Логирование: встроенный логгер Echo
* Конфигурация: переменные окружения

## Структура репозитория

```
├── cmd/server            # Точка входа приложения
├── internal
│   ├── handlers         # HTTP handlers (Auth, Product, Cart)
│   ├── jwtmiddleware    # Middleware для JWT с KeyFunc
│   ├── models           # Модели GORM (User, Product, CartItem, RefreshToken)
│   ├── mykafka          # Обёртка для Kafka-производителя
│   └── hash             # Утилиты для хеширования паролей
│   └── neuronet         # Нейросетевые алгоритмы подбора товаров
├── Dockerfile           # Мультистадийная сборка Go-приложения
├── docker-compose.yml   # Сборка и запуск всех сервисов (DB, Kafka, Zookeeper, App)
├── go.mod
├── go.sum
└── README.md            # Этот файл
```

## Установка и запуск

1. **Клонируйте репозиторий**

   ```bash
   git clone https://github.com/Skotchmaster/online_shop.git
   cd online_shop

2. **Настройте переменные окружения** (пример в `docker-compose.yml`)

3. **Запустите через Docker Compose**

   ```bash
   docker-compose up --build
   ```

   После запуска:

   * Приложение доступно на `http://localhost:8080`
   * PostgreSQL — на `localhost:5432`
   * Kafka — на `localhost:9092`

## Настройка окружения

Переменные окружения, необходимые для работы:

```env
# PostgreSQL
DB_HOST=db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=root
DB_NAME=online_shop

# Kafka
KAFKA_ADDRESS=kafka:9092

# JWT секреты (версия 1)
ACCESS_SECRET_V1=yourAccessSecretV1
REFRESH_SECRET_V1=yourRefreshSecretV1

```
## Docker и Docker Compose

В `docker-compose.yml` описаны следующие сервисы:

* **zookeeper** (Confluent CP)
* **kafka** (Confluent CP)
* **db** (PostgreSQL)
* **app** (ваше Go-приложение)

Образ приложения собирается из `Dockerfile`

## API Endpoints

### Публичные

* **POST /register** — регистрация пользователя
* **POST /login** — логин и выдача `access_token` + `refresh_token`

### Защищённые (JWT Middleware)

#### Товары (только роль `admin`)

* **POST /product** — создать товар
* **PATCH /product/:id** — обновить товар
* **DELETE /product/:id** — удалить товар

#### Корзина (роль `user` или `admin`)

* **GET /cart** — получить список товаров в корзине
* **POST /cart** — добавить товар в корзину (body: `{ProductID, Quantity}`)
* **DELETE /cart/:id** — уменьшить количество или удалить товар
* **DELETE /cart/:id?all=true** — удалить все единицы товара из корзины

## Аутентификация и JWT

* **Access Token** (HS256, 15 мин по умолчанию)

  * Выдается при `/login`
  * Содержит `sub` (UserID), `role`, `exp`, `kid`
* **Refresh Token** (HS256, 7 дней)

  * Содержит `sub`, `exp`, `typ = refresh`
  * Хранится в БД для возможности отзыва

### Key Rotation

* В заголовке JWT поле `kid` указывает версию ключа (например, `v1`)
* В `jwtmiddleware` используется `KeyFunc`, которое по `kid` выбирает нужный секрет из `map[string][]byte`
* Позволяет безопасно ротировать секреты без слома выданных токенов

## Kafka-события

При ключевых операциях отправляются события в топик:

* `user_registrated` — после успешной регистрации
* `user_loged_in` — после логина
* `product_events` — создание, обновление, удаление товаров
* `cart_events` — получение, добавление, удаление в корзине

Формат события — JSON с полями `type`, `UserID`, дополнительными данными.

## Работа с базой данных

* Инициализация через `handlers.InitDB()`
* Модели GORM в `internal/models`
* Автоматическая миграция таблиц при старте
