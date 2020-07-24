#!/bin/sh

if [ "$1" = "/bin/start.sh" ]; then
  if [ "$(id -u)" = "0" ]; then
    exec gosu $IMAGE_USER "$@"
  fi
fi

exec "$@"