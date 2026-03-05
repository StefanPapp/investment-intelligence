FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server/

FROM alpine:3.21

RUN apk --no-cache add ca-certificates
RUN adduser -D appuser

WORKDIR /app
COPY --from=builder /server .
COPY migrations/ ./migrations/

USER appuser

EXPOSE 8080

CMD ["./server"]
