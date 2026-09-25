FROM node:22-alpine AS frontend
WORKDIR /build/web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web ./
RUN npm run build

FROM golang:1.26.1-alpine AS builder
ENV GO111MODULE=on CGO_ENABLED=0
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
COPY --from=frontend /build/web/dist ./web/dist
RUN go build -ldflags "-s -w" -o operation-admin .

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata wget \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates
COPY --from=builder /build/operation-admin /operation-admin
EXPOSE 3100
WORKDIR /data
ENTRYPOINT ["/operation-admin"]
