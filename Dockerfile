FROM node:24-bookworm-slim AS frontend

WORKDIR /app/web

RUN corepack enable && corepack prepare pnpm@10.30.3 --activate

COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

COPY web/src ./src
COPY web/public ./public
COPY web/index.html web/env.d.ts web/tsconfig.json web/tsconfig.node.json web/vite.config.ts ./
RUN pnpm run build && mv ../data/panel ./dist

FROM golang:1.25.4-bookworm AS backend

WORKDIR /src

RUN apt-get update \
    && apt-get install -y --no-install-recommends gcc libc6-dev \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/blog-api ./main.go

FROM debian:bookworm-slim AS runtime

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl libc6 libgcc-s1 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=backend /out/blog-api /app/blog-api
COPY --from=backend /src/migrations /app/migrations
COPY --from=frontend /app/web/dist /app/panel

ENV PANEL_ROOT=/app/panel
ENV PORT=10024
ENV LISTEN_ADDRESS=0.0.0.0

EXPOSE 10024

ENTRYPOINT ["/app/blog-api"]
