FROM golang:1.25.0 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gateway ./cmd/api

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /gateway /gateway
USER app
EXPOSE 8080
ENTRYPOINT ["/gateway"]