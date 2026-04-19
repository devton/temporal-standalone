# Custom Temporal UI for Railway
# Builds from upstream submodule + overlays
# Railway handles submodule init via docker-compose.railway.yml (submodules: true)

# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app

# Copy upstream source (submodule initialized by Railway)
COPY ui-custom/upstream/ ./

# Apply custom overlays
COPY ui-custom/overlays/ /overlays/
RUN cp -r /overlays/* ./ || true

# Build frontend
RUN npm install -g pnpm@latest && pnpm install --frozen-lockfile
RUN pnpm build:server

# Stage 2: Build Backend (Go)
FROM golang:1.24-alpine AS server-builder
RUN apk add --no-cache make git curl
WORKDIR /home/server-builder
COPY --from=frontend-builder /app/server/ ./
RUN go mod download && make build-server

# Stage 3: Final image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata bash
WORKDIR /home/ui-server
RUN addgroup -g 1000 temporal && adduser -u 1000 -G temporal -D temporal

COPY --from=server-builder /home/server-builder/ui-server ./
COPY --from=frontend-builder /app/server/config/docker.yaml ./config/docker.yaml
COPY --from=frontend-builder /app/server/docker/start-ui-server.sh ./start-ui-server.sh

USER temporal
EXPOSE 8080
CMD ["./start-ui-server.sh"]
