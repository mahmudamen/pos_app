# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/pos-api ./cmd/api

FROM alpine:3.20
RUN addgroup -S pos-api && adduser -S -G pos-api pos-api
WORKDIR /app
COPY --from=build /out/pos-api /app/pos-api
USER pos-api
EXPOSE 8080
ENTRYPOINT ["/app/pos-api"]
