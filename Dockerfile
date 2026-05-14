FROM node:20-alpine AS base
RUN apk add --no-cache dumb-init

FROM node:20-alpine AS deps
RUN apk add --no-cache python3 make g++
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production && npm cache clean --force

FROM node:20-alpine AS builder
RUN apk add --no-cache python3 make g++
WORKDIR /app
COPY package*.json ./
RUN npm ci && npm cache clean --force
COPY . .
RUN npm run build --if-present || true

FROM node:20-alpine AS runner
ENV NODE_ENV=production
RUN apk add --no-cache dumb-init curl

WORKDIR /app

RUN addgroup -g 1001 -S nodejs && \
    adduser -S nodejs -u 1001

COPY --from=deps --chown=nodejs:nodejs /app/node_modules ./node_modules
COPY --chown=nodejs:nodejs package*.json ./
COPY --chown=nodejs:nodejs src ./src
COPY --chown=nodejs:nodejs config ./config
COPY --chown=nodejs:nodejs migrations ./migrations
COPY --chown=nodejs:nodejs scripts ./scripts

RUN mkdir -p logs && chown -R nodejs:nodejs /app

USER nodejs

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
  CMD curl -f http://localhost:3000/api/v1/health || exit 1

ENV PORT=3000
ENV NODE_ENV=production

ENTRYPOINT ["dumb-init", "--"]
CMD ["node", "src/index.js"]
