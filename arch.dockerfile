# :: Distroless
  FROM alpine AS fs
  USER root
  RUN set -ex; \
    mkdir -p /rootfs/run/proxy; \
    mkdir -p /rootfs/etc; \
    echo "root:x:0:0:root:/root:/bin/sh" > /rootfs/etc/passwd; \
    echo "root:x:0:root" > /rootfs/etc/group;

# :: Build // socket-proxy
  FROM golang:1.24-alpine AS socket-proxy
  ARG TARGETARCH
  USER root
  COPY ./go/ /go
  RUN set -ex; \
    cd /go/socket-proxy; \
    go build -ldflags="-extldflags=-static" -o socket-proxy main.go; \
    mv socket-proxy /usr/local/bin/socket-proxy;

# :: Header
  FROM scratch

  # :: arguments
    ARG TARGETARCH
    ARG APP_IMAGE
    ARG APP_NAME
    ARG APP_VERSION
    ARG APP_ROOT
    ARG APP_UID
    ARG APP_GID

  # :: environment
    ENV APP_IMAGE=${APP_IMAGE}
    ENV APP_NAME=${APP_NAME}
    ENV APP_VERSION=${APP_VERSION}
    ENV APP_ROOT=${APP_ROOT}

    ENV SOCKET_PROXY_VOLUME="/run/proxy"
    ENV SOCKET_PROXY_DOCKER_SOCKET="/run/docker.sock"
    ENV SOCKET_PROXY_UID=1000
    ENV SOCKET_PROXY_GID=1000

  # :: multi-stage
    COPY --from=fs /rootfs/ /
    COPY --from=socket-proxy /usr/local/bin/socket-proxy /

# :: Monitor
  HEALTHCHECK --interval=5s --timeout=2s CMD ["/socket-proxy", "--healthcheck"]

# :: Start
  USER root
  ENTRYPOINT ["/socket-proxy"]