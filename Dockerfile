# syntax=docker/dockerfile:1

# Debian GNU/Linux 12 (bookworm)
FROM golang:1.24.1

WORKDIR /app
COPY . .

RUN go build -o /site ./

EXPOSE 8000

CMD ["/site"]
