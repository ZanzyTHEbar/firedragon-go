# Go build flags
BUILD_FLAGS := -v -ldflags "-X 'version.BuildTime=$(shell date -u)'"

# Binary name base
BINARY_BASE_NAME := firedragon

# Detect the operating system and architecture
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Binary names
BINARY_NAME := $(BINARY_BASE_NAME)_client_$(GOOS)_$(GOARCH)

# Binary path
BINARY_PATH_PREFIX := ./bin
BINARY_PATH := $(BINARY_PATH_PREFIX)/$(GOOS)_$(GOARCH)

MODULE_PATH := ./cmd/firedragon.go

# Default target
.DEFAULT_GOAL := build

TEMP_PATHS := /tmp/$(BINARY_BASE_NAME)_test_*

all: build

run:
	@echo "Running $(BINARY_NAME)..."
	@go run $(MODULE_PATH) &

dev: start-nats
	@command -v air >/dev/null 2>&1 || go install github.com/air-verse/air@latest
	@mkdir -p tmp
	@air -c .air.toml > $(BINARY_NAME).log 2>&1 & echo $$! > $(BINARY_NAME).pid
	@echo "Development environment started. Logs are in $(BINARY_NAME).log."
	@echo "NATs cli is running. Logs are in nats.log."
	@echo "Use 'make stop-dev' to stop the development environment."

stop-dev:
	@if [ -f $(BINARY_NAME).pid ]; then \
		if ps -p $$(cat $(BINARY_NAME).pid) > /dev/null; then \
			kill $$(cat $(BINARY_NAME).pid) && rm $(BINARY_NAME).pid; \
		else \
			echo "$(BINARY_NAME) process not running. Removing stale PID file."; \
			rm $(BINARY_NAME).pid; \
		fi \
	fi
	@make stop-nats
	@echo "Development environment stopped."

monitor-logs:
	@if [ -f $(BINARY_NAME).log ]; then $(TERMINAL) -e "tail -f $(BINARY_NAME).log" & fi
	@if [ -f nats.log ]; then $(TERMINAL) -e "tail -f nats.log" & fi

# nats context save nats_development --user nats_development --password <password> --server <server-url>
start-nats:
	@echo "Checking for local NATs server..."
	@if ! command -v nats >/dev/null 2>&1; then \
		echo "Local NATs server not found. Please install NATs.io cli."; \
		exit 1; \
	fi
	@if [ -f nats.pid ] && ps -p $$(cat nats.pid) > /dev/null; then \
		echo "Local NATs server is already running with PID $$(cat nats.pid)."; \
	else \
		[ -f nats.pid ] && rm nats.pid; \
		echo "Starting Local NATs server..."; \
		nats server run --jetstream --clean > nats.log 2>&1 & echo $$! > nats.pid; \
	fi
	@echo "Setting up NATs context..."
	@nats context save --select localdev --server localhost:9222 --user dev --password devpass
	@echo "Local NATs server is running. Logs are in nats.log."

stop-nats:
	@if [ -f nats.pid ]; then \
		if ps -p $$(cat nats.pid) > /dev/null; then \
			kill $$(cat nats.pid) && rm nats.pid; \
			echo "NATs cli stopped."; \
		else \
			echo "NATs cli is not running. Removing stale PID file."; \
			rm nats.pid; \
		fi \
	else \
		echo "NATs cli is not running or PID file not found."; \
	fi

build:
	@echo "Building $(CLIENT_BINARY_NAME) and $(SERVER_BINARY_NAME) ..."
	@mkdir -p $(BINARY_PATH)
	@go mod tidy -v
	@go build $(BUILD_FLAGS) -o $(BINARY_PATH)/$(CLIENT_BINARY_NAME) $(MODULE_PATH)

clean-all:
	@echo "Cleaning All..."
	@rm -rf $(BINARY_PATH_PREFIX)
	@go clean -cache -modcache -i -r

clean:
	@echo "Cleaning Binary..."
	@echo "Removing $(DEFAULT_DB_PATH) ..."
	@rm -f $(DEFAULT_DB_PATH)
	@echo "Removing $(BINARY_PATH_PREFIX) ..."
	@rm -rf $(BINARY_PATH_PREFIX)
	@echo "Removing temporary test directories $(TEMP_PATHS) ..."
	@rm -rf $(TEMP_PATHS)

gentest:
	@echo "Generating test directory..."
	@mkdir -p /tmp/$(BINARY_BASE_NAME)_test_$(shell date +'%Y%m%d%H%M%S')_$(shell shuf -i 1-1000 -n 1)

genproto:
	@echo "Generating proto files using buf cli ..."
	@buf lint $(PROTO_DIR) || (echo "Linting failed. Fix the issues above and try again." && exit 1)
	@buf generate $(PROTO_DIR)
	@echo "Done generating proto files"

delproto:
	@echo "Deleting generated proto files ..."
	@rm -rf $(PROTO_DIR)/gen
	@echo "Done deleting generated proto files"

# Test the application
test:
	@echo "Testing..."
	@go test ./tests -v

# Detect the operating system
OS := $(shell uname -s)

# Define terminal commands based on OS
ifeq ($(OS),Linux)
    # Check for common Linux terminal emulators in order of preference
    ifeq ($(shell command -v gnome-terminal),)
        ifeq ($(shell command -v kitty),)
            ifeq ($(shell command -v konsole),)
                ifeq ($(shell command -v xterm),)
                    TERMINAL := echo "No terminal emulator found."
                else
                    TERMINAL := xterm
                endif
            else
                TERMINAL := konsole
            endif
        else
            TERMINAL := kitty
        endif
    else
        TERMINAL := gnome-terminal
    endif
else ifeq ($(OS),Darwin)  # macOS
    TERMINAL := open -a Terminal
else  # Assume Windows if not Linux or macOS
    TERMINAL := powershell -Command "Start-Process powershell -ArgumentList '-NoExit', '-Command', 'Get-Content %1 -Wait'"
endif

.PHONY: all build run test clean clean-all dev