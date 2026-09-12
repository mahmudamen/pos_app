# syntax=docker/dockerfile:1

FROM golang:1.23-alpine@sha256:383395b794dffa5b53012a212365d40c8e37109a626ca30d6151c8348d380b5f AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/pos-api ./cmd/api \
    && GOFLAGS=-mod=mod go install github.com/pressly/goose/v3/cmd/goose@v3.21.1

FROM alpine:3.20@sha256:d9e853e87e55526f6b2917df91a2115c36dd7c696a35be12163d44e6e2a4b6bc
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