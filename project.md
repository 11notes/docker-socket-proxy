${{ content_synopsis }} This image will run a proxy to access your docker socket as read-only. The exposed proxy socket is run as 1000:1000, not as root, although the image starts the proxy process as root to interact with the actual docker socket. There is also a TCP endpoint started at 2375 that will also proxy to the actual docker socket if needed. It is not exposed by default and must be exposed via using ```- "2375:2375/tcp"``` in your compose.

Make sure that the docker socket is accessible by the ```user:``` specification in your compose, if the UID/GID are not correct, the image will print out the correct UID/GID for you to set:

```shell
socket-proxy-1  | 2025/03/26 10:16:33 can’t access docker socket as GID 0 owned by GID 991
socket-proxy-1  | please change the user setting in your compose to the correct UID/GID pair like this:
socket-proxy-1  | services:
socket-proxy-1  |   socket-proxy:
socket-proxy-1  |     user: "0:991"
```

${{ content_uvp }} Good question! Because ...

${{ github:> [!IMPORTANT] }}
${{ github:> }}* ... this image exposes the socket not as root but as 1000:1000
${{ github:> }}* ... this image has no shell since it is [distroless](https://github.com/11notes/RTFM/blob/main/linux/container/image/distroless.md)
${{ github:> }}* ... this image is auto updated to the latest version via CI/CD
${{ github:> }}* ... this image has a health check
${{ github:> }}* ... this image runs read-only
${{ github:> }}* ... this image is automatically scanned for CVEs before and after publishing
${{ github:> }}* ... this image is created via a secure and pinned CI/CD process
${{ github:> }}* ... this image is very small

If you value security, simplicity and optimizations to the extreme, then this image might be for you.

${{ content_compose }}

${{ content_environment }}
| `SOCKET_PROXY_VOLUME` | path to the docker volume used to expose the prox socket | /run/proxy |
| `SOCKET_PROXY_DOCKER_SOCKET` | path to the actual docker socket | /run/docker.sock |
| `SOCKET_PROXY_UID` | the UID used to run the proxy parts | 1000 |
| `SOCKET_PROXY_GID` | the GID used to run the proxy parts | 1000 |
| `SOCKET_PROXY_KEEPALIVE` | connection keep alive interval to SOCKET_PROXY_DOCKER_SOCKET | 10s |
| `SOCKET_PROXY_TIMEOUT` | connection max. timeout to SOCKET_PROXY_DOCKER_SOCKET | 30s |

${{ content_source }}

${{ content_parent }}

${{ content_built }}

${{ content_tips }}