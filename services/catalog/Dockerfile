FROM golang:1.24-bookworm AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/catalog ./services/catalog/cmd/catalog

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/catalog /app/catalog
EXPOSE 8080
ENTRYPOINT ["/app/catalog"]
