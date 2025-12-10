# Docker Stats Monitor - Makefile
# Terminal-based Docker statistics monitor

# Colors
ifneq (,$(findstring xterm,${TERM}))
   BLACK        := $(shell tput -Txterm setaf 0)
   RED          := $(shell tput -Txterm setaf 1)
   GREEN        := $(shell tput -Txterm setaf 2)
   YELLOW       := $(shell tput -Txterm setaf 3)
   LIGHTPURPLE  := $(shell tput -Txterm setaf 4)
   PURPLE       := $(shell tput -Txterm setaf 5)
   BLUE         := $(shell tput -Txterm setaf 6)
   WHITE        := $(shell tput -Txterm setaf 7)
   RESET := $(shell tput -Txterm sgr0)
else
   BLACK        := ""
   RED          := ""
   GREEN        := ""
   YELLOW       := ""
   LIGHTPURPLE  := ""
   PURPLE       := ""
   BLUE         := ""
   WHITE        := ""
   RESET        := ""
endif

# Variables
BINARY_NAME=docker-stats
BUILD_DIR=build
GO=go
GOFLAGS=-ldflags="-s -w"

.PHONY: all build clean test lint fmt security run help deps tidy

all: deps fmt lint test build ## 🚀 Run all tasks (deps, fmt, lint, test, build)

help: ## 📖 Show this help message
	@echo "${BLUE}Docker Stats Monitor - Available Commands${RESET}"
	@echo ""
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "${GREEN}%-20s${RESET} %s\n", $$1, $$2}'

deps: ## 📦 Download dependencies
	@echo "${BLUE}📦 Downloading dependencies...${RESET}"
	$(GO) mod download
	@echo "${GREEN}✓ Dependencies downloaded${RESET}"

tidy: ## 🧹 Tidy go modules
	@echo "${BLUE}🧹 Tidying go modules...${RESET}"
	$(GO) mod tidy
	@echo "${GREEN}✓ Modules tidied${RESET}"

build: ## 🔨 Build the binary
	@echo "${BLUE}🔨 Building $(BINARY_NAME)...${RESET}"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "${GREEN}✓ Built: $(BUILD_DIR)/$(BINARY_NAME)${RESET}"

build-all: ## 🔨 Build for multiple platforms
	@echo "${BLUE}🔨 Building for multiple platforms...${RESET}"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "${GREEN}✓ Built all platforms${RESET}"

clean: ## 🗑️  Clean build artifacts
	@echo "${BLUE}🗑️  Cleaning...${RESET}"
	rm -rf $(BUILD_DIR)
	$(GO) clean
	@echo "${GREEN}✓ Cleaned${RESET}"

test: ## 🧪 Run tests
	@echo "${BLUE}🧪 Running tests...${RESET}"
	$(GO) test -v -race -cover ./...
	@echo "${GREEN}✓ Tests passed${RESET}"

test-coverage: ## 📊 Run tests with coverage report
	@echo "${BLUE}📊 Running tests with coverage...${RESET}"
	@mkdir -p $(BUILD_DIR)
	$(GO) test -v -race -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "${GREEN}✓ Coverage report: $(BUILD_DIR)/coverage.html${RESET}"

lint: ## 🔍 Run linter
	@echo "${BLUE}🔍 Running linter...${RESET}"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "${YELLOW}⚠ golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest${RESET}"; \
		$(GO) vet ./...; \
	fi
	@echo "${GREEN}✓ Lint passed${RESET}"

fmt: ## 🎨 Format code
	@echo "${BLUE}🎨 Formatting code...${RESET}"
	$(GO) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi
	@echo "${GREEN}✓ Code formatted${RESET}"

security: ## 🔒 Run security scan
	@echo "${BLUE}🔒 Running security scan...${RESET}"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -quiet ./...; \
	else \
		echo "${YELLOW}⚠ gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest${RESET}"; \
	fi
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "${YELLOW}⚠ govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest${RESET}"; \
	fi
	@echo "${GREEN}✓ Security scan complete${RESET}"

run: build ## ▶️  Build and run
	@echo "${BLUE}▶️  Running $(BINARY_NAME)...${RESET}"
	./$(BUILD_DIR)/$(BINARY_NAME)

install: build ## 📥 Install to /usr/local/bin
	@echo "${BLUE}📥 Installing $(BINARY_NAME)...${RESET}"
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "${GREEN}✓ Installed to /usr/local/bin/$(BINARY_NAME)${RESET}"

uninstall: ## 📤 Uninstall from /usr/local/bin
	@echo "${BLUE}📤 Uninstalling $(BINARY_NAME)...${RESET}"
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "${GREEN}✓ Uninstalled${RESET}"

dev-tools: ## 🛠️  Install development tools
	@echo "${BLUE}🛠️  Installing development tools...${RESET}"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "${GREEN}✓ Development tools installed${RESET}"
