FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /ticketflow-api ./cmd

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /ticketflow-api /usr/local/bin/ticketflow-api

RUN mkdir -p /app/uploads

EXPOSE 8080

CMD ["ticketflow-api"]
