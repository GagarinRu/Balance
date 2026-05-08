FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/gophermart ./cmd/gophermart

FROM debian:bookworm-slim AS accrual

WORKDIR /app

COPY cmd/accrual/accrual_linux_amd64 ./accrual/accrual_linux_amd64
RUN chmod +x ./accrual/accrual_linux_amd64

EXPOSE 8080

CMD ["./accrual/accrual_linux_amd64"]

FROM golang:1.23-bookworm AS integration

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o ./gophermart ./cmd/gophermart
RUN mkdir -p accrual && cp cmd/accrual/accrual_linux_amd64 ./accrual/accrual_linux_amd64 && chmod +x ./accrual/accrual_linux_amd64

FROM alpine:3.19 AS runtime

WORKDIR /app

COPY --from=builder /app/gophermart .

EXPOSE 8080

CMD ["./gophermart"]
