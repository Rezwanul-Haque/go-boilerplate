FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go install github.com/swaggo/swag/cmd/swag@v1.16.6

COPY . .
RUN swag init -g cmd/main.go -o docs/swagger --parseDependency --parseInternal
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/main.go

FROM scratch AS final

# CA certs for TLS (postgres SSL, any outbound HTTPS)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/app/infra/database/migrations ./app/infra/database/migrations

# Run as non-root (numeric UID — scratch has no /etc/passwd)
USER 10001:10001

EXPOSE 8080

CMD ["/app/server"]
