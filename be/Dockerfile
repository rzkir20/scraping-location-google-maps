FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/maps-scraper .

FROM debian:bookworm-slim

ENV PORT=8080

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    chromium \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/maps-scraper /app/maps-scraper

EXPOSE 8080

CMD ["sh", "-c", "/app/maps-scraper server 0.0.0.0:${PORT}"]
