FROM golang:1.26-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -tags sqlite_fts5,container -ldflags="-w -s" -o sharearr ./cmd/sharearr

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
