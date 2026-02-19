FROM golang:1.24-bookworm
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /out/server ./cmd/gateway

USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/out/server"]
