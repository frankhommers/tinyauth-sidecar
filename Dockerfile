# frontend build
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/tsconfig.json frontend/vite.config.ts frontend/index.html ./
COPY frontend/src ./src
RUN corepack enable && corepack prepare pnpm@latest --activate && pnpm install && pnpm build

# backend build
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum* ./
COPY . .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /out/tinyauth-sidecar ./main.go

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend-builder /out/tinyauth-sidecar /app/tinyauth-sidecar
EXPOSE 8080
CMD ["/app/tinyauth-sidecar"]
