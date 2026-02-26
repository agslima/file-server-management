# Main Makefile for File Server Management 
# Coordinates between multiple sub-Makefiles organized by functionality

# Go binary and tools
GO := go
GOLANGCI_LINT := golangci-lint
GOTESTSUM := gotestsum
GOCOVER_COBERTURA := gocover-cobertura
ENVTEST := setup-envtest

# Variables
GOPATH ?= $(shell go env GOPATH)
GO_CACHE_DIR ?= $(shell go env GOCACHE)

# Load tool versions from .versions.yaml (single source of truth)
# Requires yq to be installed: brew install yq (macOS) or see https://github.com/mikefarah/yq
YQ := $(shell command -v yq 2> /dev/null)

ifndef YQ
$(error yq is required but not found. Install it with: brew install yq (macOS) or see https://github.com/mikefarah/yq)
endif

# Load versions from .versions.yaml
ADDLICENSE_VERSION := $(shell $(YQ) '.linting.addlicense' .versions.yaml)
BLACK_VERSION := $(shell $(YQ) '.linting.black' .versions.yaml)
DOCKER_BUILDX_VERSION := $(shell $(YQ) '.container_tools.docker_buildx' .versions.yaml)
GO_VERSION := $(shell $(YQ) '.languages.go' .versions.yaml)
GOCOVER_COBERTURA_VERSION := $(shell $(YQ) '.go_tools.gocover_cobertura' .versions.yaml)
GOLANGCI_LINT_VERSION := $(shell $(YQ) '.go_tools.golangci_lint' .versions.yaml)
GOTESTSUM_VERSION := $(shell $(YQ) '.go_tools.gotestsum' .versions.yaml)
GRPCIO_TOOLS_VERSION := $(shell $(YQ) '.protobuf.grpcio_tools' .versions.yaml)
KO_VERSION := $(shell $(YQ) '.container_tools.ko' .versions.yaml)
PROTOBUF_VERSION := $(shell $(YQ) '.protobuf.protobuf' .versions.yaml)
PROTOC_GEN_GO_GRPC_VERSION := $(shell $(YQ) '.protobuf.protoc_gen_go_grpc' .versions.yaml)
PROTOC_GEN_GO_VERSION := $(shell $(YQ) '.protobuf.protoc_gen_go' .versions.yaml)
PYTHON_VERSION := $(shell $(YQ) '.languages.python' .versions.yaml)
SHELLCHECK_VERSION := $(shell $(YQ) '.linting.shellcheck' .versions.yaml)
PHP_VERSION := $(shell $(YQ) '.languages.php' .versions.yaml)
NODEJS_VERSION := $(shell $(YQ) '.linting.nodejs' .versions.yaml)

# Go modules with specific patterns from CI
GO_MODULE := \
	file-engine

# Python modules
PYTHON_MODULE := \
	tests

# Container-only modules
CONTAINER_MODULES := \
	backend \
    file-engine \
    file-engine\api \
    frontend

# Backend module 
BACKEND_MODULE := \
    backend

# Frontend module
FRONTEND_MODULE := \
    frontend

# Default target
.PHONY: all
all: lint-test-all ## Run lint-test-all (default target)

# Show loaded tool versions
.PHONY: show-versions
show-versions: ## Display all tool versions loaded from .versions.yaml
ifndef YQ
	@echo "⚠️  ERROR: yq is required to display versions"
	@echo "   Install yq for version management:"
	@echo "   macOS:  brew install yq"
	@echo "   Linux:  See https://github.com/mikefarah/yq"
	@exit 1
else
	@echo "=== Tool Versions (from .versions.yaml) ==="
	@echo ""
	@$(YQ) eval '.languages | to_entries | .[] | "  " + .key + ": " + .value' .versions.yaml | \
		(echo "Languages:" && cat)
	@echo ""
	@$(YQ) eval '.build_tools | to_entries | .[] | "  " + .key + ": " + .value' .versions.yaml | \
		(echo "Build Tools:" && cat)
	@echo ""
	@$(YQ) eval '.go_tools | to_entries | .[] | "  " + .key + ": " + .value' .versions.yaml | \
		(echo "Go Tools:" && cat)
	@echo ""
	@$(YQ) eval '.protobuf | to_entries | .[] | "  " + .key + ": " + .value' .versions.yaml | \
		(echo "Protocol Buffers:" && cat)
	@echo ""
	@$(YQ) eval '.linting | to_entries | .[] | "  " + .key + ": " + .value' .versions.yaml | \
		(echo "Linting:" && cat)
	@echo ""
	@$(YQ) eval '.container_tools | to_entries | .[] | "  " + .key + ": " + .value' .versions.yaml | \
		(echo "Container Tools:" && cat)
	@echo ""
	@$(YQ) eval '.testing_tools | to_entries | .[] | "  " + .key + ": " + .value' .versions.yaml | \
		(echo "Testing & E2E Tools:" && cat)
	@echo ""
	@echo "==========================================="
endif

# ============================================================================
# Development Environment
# ============================================================================
# Setup development environment
.PHONY: dev-env-setup
dev-env-setup: ## Setup complete development environment (installs all required tools). Use AUTO_MODE=true to skip prompts
	@echo "Setting up File Server Management development environment..."
	@AUTO_MODE=$(AUTO_MODE) bash scripts/setup-dev-env.sh

# Install lint tools
.PHONY: install-lint-tools
install-lint-tools: install-golangci-lint install-gotestsum install-gocover-cobertura ## Install all lint tools (golangci-lint, gotestsum, gocover-cobertura)
	@echo "All lint tools installed successfully"
	@echo ""
	@echo "=== Installed Tool Versions and Locations ==="
	@echo "Go: $$(go version)"
	@echo "    Location: $$(which go)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint: $$(golangci-lint version 2>/dev/null | head -1)"; \
		echo "    Location: $$(which golangci-lint)"; \
	else \
		echo "golangci-lint: not found"; \
	fi
	@if command -v gotestsum >/dev/null 2>&1; then \
		echo "gotestsum: $$(gotestsum --version 2>/dev/null || echo 'version command not available')"; \
		echo "    Location: $$(which gotestsum)"; \
	else \
		echo "gotestsum: not found"; \
	fi
	@if command -v gocover-cobertura >/dev/null 2>&1; then \
		echo "gocover-cobertura: installed (no version command available)"; \
		echo "    Location: $$(which gocover-cobertura)"; \
	else \
		echo "gocover-cobertura: not found"; \
	fi
	@echo "=============================================="

# Install golangci-lint
.PHONY: install-golangci-lint
install-golangci-lint:
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		current_version=$$(golangci-lint version 2>/dev/null | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' || echo "unknown"); \
		if [ "$$current_version" = "$(GOLANGCI_LINT_VERSION)" ]; then \
			echo "golangci-lint $(GOLANGCI_LINT_VERSION) is already installed at $$(which golangci-lint)"; \
		else \
			existing_path=$$(which golangci-lint); \
			install_dir=$$(dirname "$$existing_path"); \
			echo "Current version: $$current_version at $$existing_path, installing $(GOLANGCI_LINT_VERSION) to $$install_dir..."; \
			curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$$install_dir" $(GOLANGCI_LINT_VERSION); \
		fi; \
	else \
		echo "golangci-lint not found, installing $(GOLANGCI_LINT_VERSION) to $(GOPATH)/bin..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	fi
	@echo "golangci-lint installation complete"

# Install gotestsum
.PHONY: install-gotestsum
install-gotestsum:
	@echo "Installing gotestsum..."
	@if ! command -v gotestsum >/dev/null 2>&1; then \
		echo "gotestsum not found, installing..."; \
		$(GO) install gotest.tools/gotestsum@$(GOTESTSUM_VERSION); \
	else \
		echo "gotestsum is already installed"; \
	fi

# Install gocover-cobertura
.PHONY: install-gocover-cobertura
install-gocover-cobertura:
	@echo "Installing gocover-cobertura..."
	@if ! command -v gocover-cobertura >/dev/null 2>&1; then \
		echo "gocover-cobertura not found, installing..."; \
		$(GO) install github.com/boumenot/gocover-cobertura@$(GOCOVER_COBERTURA_VERSION); \
	else \
		echo "gocover-cobertura is already installed"; \
	fi

# Install Go $(GO_VERSION) for CI environments (Linux and macOS, amd64 and arm64)
.PHONY: install-go-ci
install-go-ci: ## Install Go $(GO_VERSION) for CI environments (Linux/macOS, amd64/arm64)
	@echo "Installing Go $(GO_VERSION) for CI..."
	@# Detect platform and architecture
	@OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	ARCH=$$(uname -m); \
	case "$$ARCH" in \
		x86_64) ARCH=amd64 ;; \
		aarch64|arm64) ARCH=arm64 ;; \
		*) echo "Unsupported architecture: $$ARCH" && exit 1 ;; \
	esac; \
	echo "Detected platform: $$OS-$$ARCH"; \
	\
	if command -v go >/dev/null 2>&1; then \
		current_version=$$(go version | grep -o 'go[0-9]\+\.[0-9]\+\.[0-9]\+' | sed 's/go//'); \
		if [ "$$current_version" = "$(GO_VERSION)" ]; then \
			echo "Go $(GO_VERSION) is already installed"; \
			echo "Location: $$(which go)"; \
			go version; \
			exit 0; \
		else \
			echo "Current Go version: $$current_version, installing $(GO_VERSION)..."; \
		fi; \
	else \
		echo "Go not found, installing $(GO_VERSION)..."; \
	fi; \
	\
	GO_TARBALL="go$(GO_VERSION).$$OS-$$ARCH.tar.gz"; \
	GO_URL="https://go.dev/dl/$$GO_TARBALL"; \
	echo "Downloading $$GO_URL..."; \
	\
	if command -v curl >/dev/null 2>&1; then \
		if ! curl -fsSL "$$GO_URL" -o "$$GO_TARBALL"; then \
			echo "Failed to download Go tarball from $$GO_URL"; \
			exit 1; \
		fi; \
	elif command -v wget >/dev/null 2>&1; then \
		if ! wget --show-progress "$$GO_URL"; then \
			echo "Failed to download Go tarball from $$GO_URL"; \
			exit 1; \
		fi; \
	else \
		echo "Neither curl nor wget found. Please install one of them."; \
		exit 1; \
	fi; \
	\
	echo "Extracting Go $(GO_VERSION)..."; \
	if [ "$$OS" = "darwin" ]; then \
		rm -rf /usr/local/go 2>/dev/null || true; \
		tar -C /usr/local -xzf "$$GO_TARBALL"; \
		echo "Go $(GO_VERSION) installed to /usr/local/go"; \
		echo "Add /usr/local/go/bin to your PATH if not already present"; \
	else \
		rm -rf /usr/local/go 2>/dev/null || true; \
		tar -C /usr/local -xzf "$$GO_TARBALL"; \
		echo "Go $(GO_VERSION) installed to /usr/local/go"; \
		echo "Add /usr/local/go/bin to your PATH if not already present"; \
	fi; \
	\
	rm -f "$$GO_TARBALL"; \
	echo "Installation complete"; \
	\
	if [ -x /usr/local/go/bin/go ]; then \
		echo "Installed Go version: $$(/usr/local/go/bin/go version)"; \
		echo "Location: /usr/local/go/bin/go"; \
	else \
		echo "Warning: Go binary not found at expected location /usr/local/go/bin/go"; \
	fi

