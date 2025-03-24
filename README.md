![banner](https://github.com/11notes/defaults/blob/main/static/img/banner.png?raw=true)

# SOCKET-PROXY
[<img src="https://img.shields.io/badge/github-source-blue?logo=github&color=040308">](https://github.com/11notes/docker-SOCKET-PROXY)![5px](https://github.com/11notes/defaults/blob/main/static/img/transparent5x2px.png?raw=true)![size](https://img.shields.io/docker/image-size/11notes/socket-proxy/2.0.0?color=0eb305)![5px](https://github.com/11notes/defaults/blob/main/static/img/transparent5x2px.png?raw=true)![version](https://img.shields.io/docker/v/11notes/socket-proxy/2.0.0?color=eb7a09)![5px](https://github.com/11notes/defaults/blob/main/static/img/transparent5x2px.png?raw=true)![pulls](https://img.shields.io/docker/pulls/11notes/socket-proxy?color=2b75d6)![5px](https://github.com/11notes/defaults/blob/main/static/img/transparent5x2px.png?raw=true)[<img src="https://img.shields.io/github/issues/11notes/docker-SOCKET-PROXY?color=7842f5">](https://github.com/11notes/docker-SOCKET-PROXY/issues)![5px](https://github.com/11notes/defaults/blob/main/static/img/transparent5x2px.png?raw=true)![swiss_made](https://img.shields.io/badge/Swiss_Made-FFFFFF?labelColor=FF0000&logo=data:image/svg%2bxml;base64,PHN2ZyB2ZXJzaW9uPSIxIiB3aWR0aD0iNTEyIiBoZWlnaHQ9IjUxMiIgdmlld0JveD0iMCAwIDMyIDMyIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjxwYXRoIGQ9Im0wIDBoMzJ2MzJoLTMyeiIgZmlsbD0iI2YwMCIvPjxwYXRoIGQ9Im0xMyA2aDZ2N2g3djZoLTd2N2gtNnYtN2gtN3YtNmg3eiIgZmlsbD0iI2ZmZiIvPjwvc3ZnPg==)

Access your docker socket safely as read-only, rootless and distroless

# MAIN TAGS 🏷️
These are the main tags for the image. There is also a tag for each commit and its shorthand sha256 value.

* [2.0.0](https://hub.docker.com/r/11notes/socket-proxy/tags?name=2.0.0)
* [stable](https://hub.docker.com/r/11notes/socket-proxy/tags?name=stable)
* [latest](https://hub.docker.com/r/11notes/socket-proxy/tags?name=latest)

# SYNOPSIS 📖
**What can I do with this?** This image will run a proxy to access your docker socket as read-only. The exposed proxy socket is run as 1000:1000, not as root, although the image starts the proxy process as root to interact with the actual docker socket. There is also a TCP endpoint started at 2375 that will also proxy to the actual docker socket if needed. It is not exposed by default and must be exposed via using ```- "2375:2375/tcp"``` in your compose.

# UNIQUE VALUE PROPOSITION 💶
**Why should I run this image and not the other image(s) that already exist?** Good question! All the other images on the market that do exactly the same don’t do or offer these options:

> [!IMPORTANT]
>* This image runs the proxy part as a specific UID/GID (not root), all other images run everything as root
>* This image uses a single binary, all other images use apps like Nginx or HAProxy (bloat)
>* This image has no shell since it is 100% distroless, all other images run on a distro like Debian or Alpine with full shell access (security)
>* This image does not ship with any CVE and is automatically maintained via CI/CD, all other images mostly have no CVE scanning or code quality tools in place
>* This image has no upstream dependencies, all other images have upstream dependencies
>* This image exposes the socket as a UNIX socket and TCP socket, all other images only expose it via a TCP socket

If you value security, simplicity and the ability to interact with the maintainer and developer of an image. Using my images is a great start in that direction.

# COMPOSE ✂️
```yaml
name: "traefik" # this is a compose example for Traefik
services:
  socket-proxy:
    image: "11notes/socket-proxy:2.0.0"
    volumes:
      - "/run/docker.sock:/run/docker.sock:ro" # mount host docker socket, the :ro does not mean read-only for the socket, just for the actual file
      - "socket-proxy:/run/proxy" # this socket is run as 1000:1000, not as root!
    restart: "always"

  traefik:
    image: "11notes/traefik:3.2.0"
    depends_on:
      socket-proxy:
        condition: "service_healthy"
        restart: true
    command:
      - "--global.checkNewVersion=false"
      - "--global.sendAnonymousUsage=false"
      - "--api.dashboard=true"
      - "--api.insecure=true"
      - "--log.level=INFO"
      - "--log.format=json"
      - "--providers.docker.exposedByDefault=false" # use docker provider but do not expose by default
      - "--entrypoints.http.address=:80"
      - "--entrypoints.https.address=:443"
      - "--serversTransport.insecureSkipVerify=true" # do not verify downstream SSL certificates
    ports:
      - "80:80/tcp"
      - "443:443/tcp"
      - "8080:8080/tcp"
    networks:
      frontend:
      backend:
    volumes:
      - "socket-proxy:/var/run"
    sysctls:
      net.ipv4.ip_unprivileged_port_start: 80
    restart: "always"

  nginx: # example container
    image: "11notes/nginx:1.26.2"
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.default.priority=1"
      - "traefik.http.routers.default.rule=PathPrefix(`/`)"
      - "traefik.http.routers.default.entrypoints=http"
      - "traefik.http.routers.default.service=default"
      - "traefik.http.services.default.loadbalancer.server.port=8443"
      - "traefik.http.services.default.loadbalancer.server.scheme=https" # proxy from http to https since this image runs by default on https
    networks:
      backend: # allow container only to be accessed via traefik
    restart: "always"

volumes:
  socket-proxy:

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

# SOURCE 💾
* [11notes/socket-proxy](https://github.com/11notes/docker-SOCKET-PROXY)

# PARENT IMAGE 🏛️
* [scratch](https://hub.docker.com/_/scratch)

# BUILT WITH 🧰
* [11notes/util](https://github.com/11notes/docker-util)

# GENERAL TIPS 📌
> [!TIP]
>* Use a reverse proxy like Traefik, Nginx, HAproxy to terminate TLS and to protect your endpoints
>* Use Let’s Encrypt DNS-01 challenge to obtain valid SSL certificates for your services

# ElevenNotes™️
This image is provided to you at your own risk. Always make backups before updating an image to a different version. Check the [releases](https://github.com/11notes/docker-socket-proxy/releases) for breaking changes. If you have any problems with using this image simply raise an [issue](https://github.com/11notes/docker-socket-proxy/issues), thanks. If you have a question or inputs please create a new [discussion](https://github.com/11notes/docker-socket-proxy/discussions) instead of an issue. You can find all my other repositories on [github](https://github.com/11notes?tab=repositories).

*created 24.03.2025, 13:18:38 (CET)*