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

.PHONY: all 
all: DIRECTORY $(EXECUTABLE)

DIRECTORY: $(BIN_PATH)

$(BIN_PATH):
	$(if ifeq test -d "$(BIN_PATH)" 0, @mkdir -p $(BIN_PATH))

$(EXECUTABLE): $(SOURCES)
	for arch in $(TARGET_ARCH); do for os in $(TARGET_OS); do $(CC) -o $(BIN_PATH)/$@_"$$os"_"$$arch" $(CFLAGS) $^; done; done

clean:
	rm -rf $(BIN_PATH)

restruct: clean all
