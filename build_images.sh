#!/usr/bin/env bash
PREFIX=r9odt
. ./images.dat
for ((i = 0; i < ${#IMAGES[@]}; i+=4)); do
  test -d ${IMAGES[i]}
  if [[ $? == 0 ]]; then
    docker build -t $PREFIX/${IMAGES[i]}:${IMAGES[i+2]} -t $PREFIX/${IMAGES[i]}:latest --build-arg IMAGE_VERSION=${IMAGES[i+1]} ./${IMAGES[i]}
  fi
done

for i in $@; do
  cd ./$i
  if [[ $? == 0 ]]; then 
    ./build_images.sh
    cd -
  fi
done