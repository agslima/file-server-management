FROM node:25-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:25-alpine
WORKDIR /app
COPY --from=builder /app .
CMD ["npm", "start"]