FROM golang:1.24-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/paperless-scanner ./cmd/paperless-scanner

FROM debian:bookworm-slim

LABEL org.opencontainers.image.title="paperless-scanner"
LABEL org.opencontainers.image.description="Scanner webhook service for Epson devices and Paperless-ngx"
LABEL org.opencontainers.image.authors="Fischris"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/Fischris/paperless-scanner"

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        sane-utils \
        sane-airscan \
        ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

ENV PORT=8080
ENV SCAN_RESOLUTION=300

COPY --from=builder /out/paperless-scanner /app/paperless-scanner

EXPOSE 8080

ENTRYPOINT ["/app/paperless-scanner"]
