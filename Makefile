CC := go build
CFLAGS := -v
BIN_PATH := ./bin
SRC_PATH := ./cmd
GOPATH := $(SRC_PATH)
SOURCES=$(wildcard $(SRC_PATH)/*.go)
# OBJECTS=$(patsubst $(SRC_PATH)%, $(OBJ_PATH)%, $(SOURCES:.c=.o))
TARGET_ARCH := amd64 386
TARGET_OS := linux
EXECUTABLE := mags


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
all: DIRECTORY $(EXECUTABLE)

DIRECTORY: $(BIN_PATH)

$(BIN_PATH):
	$(if ifeq test -d "$(BIN_PATH)" 0, @mkdir -p $(BIN_PATH))

$(EXECUTABLE): $(SOURCES)
	for arch in $(TARGET_ARCH); do \
	  for os in $(TARGET_OS); do \
		  CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(CC) -a -installsuffix cgo -ldflags "-X main.Version=${VERSION_RELEASE} -X main.GoVersion=${GO_VERSION} -X main.GitCommit=${GIT_HASH}" -o $(BIN_PATH)/$@_"$$os"_"$$arch" $(CFLAGS) $^; \
		done; \
	done

clean:
	rm -rf $(BIN_PATH)

restruct: clean all
