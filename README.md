![banner](https://github.com/11notes/defaults/blob/main/static/img/banner.png?raw=true)

# SOCKET-PROXY
![size](https://img.shields.io/docker/image-size/11notes/socket-proxy/2.1.4?color=0eb305)![5px](https://github.com/11notes/defaults/blob/main/static/img/transparent5x2px.png?raw=true)![version](https://img.shields.io/docker/v/11notes/socket-proxy/2.1.4?color=eb7a09)![5px](https://github.com/11notes/defaults/blob/main/static/img/transparent5x2px.png?raw=true)![pulls](https://img.shields.io/docker/pulls/11notes/socket-proxy?color=2b75d6)![5px](https://github.com/11notes/defaults/blob/main/static/img/transparent5x2px.png?raw=true)[<img src="https://img.shields.io/github/issues/11notes/docker-SOCKET-PROXY?color=7842f5">](https://github.com/11notes/docker-SOCKET-PROXY/issues)![5px](https://github.com/11notes/defaults/blob/main/static/img/transparent5x2px.png?raw=true)![swiss_made](https://img.shields.io/badge/Swiss_Made-FFFFFF?labelColor=FF0000&logo=data:image/svg%2bxml;base64,PHN2ZyB2ZXJzaW9uPSIxIiB3aWR0aD0iNTEyIiBoZWlnaHQ9IjUxMiIgdmlld0JveD0iMCAwIDMyIDMyIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPgogIDxyZWN0IHdpZHRoPSIzMiIgaGVpZ2h0PSIzMiIgZmlsbD0idHJhbnNwYXJlbnQiLz4KICA8cGF0aCBkPSJtMTMgNmg2djdoN3Y2aC03djdoLTZ2LTdoLTd2LTZoN3oiIGZpbGw9IiNmZmYiLz4KPC9zdmc+)

Access your docker socket safely as read-only, rootless and distroless

# SYNOPSIS 📖
**What can I do with this?** This image will run a proxy to access your docker socket as read-only. The exposed proxy socket is run as 1000:1000, not as root, although the image starts the proxy process as root to interact with the actual docker socket. There is also a TCP endpoint started at 2375 that will also proxy to the actual docker socket if needed. It is not exposed by default and must be exposed via using ```- "2375:2375/tcp"``` in your compose.

Make sure that the docker socket is accessible by the ```user:``` specification in your compose, if the UID/GID are not correct, the image will print out the correct UID/GID for you to set:

```shell
socket-proxy-1  | 2025/03/26 10:16:33 can’t access docker socket as GID 0 owned by GID 991
socket-proxy-1  | please change the user setting in your compose to the correct UID/GID pair like this:
socket-proxy-1  | services:
socket-proxy-1  |   socket-proxy:
socket-proxy-1  |     user: "0:991"
```

You find the list of all available Docker API endpoints [here](https://docs.docker.com/reference/api/engine/version/v1.51/). The following paths are still blocked, even though they are accesses only via GET:

- [GET /containers/{id}/attach/ws](https://docs.docker.com/reference/api/engine/version/v1.51/#tag/Container/operation/ContainerAttachWebsocket)
- [GET /containers/{id}/export](https://docs.docker.com/reference/api/engine/version/v1.51/#tag/Container/operation/ContainerExport)
- [GET /containers/{id}/archive](https://docs.docker.com/reference/api/engine/version/v1.51/#tag/Container/operation/ContainerArchive)
- [GET /secrets](https://docs.docker.com/reference/api/engine/version/v1.51/#tag/Secret/operation/SecretList)
- [GET /configs](https://docs.docker.com/reference/api/engine/version/v1.51/#tag/Config)
- [GET /swarm/unlockkey](https://docs.docker.com/reference/api/engine/version/v1.51/#tag/Swarm/operation/SwarmUnlockkey)
- [GET /images/{name}/get](https://docs.docker.com/reference/api/engine/version/v1.51/#tag/Image/operation/ImageGet)

# UNIQUE VALUE PROPOSITION 💶
**Why should I run this image and not the other image(s) that already exist?** Good question! Because ...

> [!IMPORTANT]
>* ... this image exposes the socket not as root but as 1000:1000
>* ... this image has no shell since it is [distroless](https://github.com/11notes/RTFM/blob/main/linux/container/image/distroless.md)
>* ... this image is auto updated to the latest version via CI/CD
>* ... this image has a health check
>* ... this image runs read-only
>* ... this image is automatically scanned for CVEs before and after publishing
>* ... this image is created via a secure and pinned CI/CD process
>* ... this image is very small

If you value security, simplicity and optimizations to the extreme, then this image might be for you.

# COMPOSE ✂️
```yaml
name: "reverse-proxy"
services:
  socket-proxy:
    # this image is used to expose the docker socket as read-only to traefik
    # you can check https://github.com/11notes/docker-socket-proxy for all details
    image: "11notes/socket-proxy:2.1.4"
    read_only: true
    user: "0:108" 
    environment:
      TZ: "Europe/Zurich"
    volumes:
      - "/run/docker.sock:/run/docker.sock:ro" 
      - "socket-proxy.run:/run/proxy"
    restart: "always"

  traefik:
    depends_on:
      socket-proxy:
        condition: "service_healthy"
        restart: true
    image: "11notes/traefik:3.5.0"
    read_only: true
    labels:
      - "traefik.enable=true"

      # default errors middleware
      - "traefik.http.middlewares.default-errors.errors.status=402-599"
      - "traefik.http.middlewares.default-errors.errors.query=/{status}"
      - "traefik.http.middlewares.default-errors.errors.service=default-errors"

      # default ratelimit
      - "traefik.http.middlewares.default-ratelimit.ratelimit.average=100"
      - "traefik.http.middlewares.default-ratelimit.ratelimit.burst=120"
      - "traefik.http.middlewares.default-ratelimit.ratelimit.period=1s"

      # default CSP
      - "traefik.http.middlewares.default-csp.headers.contentSecurityPolicy=default-src 'self' blob: data: 'unsafe-inline'"

      # default allowlist
      - "traefik.http.middlewares.default-ipallowlist-RFC1918.ipallowlist.sourcerange=10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"

      # example on how to secure the traefik dashboard and api
      - "traefik.http.routers.dashboard.rule=Host(`${TRAEFIK_FQDN}`)"
      - "traefik.http.routers.dashboard.service=api@internal"
      - "traefik.http.routers.dashboard.middlewares=dashboard-auth"
      - "traefik.http.routers.dashboard.entrypoints=https"
      # admin / traefik, please change!
      - "traefik.http.middlewares.dashboard-auth.basicauth.users=admin:$2a$12$ktgZsFQZ0S1FeQbI1JjS9u36fAJMHDQaY6LNi9EkEp8sKtP5BK43C"

      # default catch-all router
      - "traefik.http.routers.default.rule=HostRegexp(`.+`)"
      - "traefik.http.routers.default.priority=1"
      - "traefik.http.routers.default.entrypoints=https"
      - "traefik.http.routers.default.service=default-errors"

      # default http to https redirection
      - "traefik.http.middlewares.default-http.redirectscheme.permanent=true"
      - "traefik.http.middlewares.default-http.redirectscheme.scheme=https"
      - "traefik.http.routers.default-http.priority=1"
      - "traefik.http.routers.default-http.rule=HostRegexp(`.+`)"
      - "traefik.http.routers.default-http.entrypoints=http"
      - "traefik.http.routers.default-http.middlewares=default-http"
      - "traefik.http.routers.default-http.service=default-http"
      - "traefik.http.services.default-http.loadbalancer.passhostheader=true"
    environment:
      TZ: "Europe/Zurich"
    command:
      # ping is needed for the health check to work!
      - "--ping=true"
      - "--ping.terminatingStatusCode=204"
      - "--global.checkNewVersion=false"
      - "--global.sendAnonymousUsage=false"
      - "--accesslog=true"
      - "--api.dashboard=true"
      # disable insecure api and dashboard access
      - "--api.insecure=false"
      - "--log.level=INFO"
      - "--log.format=json"
      - "--providers.docker.exposedByDefault=false"
      - "--providers.file.directory=/traefik/var"
      - "--entrypoints.http.address=:80"
      - "--entrypoints.http.http.middlewares=default-errors,default-ratelimit,default-ipallowlist-RFC1918,default-csp"
      - "--entrypoints.https.address=:443"
      - "--entrypoints.https.http.tls=true"
      - "--entrypoints.https.http.middlewares=default-errors,default-ratelimit,default-ipallowlist-RFC1918,default-csp"
      # disable upstream HTTPS certificate checks (https > https)
      - "--serversTransport.insecureSkipVerify=true"
      - "--experimental.plugins.rewriteResponseHeaders.moduleName=github.com/jamesmcroft/traefik-plugin-rewrite-response-headers"
      - "--experimental.plugins.rewriteResponseHeaders.version=v1.1.2"
      - "--experimental.plugins.geoblock.moduleName=github.com/PascalMinder/geoblock"
      - "--experimental.plugins.geoblock.version=v0.3.3"
    ports:
      - "80:80/tcp"
      - "443:443/tcp"
    volumes:
      - "var:/traefik/var"
      - "plugins:/traefik/plugins"
      # access docker socket via proxy read-only
      - "socket-proxy.run:/var/run"
    networks:
      backend:
      frontend:
    sysctls:
      # allow rootless container to access ports < 1024
      net.ipv4.ip_unprivileged_port_start: 80
    restart: "always"

  errors:
    # this image can be used to display a simple error message since Traefik can’t serve content
    image: "11notes/traefik:errors"
    read_only: true
    labels:
      - "traefik.enable=true"
      - "traefik.http.services.default-errors.loadbalancer.server.port=8080"
    environment:
      TZ: "Europe/Zurich"
    networks:
      backend:
    restart: "always"

  # example container
  nginx:
    image: "11notes/nginx:stable"
    read_only: true
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.nginx-example.rule=Host(`${NGINX_FQDN}`)"
      - "traefik.http.routers.nginx-example.entrypoints=https"
      - "traefik.http.routers.nginx-example.service=nginx-example"
    ports:
      - "3000:3000/tcp"
    tmpfs:
      # needed for read_only: true
      - "/nginx/cache:uid=1000,gid=1000"
      - "/nginx/run:uid=1000,gid=1000"
    networks:
      backend:
    restart: "always"

volumes:
  var:
  plugins:
  socket-proxy.run:

networks:
  frontend:
  backend:
    internal: true
```

# ENVIRONMENT 📝
| Parameter | Value | Default |
| --- | --- | --- |
| `TZ` | [Time Zone](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) | |
| `DEBUG` | Will activate debug option for container image and app (if available) | |
| `SOCKET_PROXY_VOLUME` | path to the docker volume used to expose the prox socket | /run/proxy |
| `SOCKET_PROXY_DOCKER_SOCKET` | path to the actual docker socket | /run/docker.sock |
| `SOCKET_PROXY_UID` | the UID used to run the proxy parts | 1000 |
| `SOCKET_PROXY_GID` | the GID used to run the proxy parts | 1000 |
| `SOCKET_PROXY_KEEPALIVE` | connection keep alive interval to SOCKET_PROXY_DOCKER_SOCKET | 10s |
| `SOCKET_PROXY_TIMEOUT` | connection max. timeout to SOCKET_PROXY_DOCKER_SOCKET | 30s |

# MAIN TAGS 🏷️
These are the main tags for the image. There is also a tag for each commit and its shorthand sha256 value.

* [2.1.4](https://hub.docker.com/r/11notes/socket-proxy/tags?name=2.1.4)

### There is no latest tag, what am I supposed to do about updates?
It is of my opinion that the ```:latest``` tag is dangerous. Many times, I’ve introduced **breaking** changes to my images. This would have messed up everything for some people. If you don’t want to change the tag to the latest [semver](https://semver.org/), simply use the short versions of [semver](https://semver.org/). Instead of using ```:2.1.4``` you can use ```:2``` or ```:2.1```. Since on each new version these tags are updated to the latest version of the software, using them is identical to using ```:latest``` but at least fixed to a major or minor version.

If you still insist on having the bleeding edge release of this app, simply use the ```:rolling``` tag, but be warned! You will get the latest version of the app instantly, regardless of breaking changes or security issues or what so ever. You do this at your own risk!

# REGISTRIES ☁️
```
docker pull 11notes/socket-proxy:2.1.4
docker pull ghcr.io/11notes/socket-proxy:2.1.4
docker pull quay.io/11notes/socket-proxy:2.1.4
```

# SOURCE 💾
* [11notes/socket-proxy](https://github.com/11notes/docker-SOCKET-PROXY)

# PARENT IMAGE 🏛️
> [!IMPORTANT]
>This image is not based on another image but uses [scratch](https://hub.docker.com/_/scratch) as the starting layer.
>The image consists of the following distroless layers that were added:
>* [11notes/distroless](https://github.com/11notes/docker-distroless/blob/master/arch.dockerfile) - contains users, timezones and Root CA certificates



# GENERAL TIPS 📌
> [!TIP]
>* Use a reverse proxy like Traefik, Nginx, HAproxy to terminate TLS and to protect your endpoints
>* Use Let’s Encrypt DNS-01 challenge to obtain valid SSL certificates for your services

# ElevenNotes™️
This image is provided to you at your own risk. Always make backups before updating an image to a different version. Check the [releases](https://github.com/11notes/docker-socket-proxy/releases) for breaking changes. If you have any problems with using this image simply raise an [issue](https://github.com/11notes/docker-socket-proxy/issues), thanks. If you have a question or inputs please create a new [discussion](https://github.com/11notes/docker-socket-proxy/discussions) instead of an issue. You can find all my other repositories on [github](https://github.com/11notes?tab=repositories).

*created 12.08.2025, 08:20:27 (CET)*