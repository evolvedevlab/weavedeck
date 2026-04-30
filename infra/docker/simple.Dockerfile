FROM golang:1.25.0-alpine AS builder

WORKDIR /app

ENV CGO_ENABLED=0
ENV GO111MODULE=on

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o ./bin/simple ./cmd/simple

# Prod stage
FROM alpine:latest

RUN apk add --no-cache \
    ca-certificates \
    curl \
    tar \
    bash \
    libc6-compat \
    libstdc++ \
    libgcc \
    shadow \
    su-exec

COPY --from=builder /app/script/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ENV HUGO_VERSION=latest
RUN set -eux; \
    ARCH=$(uname -m); \
    case "$ARCH" in \
        x86_64) ARCH=64bit ;; \
        aarch64) ARCH=ARM64 ;; \
        *) echo "Unsupported arch: $ARCH"; exit 1 ;; \
    esac; \
    \
    if [ "$HUGO_VERSION" = "latest" ]; then \
        HUGO_VERSION=$(curl -s https://api.github.com/repos/gohugoio/hugo/releases/latest | grep tag_name | cut -d '"' -f 4); \
    fi; \
    \
    curl -L -o hugo.tar.gz \
        "https://github.com/gohugoio/hugo/releases/download/${HUGO_VERSION}/hugo_extended_${HUGO_VERSION#v}_Linux-${ARCH}.tar.gz"; \
    \
    tar -xzf hugo.tar.gz; \
    mv hugo /usr/local/bin/hugo; \
    chmod +x /usr/local/bin/hugo; \
    rm -f hugo.tar.gz LICENSE README.md

# Create application directory
RUN mkdir -p /app /app/site

WORKDIR /app

ENV ENVIRONMENT="production"

# Copying the Go binary & site
COPY --from=builder /app/bin ./bin
COPY ./site ./site

EXPOSE 3000

# DO NOT set USER here - we need to run as root initially to create users and fix permissions
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["./bin/simple"]
