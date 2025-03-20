![banner](https://github.com/11notes/defaults/blob/main/static/img/banner.png?raw=true)

# ⛰️ socket-proxy
[<img src="https://img.shields.io/badge/github-source-blue?logo=github&color=040308">](https://github.com/11notes/docker-socket-proxy)![size](https://img.shields.io/docker/image-size/11notes/socket-proxy/1.0.1?color=0eb305)![version](https://img.shields.io/docker/v/11notes/socket-proxy/1.0.1?color=eb7a09)![pulls](https://img.shields.io/docker/pulls/11notes/socket-proxy?color=2b75d6)[<img src="https://img.shields.io/github/issues/11notes/docker-socket-proxy?color=7842f5">](https://github.com/11notes/docker-socket-proxy/issues)

Access your docker socket safely as read-only

# MAIN TAGS 🏷️
These are the main tags for the image. There is also a tag for each commit and its shorthand sha256 value.

* [1.0.1](https://hub.docker.com/r/11notes/socket-proxy/tags?name=1.0.1)
* [stable](https://hub.docker.com/r/11notes/socket-proxy/tags?name=stable)
* [latest](https://hub.docker.com/r/11notes/socket-proxy/tags?name=latest)

# SYNOPSIS 📖
**What can I do with this?** This image will run a proxy to access your docker socket read-only. The exposed proxy socket is run as 1000:1000, not as root, although the image starts the proxy process as root to interact with the actual docker socket as root. There is also a TCP endpoint started at 8080 that will also proxy to the actual docker socket if needed.

# COMPOSE ✂️
```yaml
name: "socket-proxy"
services:
  socket-proxy:
    image: "11notes/socket-proxy:1.0.1"
    volumes:
      - "/run/docker.sock:/run/docker.sock:ro" # mount host docker socket, the :ro does not mean read-only for the socket, just for the actual file
      - "socket-proxy:/socket-proxy/run" # this socket is run as 1000:1000, not as root!
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

  nginx:
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

# DEFAULT SETTINGS 🗃️
| Parameter | Value | Description |
| --- | --- | --- |
| `user` | docker | user name |
| `uid` | 1000 | [user identifier](https://en.wikipedia.org/wiki/User_identifier) |
| `gid` | 1000 | [group identifier](https://en.wikipedia.org/wiki/Group_identifier) |
| `home` | /socket-proxy | home directory of user docker |

# ENVIRONMENT 📝
| Parameter | Value | Default |
| --- | --- | --- |
| `TZ` | [Time Zone](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) | |
| `DEBUG` | Will activate debug option for container image and app (if available) | |
| `SOCKET_PROXY` | path to the socket used as a proxy | /socket-proxy$/run/docker.sock |
| `SOCKET_PROXY_DOCKER_SOCKET` | path to the actual docker socket | /run/docker.sock |

# SOURCE 💾
* [11notes/socket-proxy](https://github.com/11notes/docker-socket-proxy)

# PARENT IMAGE 🏛️
* [11notes/alpine:stable](https://hub.docker.com/r/11notes/alpine)

# BUILT WITH 🧰
* [11notes/util](https://github.com/11notes/docker-util)

# GENERAL TIPS 📌
* Use a reverse proxy like Traefik, Nginx, HAproxy to terminate TLS and to protect your endpoints
* Use Let’s Encrypt DNS-01 challenge to obtain valid SSL certificates for your services

# ElevenNotes™️
This image is provided to you at your own risk. Always make backups before updating an image to a different version. Check the [releases](https://github.com/11notes/docker-socket-proxy/releases) for breaking changes. If you have any problems with using this image simply raise an [issue](https://github.com/11notes/docker-socket-proxy/issues), thanks. If you have a question or inputs please create a new [discussion](https://github.com/11notes/docker-socket-proxy/discussions) instead of an issue. You can find all my other repositories on [github](https://github.com/11notes?tab=repositories).

*created 20.03.2025, 01:46:58 (CET)*