# Dockerfile
FROM golang:1.24-alpine

RUN go install github.com/pressly/goose/v3/cmd/goose@latest
RUN apk add --no-cache postgresql-client

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o bot ./cmd

COPY compose/entrypoint.sh /entrypoint.sh
RUN sed -i 's/\r$//g' /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY compose/wait.sh /wait.sh
RUN sed -i 's/\r$//g' /wait.sh
RUN chmod +x /wait.sh

ENTRYPOINT ["/entrypoint.sh"]
