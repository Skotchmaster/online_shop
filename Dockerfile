FROM golang:1.24-bullseye AS builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      pkg-config \
      librdkafka-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64
RUN go build -o server ./cmd/server

FROM debian:bullseye-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      librdkafka1 && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /build/server .

EXPOSE 8080
ENTRYPOINT ["./server"]
