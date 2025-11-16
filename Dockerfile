FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pr-service ./cmd/app

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/pr-service /app/pr-service
COPY --from=builder /app/migrations /app/migrations

ENV HTTP_PORT=8080
ENV LOG_LEVEL=INFO
ENV MIGRATIONS_DIR=/app/migrations

EXPOSE 8080

CMD ["/app/pr-service"]
