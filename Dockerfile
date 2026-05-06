# Builder stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -v -ldflags "-s -w" -o /app/fairchain-miner ./cmd/fairchain-miner

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/fairchain-miner /usr/local/bin/

WORKDIR /root

ENTRYPOINT ["/usr/local/bin/fairchain-miner"]