# syntax=docker/dockerfile:1.6

FROM node:24-alpine AS frontend-builder

WORKDIR /src/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

FROM golang:1.25.6-alpine AS go-builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/oidc-server ./cmd/server

FROM alpine:3.21 AS runner

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=go-builder /out/oidc-server /app/oidc-server
COPY --from=go-builder /src/configs /app/configs
COPY --from=go-builder /src/db/migrations /app/db/migrations
COPY --from=frontend-builder /src/frontend/dist /app/frontend/dist

USER app

EXPOSE 8080

ENTRYPOINT ["/app/oidc-server"]
