CC := go build
CFLAGS := -v
BIN_PATH := ./build
SRC_PATH := ./cmd
GOPATH := $(SRC_PATH)
SOURCES=$(wildcard $(SRC_PATH)/*.go)
# OBJECTS=$(patsubst $(SRC_PATH)%, $(OBJ_PATH)%, $(SOURCES:.c=.o))
TARGET_ARCHS := amd64 386
TARGET_OSS := linux
SERVICE := atella
DESCRIPTION := "Atella. Agent for distributed checking servers status"
EXECUTABLE := ${SERVICE}
LICENSE := "GPL-3.0"
URL := "https://github.com/JIexa24/Atella"

ARCH := amd64
OS := linux

GIT_BRANCH := "unknown"
GIT_HASH := $(shell git log --pretty=format:%H -n 1)
GIT_HASH_SHORT := $(shell echo "${GIT_HASH}" | cut -c1-7)
GIT_TAG := $(shell git describe --always --tags --abbrev=0 | tail -c+2)
GIT_COMMIT := $(shell git rev-list v${GIT_TAG}..HEAD --count)
GIT_COMMIT_DATE := $(shell git show -s --format=%ci | cut -d\  -f1)
GO_VERSION := $(shell go version | cut -d' ' -f3)
GO_PATH := $(shell go env GOPATH)

MAINTAINER := R9ODT

VERSION_RELEASE := ${GIT_TAG}.${GIT_COMMIT}

.PHONY: all 
all: build tar

.PHONY: DIRECTORY 
DIRECTORY: ${BIN_PATH}

${BIN_PATH}:
	$(if ifeq test -d "${BIN_PATH}" 0, @mkdir -p ${BIN_PATH})

.PHONY: build 
build: testbuild ${EXECUTABLE}

.PHONY: testbuild 
testbuild: ${SOURCES}
	CGO_ENABLED=0 ${CC} -a -installsuffix cgo -ldflags "-X main.Version=${VERSION_RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitCommit=${GIT_HASH}" -o $(BIN_PATH)/$@ $(CFLAGS) $^;

${EXECUTABLE}: ${SOURCES}
	for arch in ${TARGET_ARCHS}; do \
	  for os in ${TARGET_OSS}; do \
		  CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(CC) -a -installsuffix cgo -ldflags "-X main.Version=${VERSION_RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitCommit=${GIT_HASH}" -o $(BIN_PATH)/$@_"$$os"_"$$arch" $(CFLAGS) $^; \
		done; \
	done

.PHONY: tar-deb
tar-deb:
	rm -rf build/root
	mkdir -p build/root/${SERVICE}/usr/bin 
	mkdir -p build/root/${SERVICE}/etc/
	mkdir -p build/root/${SERVICE}/usr/lib/atella/scripts
	$(if ifeq ${ARCH} amd64, @cp build/${SERVICE}_${OS}_${ARCH} build/root/${SERVICE}/usr/bin/${SERVICE})
	cp -r etc/ build/root/${SERVICE}/etc/${SERVICE}
	cp pkg/atella.service build/root/${SERVICE}/usr/lib/atella/scripts/
	cp pkg/init.sh build/root/${SERVICE}/usr/lib/atella/scripts/
	cp pkg/atella.logrotate build/root/${SERVICE}/usr/lib/atella/scripts/
	tar -czvPf pkg/tar/${SERVICE}-${VERSION_RELEASE}.tar.gz -C build/root/${SERVICE} . 	

.PHONY: docker-pkgbuilder-64
docker-pkgbuilder-64:
	docker build -t ${SERVICE}-deb-pkgbuilder-64 -f docker/Dockerfile.deb-pkgbuilder-64 ./docker

.PHONY: deb-64
deb-64: tar-deb docker-pkgbuilder-64
	docker run --rm \
	-v "$(PWD)/pkg:/pkg" \
	-v "$(PWD)/etc:/etc/${SERVICE}" \
	-e ARCH="amd64" \
	-e DESCRIPTION=\"${DESCRIPTION}\" \
	-e VENDOR=${MAINTAINER} \
	-e URL=${URL} \
	-e LICENSE=${LICENSE} \
	-e SERVICE=${SERVICE} \
	-e VERSION_RELEASE=${VERSION_RELEASE} \
	${SERVICE}-deb-pkgbuilder-64

.PHONY: clean 
clean:
	rm -rf $(BIN_PATH)

.PHONY: restruct 
restruct: clean all
