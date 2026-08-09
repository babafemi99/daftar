FROM node:24-alpine AS dependencies
WORKDIR /app
COPY package.json package-lock.json ./
# npm 11 currently validates platform-specific Sharp optional dependencies
# differently across macOS lockfile generation and Alpine installs.
RUN npm install --ignore-scripts

FROM node:24-alpine AS build
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
ARG DAFTAR_API_INTERNAL_URL=http://localhost:8080
ENV DAFTAR_API_INTERNAL_URL=${DAFTAR_API_INTERNAL_URL}
COPY --from=dependencies /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:24-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000 HOSTNAME=0.0.0.0
RUN addgroup --system --gid 1001 nodejs && adduser --system --uid 1001 nextjs
COPY --from=build --chown=nextjs:nodejs /app/public ./public
COPY --from=build --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=build --chown=nextjs:nodejs /app/.next/static ./.next/static
USER nextjs
EXPOSE 3000
CMD ["node", "server.js"]