# Lint and test all modules (delegates to sub-Makefiles)
.PHONY: lint-test-all
lint-test-all: protos-lint license-headers-lint gomod-lint health-monitors-lint-test-all go-lint-test-all python-lint-test-all kubernetes-distro-lint log-collector-lint gpu-reset-lint ## Lint and test all modules

# Generate protobuf files
.PHONY: protos-generate
protos-generate: protos-clean ## Generate protobuf files from .proto sources
	@echo "Generating protobuf files in data-models (Go) and gpu-health-monitor (Python)..."
	@echo "=== Tool Versions ==="
	@echo "Go: $$(go version)"
	@echo "protoc: $$(protoc --version)"
	@echo "protoc-gen-go: $$(protoc-gen-go --version)"
	@echo "protoc-gen-go-grpc: $$(protoc-gen-go-grpc --version)"
	@if command -v python3 >/dev/null 2>&1; then \
		grpcio_tools_version=$$(python3 -c "import importlib.metadata; print('grpcio-tools', importlib.metadata.version('grpcio-tools'))" 2>/dev/null || echo "grpcio-tools: not installed"); \
		echo "$$grpcio_tools_version"; \
	fi
	@if command -v black >/dev/null 2>&1; then \
		black_version=$$(black --version 2>/dev/null | head -1 || echo "black: not available"); \
		echo "$$black_version"; \
	else \
		echo "black: not found"; \
	fi
	@echo "========================"
	# Generate Go protobuf files in data-models (shared by all Go modules)
	$(MAKE) -C data-models protos-generate
	# Generate Go protobuf files in api
	$(MAKE) -C api protos-generate
	# Generate Python protobuf files for gpu-health-monitor
	$(MAKE) -C health-monitors/gpu-health-monitor protos-generate
	# Generate Python protobuf files for dcgm-diag preflight check
	$(MAKE) -C preflight-checks/dcgm-diag protos-generate
	# Generate Python protobuf files for nccl-allreduce preflight check
	$(MAKE) -C preflight-checks/nccl-allreduce protos-generate

