${{ content_synopsis }} This image will run a proxy to access your docker socket read-only. The exposed proxy socket is run as 1000:1000, not as root, although the image starts the proxy process as root to interact with the actual docker socket as root. There is also a TCP endpoint started at 8080 that will also proxy to the actual docker socket if needed.

${{ content_compose }}

${{ content_defaults }}

${{ content_environment }}
| `SOCKET_PROXY` | path to the socket used as a proxy | ${{ json_root }}$/run/docker.sock |
| `SOCKET_PROXY_DOCKER_SOCKET` | path to the actual docker socket | /run/docker.sock |

${{ content_source }}

${{ content_parent }}

${{ content_built }}

${{ content_tips }}