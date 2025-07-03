# Stage 1: build
FROM golang:1.24.1 as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main ./cmd/main.go

# Stage 2: runtime
FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /app/main .
COPY ./web /app/web
COPY internal/db/migrations internal/db/migrations
CMD ["./main"]