# Check protobuf files
.PHONY: protos-lint
protos-lint: protos-generate ## Generate and check protobuf files are up to date
	@echo "Checking protobuf files..."
	git status --porcelain --untracked-files=no
	git --no-pager diff
	@echo "Checking if protobuf files are up to date..."
	test -z "$$(git status --porcelain --untracked-files=no)"

# Clean generated protobuf files
.PHONY: protos-clean
protos-clean: ## Remove all generated protobuf files
	@echo "Cleaning generated protobuf files..."
	@echo "Removing Go protobuf files (.pb.go)..."
	find . -name "*.pb.go" -type f -delete
	@echo "Removing Python protobuf files (*_pb2.py, *_pb2_grpc.py, *_pb2.pyi)..."
	find . \( -name "*_pb2.py" -o -name "*_pb2_grpc.py" -o -name "*_pb2.pyi" \) -type f -delete
	@echo "All generated protobuf files have been removed."

# Check license headers
.PHONY: license-headers-lint
license-headers-lint: ## Check license headers in source files
	@echo "Checking license headers..."
	addlicense -f .github/headers/LICENSE -check \
		-ignore '.venv/**' \
		-ignore '**/__pycache__/**' \
		-ignore '**/.venv/**' \
		-ignore '**/site-packages/**' \
		-ignore '*/.venv/**' \
		-ignore '**/.idea/**' \
		-ignore '**/*.csv' \
		-ignore '**/*.pyc' \
		-ignore '**/*.txt' \
		-ignore '**/*.xml' \
		-ignore '**/*.yaml' \
		-ignore '**/*.toml' \
		-ignore '**/*lock.hcl' \
		-ignore '**/*pb2*' \
		.

