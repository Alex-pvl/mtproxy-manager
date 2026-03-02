FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
ARG VITE_TG_CLIENT_ID
ENV VITE_TG_CLIENT_ID=$VITE_TG_CLIENT_ID
RUN npm run build

FROM golang:1.25-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=1 go build -o mtproxy-manager ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend /app/mtproxy-manager .
COPY --from=backend /app/frontend/dist ./frontend/dist
EXPOSE 8080
CMD ["./mtproxy-manager"]
