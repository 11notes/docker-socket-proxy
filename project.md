${{ content_synopsis }} This image will run a proxy to access your docker socket as read-only. The exposed proxy socket is run as 1000:1000, not as root, although the image starts the proxy process as root to interact with the actual docker socket. There is also a TCP endpoint started at 2375 that will also proxy to the actual docker socket if needed. It is not exposed by default and must be exposed via using ```- "2375:2375/tcp"``` in your compose.

Make sure that the docker socket is accessible by the ```user:``` specification in your compose, if the UID/GID are not correct, the image will print out the correct UID/GID for you to set:

```shell
socket-proxy-1  | 2025/03/26 10:16:33 can’t access docker socket as GID 0 owned by GID 991
socket-proxy-1  | please change the user setting in your compose to the correct UID/GID pair like this:
socket-proxy-1  | services:
socket-proxy-1  |   socket-proxy:
socket-proxy-1  |     user: "0:991"
```

${{ content_uvp }} Good question! All the other images on the market that do exactly the same don’t do or offer these options:

${{ github:> [!IMPORTANT] }}
${{ github:> }}* This image runs the proxy part as a specific UID/GID (not root), most other images run everything as root
${{ github:> }}* This image uses a single binary, most other images use apps like Nginx or HAProxy (bloat)
${{ github:> }}* This image has no shell since it is 100% distroless, most other images run on a distro like Debian or Alpine with full shell access (security)
${{ github:> }}* This image does not ship with any critical or high rated CVE and is automatically maintained via CI/CD, most other images mostly have no CVE scanning or code quality tools in place
${{ github:> }}* This image is created via a secure, pinned CI/CD process and immune to upstream attacks, most other images have upstream dependencies that can be exploited
${{ github:> }}* This image contains a proper health check that verifies the app is actually working, most other images have either no health check or only check if a port is open or ping works
${{ github:> }}* This image exposes the socket as a UNIX socket and TCP socket, most other images only expose it via a TCP socket
${{ github:> }}* This image works as read-only, most other images need to write files to the image filesystem

If you value security, simplicity and the ability to interact with the maintainer and developer of an image. Using my images is a great start in that direction.

${{ content_compose }}

${{ content_environment }}
| `SOCKET_PROXY_VOLUME` | path to the docker volume used to expose the prox socket | /run/proxy |
| `SOCKET_PROXY_DOCKER_SOCKET` | path to the actual docker socket | /run/docker.sock |
| `SOCKET_PROXY_UID` | the UID used to run the proxy parts | 1000 |
| `SOCKET_PROXY_GID` | the GID used to run the proxy parts | 1000 |

${{ content_source }}

${{ content_parent }}

${{ content_built }}

${{ content_tips }}