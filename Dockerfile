# frontend build
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/tsconfig.json frontend/vite.config.ts frontend/index.html ./
COPY frontend/src ./src
RUN corepack enable && corepack prepare pnpm@latest --activate && pnpm install && pnpm build

# backend build
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app
RUN apk add --no-cache git build-base
COPY go.mod ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/tinyauth-usermanagement ./main.go

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend-builder /out/tinyauth-usermanagement /app/tinyauth-usermanagement
EXPOSE 8080
CMD ["/app/tinyauth-usermanagement"]
