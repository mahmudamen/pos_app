# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/pos-api ./cmd/api \
    && GOFLAGS=-mod=mod go install github.com/pressly/goose/v3/cmd/goose@v3.21.1

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S pos-api && adduser -S -G pos-api pos-api
WORKDIR /app
COPY --from=build /out/pos-api /app/pos-api
COPY --from=build /go/bin/goose /usr/local/bin/goose
COPY internal/infrastructure/database/migrations /app/migrations
USER pos-api
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q --spider http://127.0.0.1:8080/health/live || exit 1
ENTRYPOINT ["/app/pos-api"]