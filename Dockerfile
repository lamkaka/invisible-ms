FROM golang:1.26-alpine AS builder

WORKDIR /app

# Allow Go to download the required toolchain version
ENV GOTOOLCHAIN=auto

# Download dependencies first (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build server binary
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

# Copy binary
COPY --from=builder /app/server /app/

# Copy template and static files
COPY templates/ ./templates/
COPY web/static/ ./web/static/
COPY migrations/ ./migrations/

EXPOSE 8080

CMD ["/app/server"]
