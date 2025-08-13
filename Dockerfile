FROM golang:1.24-bookworm AS build

RUN set -eux; \
    echo 'Acquire::Retries "5";' > /etc/apt/apt.conf.d/80-retries; \
    apt-get update -o Acquire::ForceIPv4=true; \
    apt-get install -y --no-install-recommends pkg-config librdkafka-dev; \
    rm -rf /var/lib/apt/lists/*

ENV GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org \
    GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download || (sleep 2 && go mod download)

COPY . .
RUN go build -o /out/server ./cmd/server

FROM golang:1.24-bookworm AS tester

RUN set -eux; \
    echo 'Acquire::Retries "5";' > /etc/apt/apt.conf.d/80-retries; \
    apt-get update -o Acquire::ForceIPv4=true; \
    apt-get install -y --no-install-recommends pkg-config librdkafka-dev; \
    rm -rf /var/lib/apt/lists/*

ENV GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org \
    GO111MODULE=on \
    CGO_ENABLED=1

WORKDIR /build
COPY . .
FROM debian:bookworm-slim

RUN set -eux; \
    echo 'Acquire::Retries "5";' > /etc/apt/apt.conf.d/80-retries; \
    apt-get update -o Acquire::ForceIPv4=true; \
    apt-get install -y --no-install-recommends ca-certificates librdkafka1; \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /out/server ./server

EXPOSE 8080
ENTRYPOINT ["./server"]
