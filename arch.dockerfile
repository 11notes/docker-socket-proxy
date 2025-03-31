# :: Util
  FROM 11notes/util AS util

# :: Build / socket-proxy
  FROM golang:1.24-alpine AS build
  ARG TARGETARCH
  ENV CGO_ENABLED=0
  ENV BUILD_DIR=/go/socket-proxy
  ENV BUILD_BIN=${BUILD_DIR}/socket-proxy

  USER root
  COPY --from=util /usr/local/bin/ /usr/local/bin
  COPY ./go/ /go

  RUN set -ex; \
    apk --update --no-cache add \
      build-base \
      upx;

  RUN set -ex; \
    cd ${BUILD_DIR}; \
    mkdir -p /distroless/usr/local/bin; \
    go build -ldflags="-extldflags=-static" -o ${BUILD_BIN} main.go; \
    eleven strip ${BUILD_BIN}; \
    cp ${BUILD_BIN} /distroless/usr/local/bin;

# :: Distroless / socket-proxy
  FROM scratch AS distroless-socket-proxy
  COPY --from=build /distroless/ /

# :: Build / file system
  FROM alpine AS fs
  ARG APP_ROOT
  USER root

  RUN set -ex; \
    mkdir -p ${APP_ROOT}/etc;

  COPY ./rootfs /

# :: Distroless / file system
  FROM scratch AS distroless-fs
  ARG APP_ROOT
  COPY --from=fs ${APP_ROOT} /${APP_ROOT}

# :: Header
  FROM 11notes/distroless AS distroless
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
    COPY --from=distroless-fs / /
    COPY --from=distroless-socket-proxy / /

# :: Monitor
  HEALTHCHECK --interval=5s --timeout=2s CMD ["socket-proxy", "--healthcheck"]

# :: Start
  ENTRYPOINT ["socket-proxy"]