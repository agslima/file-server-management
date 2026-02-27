# Backend Makefile
# Container-only module build targets

# Repository root path calculation (works from any subdirectory depth)
REPO_ROOT := $(shell git rev-parse --show-toplevel)

# Auto-detect current module path relative to repo root
MODULE_PATH := $(subst $(REPO_ROOT)/,,$(CURDIR))

# Docker configuration (set from environment in CI)
CONTAINER_REGISTRY ?= ghcr.io
CONTAINER_ORG ?= agslima
CI_COMMIT_REF_NAME ?= $(shell git rev-parse --abbrev-ref HEAD)
SAFE_REF_NAME ?= $(shell echo $(CI_COMMIT_REF_NAME) | sed 's/\//-/g')
SAFE_REF_NAME := $(if $(SAFE_REF_NAME),$(SAFE_REF_NAME),local)
BUILDX_BUILDER ?= file-server-management-builder
PLATFORMS ?= linux/arm64,linux/amd64
DOCKER_LOAD_ARG ?= --load

# Cache configuration (can be disabled via environment variables)
DISABLE_REGISTRY_CACHE ?= false
ifeq ($(DISABLE_REGISTRY_CACHE),true)
CACHE_FROM_ARG_LOG :=
CACHE_TO_ARG_LOG :=
CACHE_FROM_ARG_CLEANUP :=
CACHE_TO_ARG_CLEANUP :=
else
CACHE_FROM_ARG_LOG := --cache-from=type=registry,ref=$(CONTAINER_REGISTRY)/$(CONTAINER_ORG)/file-server-management-buildcache:log-collector
CACHE_TO_ARG_LOG := --cache-to=type=registry,ref=$(CONTAINER_REGISTRY)/$(CONTAINER_ORG)/file-server-management-buildcache:log-collector,mode=max
CACHE_FROM_ARG_CLEANUP := --cache-from=type=registry,ref=$(CONTAINER_REGISTRY)/$(CONTAINER_ORG)/file-server-management-buildcache:file-server-cleanup
CACHE_TO_ARG_CLEANUP := --cache-to=type=registry,ref=$(CONTAINER_REGISTRY)/$(CONTAINER_ORG)/file-server-management-buildcache:file-server-cleanup,mode=max
endif

# Default target
.PHONY: all
all: lint

# Lint and test this module (matches parent Makefile expectations)
.PHONY: lint-test
lint-test: lint

# Lint shell scripts and PHP code
.PHONY: lint
lint: lint-shell lint-php

# Lint only shell scripts
.PHONY: lint-shell-backend
lint-backend: lint-shell

# Lint shell scripts
.PHONY: lint-shell
lint-shell:
	@echo "Linting backend shell scripts..."
	shellcheck backend/scripts/smoke.sh

# Lint PHO code using Pint + PHPStan
.PHONY: lint-php
lint-php:
	@echo "Linting Laravel/PHP code (Pint + PHPStan)..."
	composer install --no-interaction --prefer-dist --no-progress
	./vendor/bin/pint --test
	./vendor/bin/phpstan analyse

# Format PHP code using Pint
.PHONY: format-php
format-php:
	@echo "Formatting Laravel/PHP code (Pint)..."
	composer install --no-interaction --prefer-dist --no-progress
	./vendor/bin/pint

# Setup PHP environment
.PHONY: setup-php
setup-php:
	@echo "Setting up Composer dependencies..."
	composer install --no-interaction --prefer-dist --no-progress

# Setup buildx builder
.PHONY: setup-buildx
setup-buildx:
	@echo "Setting up Docker buildx builder..."
	@docker buildx inspect $(BUILDX_BUILDER) >/dev/null 2>&1 || \
		(docker context create $(BUILDX_BUILDER)-context || true && \
		 docker buildx create --name $(BUILDX_BUILDER) $(BUILDX_BUILDER)-context --driver docker-container --driver-opt network=host --buildkitd-flags '--allow-insecure-entitlement network.host' && \
		 docker buildx use $(BUILDX_BUILDER) && \
		 docker buildx inspect --bootstrap)

# Docker build target (for local development)
.PHONY: docker-build
docker-build: setup-buildx docker-build-log-collector docker-build-file-server-cleanup

