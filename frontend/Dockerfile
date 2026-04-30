# syntax=docker/dockerfile:1.7

# ── builder ───────────────────────────────────────────────────────────────────
FROM node:20-alpine AS builder

WORKDIR /app

# Install deps from a clean lockfile so the image is reproducible.
COPY package.json package-lock.json ./
RUN npm ci

# Build the SPA. The default empty string makes the React client use
# same-origin relative URLs (/api/*, /ws), which the nginx layer in this
# image proxies to the backend container. Override via
# `--build-arg VITE_API_BASE_URL=http://localhost:8080` if you ever want
# the bundle to talk to the backend on a different origin.
ARG VITE_API_BASE_URL=
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

COPY . .
RUN npm run build

# ── runtime ───────────────────────────────────────────────────────────────────
FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1/ >/dev/null || exit 1
