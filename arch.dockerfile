# :: Util
  FROM 11notes/util AS util

# :: Build
  FROM golang:1.24-alpine AS build
  COPY ./go/ /go
  RUN set -ex; \
    cd /go/socket-proxy; \
    go build -ldflags="-extldflags=-static" -o socket-proxy main.go; \
    mv socket-proxy /usr/local/bin/socket-proxy;

# :: Header
  FROM 11notes/alpine:stable

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

    ENV SOCKET_PROXY="${APP_ROOT}/run/docker.sock"
    ENV SOCKET_PROXY_DOCKER_SOCKET="/run/docker.sock"

  # :: multi-stage
    COPY --from=util /usr/local/bin/ /usr/local/bin
    COPY --from=build /usr/local/bin/ /usr/local/bin

# :: Run
  USER root
  RUN eleven printenv;

  # :: install application
    RUN set -ex; \
      eleven mkdir ${APP_ROOT}/{etc,run};

  # :: copy filesystem changes and set correct permissions
    COPY ./rootfs /
    RUN set -ex; \
      chmod +x -R /usr/local/bin; \
      chown -R 1000:1000 \
        ${APP_ROOT}

# :: Monitor
  HEALTHCHECK --interval=5s --timeout=2s CMD curl --unix-socket ${SOCKET_PROXY} http://localhost/version || exit 1

# :: Start
  USER root