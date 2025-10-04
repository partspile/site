# syntax=docker/dockerfile:1

# Stage 1: Build CSS
FROM node:20-alpine AS css-builder
WORKDIR /app
RUN apk add --no-cache build-base python3 make
COPY package.json package-lock.json ./
RUN npm install
COPY static/css/input.css ./static/css/
COPY tailwind.config.js ./
RUN npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/output.css --minify

# Stage 2: Build Go application
FROM golang:1.24.1 AS go-builder
WORKDIR /app
COPY . .
COPY --from=css-builder /app/static/css/output.css ./static/css/output.css
RUN go build -o /site ./

# Stage 3: Final image
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=go-builder /site ./
COPY --from=go-builder /app/static ./static
EXPOSE 8000
CMD ["./site"]
