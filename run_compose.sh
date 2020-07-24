#!/usr/bin/env bash
. ./images.dat
for ((i = 0; i < ${#IMAGES[@]}; i+=4)); do
  export ${IMAGES[i]}_image=${IMAGES[i+3]}
done

docker-compose up -d $*