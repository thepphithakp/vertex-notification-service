# --- Build ---
FROM golang:1.25.14-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-s -w" \
    -o /out/notification-service ./cmd/server

# --- Run ---
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/notification-service /app/notification-service

WORKDIR /app
USER nonroot:nonroot
EXPOSE 4003

ENTRYPOINT ["/app/notification-service"]
