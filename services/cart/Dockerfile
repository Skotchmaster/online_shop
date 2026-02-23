FROM golang:1.24-bookworm AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/cart ./services/cart/cmd/cart

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/cart /app/cart
EXPOSE 8080
ENTRYPOINT ["/app/cart"]
