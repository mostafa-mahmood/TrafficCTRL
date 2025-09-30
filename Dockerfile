FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o trafficctrl \
    ./cmd/ctrl

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/trafficctrl /app/trafficctrl

COPY --from=builder /build/config/*.yaml /app/config/

ENV CONFIG_DIR=/app/config

RUN addgroup -g 1000 trafficctrl && \
    adduser -D -u 1000 -G trafficctrl trafficctrl && \
    chown -R trafficctrl:trafficctrl /app

USER trafficctrl

EXPOSE 8080 8090

ENTRYPOINT ["/app/trafficctrl"]