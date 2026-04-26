FROM golang:1.26-bookworm AS task
RUN go install github.com/go-task/task/v3/cmd/task@v3.50.0

FROM node:lts-bookworm-slim AS web
COPY --from=task /go/bin/task /usr/local/bin/task
WORKDIR /build
COPY web/app/package.json web/app/package-lock.json web/app/
RUN npm ci --prefix web/app
COPY web/app web/app
COPY taskfile.yml .
RUN task web:build

FROM golang:1.26-bookworm AS builder
COPY --from=task /go/bin/task /usr/local/bin/task
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /build/web/app/dist ./web/app/dist
RUN task build:go:container

FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /build/sharearr /app/sharearr

VOLUME /config
VOLUME /data

EXPOSE 8787

CMD ["/app/sharearr"]
