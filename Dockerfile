FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /teledrive ./cmd/teledrive

FROM alpine:3.22

WORKDIR /app

RUN mkdir -p /data

COPY --from=builder /teledrive /usr/local/bin/teledrive

EXPOSE 8080

ENTRYPOINT ["teledrive","server"]
