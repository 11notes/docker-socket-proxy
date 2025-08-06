# ╔═════════════════════════════════════════════════════╗
# ║                       SETUP                         ║
# ╚═════════════════════════════════════════════════════╝
  # GLOBAL
  ARG APP_UID=1000 \
      APP_GID=1000 \
      BUILD_DIR=/go/socket-proxy
  ARG BUILD_BIN=${BUILD_DIR}/socket-proxy

  # :: FOREIGN IMAGES
  FROM 11notes/distroless AS distroless


# ╔═════════════════════════════════════════════════════╗
# ║                       BUILD                         ║
# ╚═════════════════════════════════════════════════════╝
  # :: SOCKET-PROXY
  FROM 11notes/go:1.24 AS build
  ARG APP_VERSION \
      BUILD_DIR \
      BUILD_BIN

  COPY ./go/ /go

  RUN set -ex; \
    cd ${BUILD_DIR}; \
    eleven go build ${BUILD_BIN} main.go; \
    eleven distroless ${BUILD_BIN};

# ╔═════════════════════════════════════════════════════╗
# ║                       IMAGE                         ║
# ╚═════════════════════════════════════════════════════╝
  # :: HEADER
  FROM scratch

  # :: default arguments
    ARG TARGETPLATFORM \
        TARGETOS \
        TARGETARCH \
        TARGETVARIANT \
        APP_IMAGE \
        APP_NAME \
        APP_VERSION \
        APP_ROOT \
        APP_UID \
        APP_GID \
        APP_NO_CACHE

  # :: default environment
    ENV APP_IMAGE=${APP_IMAGE} \
        APP_NAME=${APP_NAME} \
        APP_VERSION=${APP_VERSION} \
        APP_ROOT=${APP_ROOT}

  # :: application specific environment
    ENV SOCKET_PROXY_VOLUME="/run/proxy" \
        SOCKET_PROXY_DOCKER_SOCKET="/run/docker.sock" \
        SOCKET_PROXY_UID=${APP_UID} \
        SOCKET_PROXY_GID=${APP_GID} \
        SOCKET_PROXY_KEEPALIVE="10s" \
        SOCKET_PROXY_TIMEOUT="30s" \
        SOCKET_PROXY_DEADLINE="60s"

  # :: multi-stage
    COPY --from=distroless / /
    COPY --from=build /distroless/ /

# :: PERSISTENT DATA
  HEALTHCHECK --interval=5s --timeout=2s --start-period=5s \
    CMD ["/usr/local/bin/socket-proxy", "--healthcheck"]

# :: EXECUTE
  ENTRYPOINT ["/usr/local/bin/socket-proxy"]