# producer/Dockerfile

FROM golang:1.24.1 AS build
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o producer

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=build /app/producer .
CMD ["./producer"]