# Check go.mod files for proper replace directives
.PHONY: gomod-lint
gomod-lint: ## Validate go.mod files for local module replace directives
	@echo "Validating go.mod files for local module replace directives..."
	./scripts/validate-gomod.sh

# Sync dependencies across all go modules using Go workspace
.PHONY: dependencies-sync
dependencies-sync: dependencies-update go-mod-tidy-all ## Sync dependencies across all Go modules using workspace
	@echo "go.mod and go.sum updated and synced successfully across all go modules"

# Update dependencies across all go modules using Go workspace
.PHONY: dependencies-update
dependencies-update:
	@echo "Updating dependencies across all Go modules..."
	rm go.work >/dev/null 2>&1 || true
	find . -name "go.mod" | awk -F/go.mod '{print $$1}' | xargs go work init
	go work sync
	rm go.work go.work.sum >/dev/null 2>&1 || true
	@echo "Dependencies updated successfully"

# Sync dependencies and lint to ensure no files were modified
.PHONY: dependencies-sync-lint
dependencies-sync-lint: dependencies-sync ## Sync dependencies and verify no files were modified
	@echo "Checking if dependency sync modified any files..."
	git status --porcelain --untracked-files=no
	git --no-pager diff
	@echo "Verifying that dependency sync didn't modify any files..."
	test -z "$$(git status --porcelain --untracked-files=no)"