# Build backend image (local development)
.PHONY: docker-build-backend
docker-build-backend: setup-buildx
	@echo "Building Docker image for backend (local development)..."
	$(if $(filter true,$(DISABLE_REGISTRY_CACHE)),@echo "Registry cache disabled for this build")
	cd $(REPO_ROOT) && docker buildx build \
		--platform $(PLATFORMS) \
		--network=host \
		$(CACHE_FROM_ARG_LOG) \
		$(CACHE_TO_ARG_LOG) \
		$(DOCKER_LOAD_ARG) \
		-f $(MODULE_PATH)/Dockerfile \
		-t $(CONTAINER_REGISTRY)/$(CONTAINER_ORG)/file-server-management/backend:$(SAFE_REF_NAME) \
		.

# Local-only Docker build (no remote cache, faster for local development)
.PHONY: docker-build-local
docker-build-local: setup-buildx docker-build-local-backend

# Build backend image (local only, no remote cache)
.PHONY: docker-build-local-backend
docker-build-local-backend: setup-buildx
	@echo "Building Docker image for backend (local, no remote cache)..."
	docker buildx build \
		--platform $(PLATFORMS) \
		--network=host \
		$(DOCKER_LOAD_ARG) \
		-t log-collector:local \
		.

# Docker publish target
.PHONY: docker-publish
docker-publish: setup-buildx docker-publish-backend

# Build and publish log-collector image
.PHONY: docker-publish-backend
docker-publish-backend: setup-buildx
	@echo "Building and publishing Docker image for backend (production)..."
	$(if $(filter true,$(DISABLE_REGISTRY_CACHE)),@echo "Registry cache disabled for this build")
	cd $(REPO_ROOT) && docker buildx build \
		--platform $(PLATFORMS) \
		--network=host \
		$(CACHE_FROM_ARG_LOG) \
		$(CACHE_TO_ARG_LOG) \
		$(if $(DOCKER_METADATA_FILE),--metadata-file $(DOCKER_METADATA_FILE)) \
		--push \
		-f $(MODULE_PATH)/Dockerfile \
		-t $(CONTAINER_REGISTRY)/$(CONTAINER_ORG)/file-server-management/backend:$(SAFE_REF_NAME) \
		.

# Legacy targets for backwards compatibility
.PHONY: image
image: docker-build
	@echo "Legacy 'image' target - use 'docker-build' for local development or 'docker-publish' for CI/production"

.PHONY: publish
publish: docker-publish
	@echo "Legacy 'publish' target - use 'docker-publish' for CI/production"

# Clean target
.PHONY: clean
clean:
	@echo "Cleaning log-collector..."
	@echo "No build artifacts to clean for shell-based module"

# Help target
.PHONY: help
help:
	@echo "File Server Management Makefile - Standardized build system"
	@echo ""
	@echo "Configuration (environment variables):"
	@echo "  CONTAINER_REGISTRY=$(CONTAINER_REGISTRY)"
	@echo "  CONTAINER_ORG=$(CONTAINER_ORG)"
	@echo "  SAFE_REF_NAME=$(SAFE_REF_NAME)"
	@echo "  PLATFORMS=$(PLATFORMS)"
	@echo ""
	@echo "Main targets:"
	@echo "  all                        - Run lint (default)"
	@echo "  lint-test                  - Run all linting (matches CI)"
	@echo "  lint                       - Run all linting (shell + php)"
	@echo "  lint-backend.              - Run linting for backend only (shell scripts)"
    @echo "  lint-shell                 - Run shellcheck on shell scripts"
	@echo "  lint-php                - Run Black check on PHP code"
	@echo "  format-php              - Format PHP code with Black"
	@echo "  setup-php               - Setup Poetry environment"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build                      - Build both images (local development)"
	@echo "  docker-build-local                - Build both images (local only, no remote cache)"
	@echo "  docker-build-backend.             - Build backend image only (local)"
	@echo "  docker-publish                    - Build and publish both images (CI/production)"
	@echo "  docker-publish-backend            - Build and publish backend image only"
	@echo "  setup-buildx                      - Setup Docker buildx builder"
	@echo "  image                             - Legacy target (calls docker-build)"
	@echo "  publish                           - Legacy target (calls docker-publish)"
	@echo ""
	@echo "Utility targets:"
	@echo "  clean      - Clean build artifacts (no-op for shell module)"
	@echo "  help       - Show this help message"
	@echo ""
	@echo "Notes:"
	@echo "  - This is a container-only module using shell scripts"
	@echo "  - Multi-platform builds: $(PLATFORMS)"
	@echo "  - Images tagged with dynamic ref name: $(SAFE_REF_NAME)"
	@echo "  - Build cache is used for faster builds"
	@echo "  - Multi-platform builds with build caching"
