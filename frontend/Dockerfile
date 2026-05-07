FROM node:26-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:26-alpine
WORKDIR /app
COPY --from=builder --chown=node:node /app .
USER node
CMD ["npm", "start"]