# Run go mod tidy in all directories with go.mod files
.PHONY: go-mod-tidy-all
go-mod-tidy-all: ## Run go mod tidy in all directories with go.mod files
	@echo "Running go mod tidy in all directories with go.mod files..."
	@find . -name "go.mod" -type f | while read -r gomod_file; do \
		dir=$$(dirname "$$gomod_file"); \
		echo "Running go mod tidy in $$dir..."; \
		(cd "$$dir" && go mod tidy) || exit 1; \
	done
	@echo "go mod tidy completed in all modules"

# Lint and test non-health-monitor Go modules
.PHONY: go-lint-test-all
go-lint-test-all:
	@echo "Running lint and tests for non-health-monitor Go modules..."
	@for module in $(shell echo "$(GO_MODULES)" | tr ' ' '\n' | grep -v health-monitors); do \
		echo "Processing $$module..."; \
		$(MAKE) lint-test-$$module || exit 1; \
	done





.PHONY: lint-test-fault-remediation
lint-test-fault-remediation:
	@echo "Linting and testing fault-remediation (using standardized Makefile)..."
	$(MAKE) -C fault-remediation lint-test

.PHONY: lint-test-janitor
lint-test-janitor:
	@echo "Linting and testing janitor (using standardized Makefile)..."
	$(MAKE) -C janitor lint-test

.PHONY: lint-test-store-client
lint-test-store-client:
	@echo "Linting and testing store-client..."
	$(MAKE) -C store-client lint-test

.PHONY: lint-test-commons
lint-test-commons:
	@echo "Linting and testing commons..."
	$(MAKE) -C commons lint-test

.PHONY: lint-test-metadata-collector
lint-test-metadata-collector:
	@echo "Linting and testing metadata-collector..."
	$(MAKE) -C metadata-collector lint-test

# Log collector lint (shell script)
.PHONY: log-collector-lint
log-collector-lint: ## Lint shell scripts in log collector
	@echo "Linting log collector shell scripts..."
	$(MAKE) -C log-collector lint

# Build targets (delegate to sub-Makefiles for better organization)
.PHONY: build-all
build-all: build-health-monitors build-main-modules ## Build all modules

# Individual build targets for non-health-monitor modules
define make-build-target
.PHONY: build-$(1)
build-$(1):
	@echo "Building $(1)..."
	cd $(1) && $(GO) build ./...
endef

$(foreach module,$(shell echo "$(GO_MODULES)" | tr ' ' '\n' | grep -v health-monitors),$(eval $(call make-build-target,$(module))))


# Clean targets (delegate to sub-Makefiles for better organization)
.PHONY: clean-all
clean-all: clean-health-monitors clean-main-modules ## Clean all modules

