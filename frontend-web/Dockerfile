# Multi-stage: build the static SPA with Node, serve it with nginx.
# Build context is this directory:
#   docker build -t trading/frontend frontend-web
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install --no-audit --no-fund
COPY . .
# The browser (on the host) reaches the gateway at localhost:8090, so bake that
# in by default; override with --build-arg for other deployments.
ARG VITE_API_BASE=http://localhost:8090
ARG VITE_WS_BASE=ws://localhost:8090
ENV VITE_API_BASE=$VITE_API_BASE VITE_WS_BASE=$VITE_WS_BASE
RUN npm run build

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
