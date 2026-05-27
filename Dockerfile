# Stage 1: Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build backend (with embedded frontend)
FROM golang:1.23-alpine AS backend
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /app/backend/internal/httpserver/frontend_dist ./internal/httpserver/frontend_dist
RUN CGO_ENABLED=0 go build -o /loop .

# Stage 3: Final image
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend /loop .

EXPOSE 8080
VOLUME ["/data"]
ENV DB_PATH=/data/loop.db

ENTRYPOINT ["./loop"]