# ============================================================================
# Docker 
# ============================================================================
# Docker targets (delegate to docker/Makefile) - standardized build system
.PHONY: docker-all
docker-all: ## Build all Docker images
	@echo "Building all Docker images..."
	$(MAKE) -C docker build-all

.PHONY: docker-publish-all
docker-publish-all: ## Build and publish all Docker images
	@echo "Building and publishing all Docker images..."
	$(MAKE) -C docker publish-all

.PHONY: docker-setup-buildx
docker-setup-buildx: ## Setup Docker buildx builder
	$(MAKE) -C docker setup-buildx

# Ko targets - build Go container images without Docker
.PHONY: ko-build
ko-build: ## Build all ko-based container images locally
	@echo "Building all ko-based container images..."
	@./scripts/buildko.sh

.PHONY: ko-publish
ko-publish: ## Build and publish all ko-based container images
	@echo "Building and publishing all ko-based container images..."
	@./scripts/buildko.sh

.PHONY: docker-gpu-health-monitor
docker-gpu-health-monitor:
	$(MAKE) -C docker build-gpu-health-monitor

# Individual module Docker targets
.PHONY: docker-syslog-health-monitor
docker-syslog-health-monitor:
	$(MAKE) -C docker build-syslog-health-monitor

#==============================================================================
# PostgreSQL Schema Management
#==============================================================================
# .PHONY: validate-postgres-schema
# validate-postgres-schema: ## Validate PostgreSQL schema consistency
# 	@./scripts/validate-postgres-schema.sh

# .PHONY: update-helm-postgres-schema
# update-postgres-schema: ## Update Helm values file with schema from docs/postgresql-schema.sql
# 	@./scripts/update-helm-postgres-schema.sh

#==============================================================================
# Make File-engine
#==============================================================================
# Run linter on the code (local version)
.PHONY: lint-local
lint-local:
	golangci-lint --version
	cd file-engine && golangci-lint run --fix --verbose


#==============================================================================
# Make Backend
#==============================================================================

.PHONY: lint-backend
lint-backend: test-tools-image
	$(call run-in-test-client,make lint-backend-local)

.PHONY: lint-backend-local
lint-backend-local:
	cd backend && composer lint


#==============================================================================
# Make Frontend 
#==============================================================================

#==============================================================================
# Make Tree
# Generate and optionally publish a project directory tree.
#==============================================================================

TREE_OUT      := docs/tree.md
TREE_FLAGS    := --gitignore

.PHONY: tree tree-check tree-open tree-commit tree-push tree-publish

## Generate $(TREE_OUT) with the current project directory structure.
tree:
	@echo "Generating $(TREE_OUT)..."
	@tree $(TREE_FLAGS) > $(TREE_OUT)
	@echo "Wrote $(TREE_OUT)"

## Fail if $(TREE_OUT) is out-of-date relative to the current tree output.
tree-check:
	@echo "Checking whether $(TREE_OUT) is up-to-date..."
	@tmp=$$(mktemp); \
	tree $(TREE_FLAGS) > $$tmp; \
	if cmp -s $$tmp $(TREE_OUT); then \
		echo "$(TREE_OUT) is up-to-date."; \
		rm -f $$tmp; \
	else \
		echo "$(TREE_OUT) is out-of-date. Run: make tree"; \
		rm -f $$tmp; \
		exit 1; \
	fi

## Open $(TREE_OUT) in a pager.
tree-open:
	@less -R $(TREE_OUT)

## Commit $(TREE_OUT) (if it changed). Usage: make tree-commit MSG="Update tree"
tree-commit:
	@msg="$${MSG:-Update project tree}"; \
	git add $(TREE_OUT); \
	if git diff --cached --quiet; then \
		echo "No changes to commit for $(TREE_OUT)."; \
	else \
		git commit -m "$$msg"; \
	fi

## Push current branch to origin.
tree-push:
	@git push

## One-shot: regenerate tree, commit it, and push it.
## Usage: make tree-publish MSG="Update tree"
tree-publish: tree tree-commit tree-push
