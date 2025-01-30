FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.su[m] ./

RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/go-api ./cmd/go-api/...

FROM alpine:latest as run-stage

WORKDIR /app

COPY .env .env

COPY --from=builder /app/go-api /app/go-api

EXPOSE 8888

CMD ["/app/go-api"]