FROM golang:latest

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod tidy

# Install goose CLI
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

COPY . .

ENTRYPOINT ["goose"]
