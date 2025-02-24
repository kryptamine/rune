# Define variables
APP_NAME := myapp
BUILD_DIR := build
SRC_DIR := .

# Detect the OS and ARCH
OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)
GOOS ?= $(OS)
GOARCH ?= $(ARCH)

# Default target
.PHONY: all
all: build

# Build the Go binary
.PHONY: build
build:
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BUILD_DIR)/$(APP_NAME) $(SRC_DIR)

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# Run the application
.PHONY: run
run: build
	./$(BUILD_DIR)/$(APP_NAME)
