FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod tidy

COPY . .

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

RUN go build -o /app/main cmd/main.go

FROM debian:bullseye

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/config.yaml .

RUN chmod +x ./main

EXPOSE 8080

CMD ["./main"]
