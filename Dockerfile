# syntax=docker/dockerfile:1
FROM node:24-bookworm-slim AS frontend
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26.4-bookworm AS backend
WORKDIR /src
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    pkg-config \
    libgtk-3-dev \
    libayatana-appindicator3-dev \
    && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/web/dist ./web/dist
RUN CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o /out/naviserver-server ./cmd/server

FROM debian:bookworm-slim AS runtime
ARG UID=10001
ARG GID=10001
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    gzip \
    tar \
    unzip \
    libgtk-3-0 \
    libayatana-appindicator3-1 \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid ${GID} naviserver \
    && useradd --uid ${UID} --gid ${GID} --create-home --shell /usr/sbin/nologin naviserver \
    && mkdir -p /data && chown -R ${UID}:${GID} /data
WORKDIR /app
COPY --from=backend /out/naviserver-server /app/naviserver-server
COPY --from=frontend /src/web/dist /app/web_dist
RUN chown -R naviserver:naviserver /app
ENV HOME=/data XDG_CONFIG_HOME=/data NAVISERVER_HOST=0.0.0.0 NAVISERVER_PORT=23008
VOLUME ["/data"]
EXPOSE 23008
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD curl --fail --silent http://127.0.0.1:23008/health || exit 1
USER naviserver
ENTRYPOINT ["/app/naviserver-server", "--headless"]
