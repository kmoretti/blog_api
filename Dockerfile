# syntax=docker/dockerfile:1

FROM node:24-bookworm-slim AS frontend
WORKDIR /src/web
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.25-bookworm AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY src/ ./src/
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/blog_api .

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install --yes --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=backend /out/blog_api ./blog_api
COPY --from=frontend /src/data/panel ./data/panel
COPY migrations/ ./migrations/
EXPOSE 10024
VOLUME ["/app/data"]
ENTRYPOINT ["./blog_api"]
