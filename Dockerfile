FROM golang:1.25.7-alpine AS builder
RUN apk add --no-cache git
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /migrate ./cmd/migrate

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app /app
COPY --from=builder /migrate /migrate
COPY migrations/ /migrations/
EXPOSE 8080
ENTRYPOINT ["/app"]
