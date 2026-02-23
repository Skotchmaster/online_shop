FROM golang:1.24-bookworm AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/auth ./services/auth/cmd/auth

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/auth /app/auth
EXPOSE 8080
ENTRYPOINT ["/app/auth"]