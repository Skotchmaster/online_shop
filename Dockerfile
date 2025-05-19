# Stage 1: builder (Debian-based, с libc и librdkafka-dev)
FROM golang:1.24-bullseye AS builder

# Устанавливаем pkg-config и librdkafka-dev для cgo
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      pkg-config \
      librdkafka-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Копируем модули и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь код и собираем с поддержкой cgo
COPY . .
ENV CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64
RUN go build -o server ./cmd/server

# Stage 2: минимальный рантайм (Debian slim + librdkafka)
FROM debian:bullseye-slim

# Нужно чтобы бинарник умел проверять HTTPS-сертификаты, и чтобы был динамический librdkafka
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      librdkafka1 && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /build/server .

EXPOSE 8080
ENTRYPOINT ["./server"]
