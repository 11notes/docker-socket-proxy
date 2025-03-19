#!/bin/ash
  if [ -z "${1}" ]; then
    set -- "socket-proxy"    
    eleven log start
  fi

  exec "$@"