# syntax=docker/dockerfile:1

FROM node:22-alpine AS builder

WORKDIR /app

COPY web/package.json web/package-lock.json ./web/
WORKDIR /app/web
RUN npm ci

COPY api /app/api
COPY web /app/web

ARG VITE_API_BASE_URL=
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

RUN npm run build

FROM nginx:1.27-alpine

COPY web/nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/web/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
