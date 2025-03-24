${{ content_synopsis }} This image will run a proxy to access your docker socket as read-only. The exposed proxy socket is run as 1000:1000, not as root, although the image starts the proxy process as root to interact with the actual docker socket. There is also a TCP endpoint started at 2375 that will also proxy to the actual docker socket if needed. It is not exposed by default and must be exposed via using ```- "2375:2375/tcp"``` in your compose.

${{ content_uvp }} Good question! All the other images on the market that do exactly the same don’t do or offer these options:

${{ github:> [!IMPORTANT] }}
${{ github:> }}* This image runs the proxy part as a specific UID/GID (not root), all other images run everything as root
${{ github:> }}* This image uses a single binary, all other images use apps like Nginx or HAProxy (bloat)
${{ github:> }}* This image has no shell since it is 100% distroless, all other images run on a distro like Debian or Alpine with full shell access (security)
${{ github:> }}* This image does not ship with any CVE and is automatically maintained via CI/CD, all other images mostly have no CVE scanning or code quality tools in place
${{ github:> }}* This image has no upstream dependencies, all other images have upstream dependencies
${{ github:> }}* This image exposes the socket as a UNIX socket and TCP socket, all other images only expose it via a TCP socket

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