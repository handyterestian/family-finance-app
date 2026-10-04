# ─────────────────────────────────────────────────────────────
# Stage 1: Build React app
# ─────────────────────────────────────────────────────────────
FROM node:20-alpine AS builder

WORKDIR /app

COPY client/package.json ./
RUN npm install

COPY client/ ./
RUN npm run build

# ─────────────────────────────────────────────────────────────
# Stage 2: Serve via Nginx
# ─────────────────────────────────────────────────────────────
FROM nginx:alpine

# Hapus default nginx config
RUN rm /etc/nginx/conf.d/default.conf

# Copy custom nginx config
COPY client/nginx.conf /etc/nginx/conf.d/app.conf

# Copy built React app
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
