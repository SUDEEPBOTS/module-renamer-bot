# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/module-renamer-bot cmd/bot/main.go

# Production runtime stage
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata zip unzip

COPY --from=builder /app/module-renamer-bot /app/module-renamer-bot
COPY sample.env /app/sample.env

CMD ["/app/module-renamer-bot"]
