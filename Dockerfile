# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /orderservice \
    ./cmd/orderservice

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /orderservice /orderservice

EXPOSE 8080

ENV HTTP_ADDR=:8080

ENTRYPOINT ["/orderservice"]
