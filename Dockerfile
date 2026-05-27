# Stage 1: Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app
COPY frontend/ ./frontend/
RUN cd frontend && npm ci && npm run build
# Vite outputs to ../backend/internal/httpserver/frontend_dist relative to frontend/

# Stage 2: Build backend (with embedded frontend)
FROM golang:1.23-alpine AS backend
WORKDIR /app
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download
COPY backend/ ./backend/
COPY --from=frontend /app/backend/internal/httpserver/frontend_dist ./backend/internal/httpserver/frontend_dist
RUN cd backend && CGO_ENABLED=0 go build -o /loop .

# Stage 3: Final image
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend /loop .

EXPOSE 8080
VOLUME ["/data"]
ENV DB_PATH=/data/loop.db

ENTRYPOINT ["./loop"]
