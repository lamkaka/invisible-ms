FROM golang:1.26-alpine AS builder

WORKDIR /build

# Allow Go to download the required toolchain version
ENV GOTOOLCHAIN=auto

# Download dependencies first (layer caching)
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

# Copy Go source
COPY apps/api/ ./

# Build binaries
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/server
RUN CGO_ENABLED=0 go build -o /app/migrate ./cmd/migrate

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk --no-cache add ca-certificates wget

WORKDIR /app/api

# Copy binaries
COPY --from=builder /app/server /app/api/server
COPY --from=builder /app/migrate /app/api/migrate

# Copy web assets and migrations
COPY apps/web/ /app/web/
COPY deployments/migrations/ /app/api/migrations/

ENV TEMPLATES_PATH=/app/web/templates
ENV STATIC_PATH=/app/web/static
ENV PORT=8080

EXPOSE 8080

CMD ["./server"]
