# syntax=docker/dockerfile:1

# Stage 1: Build Go application
FROM golang:1.25.3 AS go-builder
WORKDIR /app
COPY . .
RUN go build -o /site ./

# Stage 2: Final image
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=go-builder /site ./
COPY --from=go-builder /app/static ./static
EXPOSE 8000
CMD ["./site"]
