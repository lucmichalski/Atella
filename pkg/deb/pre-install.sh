#!/bin/bash

if ! getent group "atella" > /dev/null 2>&1 ; then
  groupadd -r "atella"
fi

if ! getent passwd "atella" > /dev/null 2>&1 ; then
  useradd -m -r -g atella -d /usr/share/atella -s /sbin/nologin \
    -c "Atella user" atella
fi