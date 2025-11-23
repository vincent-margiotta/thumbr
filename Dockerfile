# syntax=docker/dockerfile:1

# Base builder image with tools and dependencies cached.
FROM golang:1.24-alpine AS base

RUN apk add --no-cache git ca-certificates make curl && \
    update-ca-certificates

WORKDIR /app
ENV CGO_ENABLED=0 GO111MODULE=on

# Cache modules first.
COPY go.mod go.sum ./
RUN go mod download

# Development target: drop into a shell with the repo mounted.
FROM base AS dev
COPY . .
CMD ["/bin/sh"]

# Build target: produces a statically linked Linux binary.
FROM base AS build
ARG VERSION=dev
ENV GOOS=linux GOARCH=amd64
COPY . .
RUN go build -ldflags "-s -w -X main.version=${VERSION}" -o /out/thumbr ./cmd/thumbr

# Minimal runtime image.
FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates && \
    update-ca-certificates && \
    adduser -D appuser

USER appuser
COPY --from=build /out/thumbr /usr/local/bin/thumbr

ENTRYPOINT ["thumbr"]
CMD ["--help"]
