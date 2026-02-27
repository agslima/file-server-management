# Docker build Makefile - Delegation-based approach
# Delegates to individual module Makefiles instead of duplicating build logic
# This ensures single source of truth per module while providing unified interface

# Export all environment variables to delegated make calls
export

# Docker configuration (set from environment in CI)
CONTAINER_REGISTRY ?= ghcr.io
CONTAINER_ORG ?= agslima
CI_COMMIT_REF_NAME ?= $(shell git rev-parse --abbrev-ref HEAD)
SAFE_REF_NAME ?= $(shell echo $(CI_COMMIT_REF_NAME) | sed 's/\//-/g')
SAFE_REF_NAME := $(if $(SAFE_REF_NAME),$(SAFE_REF_NAME),local)

# Docker buildx configuration
BUILDX_BUILDER ?= file-server-management-builder
DOCKER_BUILDKIT ?= 1
BUILDKIT_PROGRESS ?= plain
PLATFORMS ?= linux/arm64,linux/amd64

# Module definitions with their relative paths from docker/ directory
GO_MODULES := \
	health-monitors/syslog-health-monitor

CONTAINER_MODULES := \
	log-collector

ALL_MODULES := $(GO_MODULES) $(CONTAINER_MODULES)

# =============================================================================
# Build Targets (Local Development) - Delegates to individual module Makefiles
# Development workflow - builds for local use
# =============================================================================

# Build all Docker images for local development
.PHONY: build-all
build-all: build-gpu-health-monitor build-syslog-health-monitor build-metadata-collector build-log-collector
	@echo "All Docker images built successfully for local development"

# GPU health monitor (special case with DCGM variants)
.PHONY: build-gpu-health-monitor
build-gpu-health-monitor:
	@echo "Building GPU health monitor (both DCGM variants)..."
	$(MAKE) -C ../health-monitors/gpu-health-monitor image


# Standard Go modules
.PHONY: build-syslog-health-monitor
build-syslog-health-monitor:
	@echo "Building syslog health monitor..."
	$(MAKE) -C ../health-monitors/syslog-health-monitor docker-build

# =============================================================================
# Publish Targets (CI/Production) - Delegates to individual module Makefiles
# These use --push for direct registry push (standardized CI build)
# =============================================================================

# Publish all Docker images to registry (CI-like behavior)
.PHONY: publish-all
publish-all: publish-gpu-health-monitor publish-syslog-health-monitor publish-metadata-collector publish-log-collector
	@echo "All Docker images published successfully to registry"


.PHONY: publish-gpu-health-monitor-dcgm4
publish-gpu-health-monitor-dcgm4:
	@echo "Publishing GPU health monitor with DCGM 4.x..."
	$(MAKE) -C ../health-monitors/gpu-health-monitor docker-publish-dcgm4

# Standard Go modules publish
.PHONY: publish-syslog-health-monitor
publish-syslog-health-monitor:
	@echo "Publishing syslog health monitor..."
	$(MAKE) -C ../health-monitors/syslog-health-monitor docker-publish

# =============================================================================
# Group Targets
# =============================================================================

# Health monitors group
.PHONY: build-health-monitors
build-health-monitors: build-gpu-health-monitor build-syslog-health-monitor
	@echo "All health monitor Docker images built successfully"



# =============================================================================
# Utility Targets
# =============================================================================

# Clean Docker images
.PHONY: clean
clean:
	@echo "Removing nvsentinel Docker images..."
	@docker images --format "{{.Repository}}:{{.Tag}}" | grep "$(CONTAINER_REGISTRY)/$(CONTAINER_ORG)/nvsentinel/" | xargs -r docker rmi || true

# List built images
.PHONY: list
list:
	@echo "nvsentinel Docker images:"
	@docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}" | grep "$(CONTAINER_REGISTRY)/$(CONTAINER_ORG)/nvsentinel/" || echo "No nvsentinel images found"

# Help target
.PHONY: help
help:
	@echo "Docker Build Makefile - Delegation-based Architecture"
	@echo ""
	@echo "This Makefile delegates to individual module Makefiles instead of"
	@echo "duplicating build logic. Each module is the single source of truth"
	@echo "for its own Docker build configuration."
	@echo ""
	@echo "Configuration (environment variables):"
	@echo "  CONTAINER_REGISTRY=$(CONTAINER_REGISTRY)"
	@echo "  CONTAINER_ORG=$(CONTAINER_ORG)"
	@echo "  SAFE_REF_NAME=$(SAFE_REF_NAME)"
	@echo "  PLATFORMS=$(PLATFORMS)"
	@echo ""
	@echo "Build targets (local development):"
	@echo "  build-all              - Build all Docker images locally"
	@echo "  build-health-monitors  - Build all health monitor images"
	@echo "  build-main-modules     - Build main module images (non-health-monitors)"
	@echo ""
	@echo "Publish targets (CI/production with --push):"
	@echo "  publish-all            - Build and push all Docker images to registry"
	@echo "  publish-health-monitors - Build and push health monitor images"
	@echo "  publish-main-modules   - Build and push main module images"
	@echo ""
	@echo "Special GPU health monitor targets:"
	@echo "  build-gpu-health-monitor       - Build both DCGM 3.x and 4.x GPU monitor images"
	@echo "  build-gpu-health-monitor-dcgm3 - Build GPU monitor with DCGM 3.x"
	@echo "  build-gpu-health-monitor-dcgm4 - Build GPU monitor with DCGM 4.x"
	@echo "  publish-gpu-health-monitor     - Publish both DCGM variants"
	@echo ""
	@echo "Individual module targets:"
	@echo "  build-<module-name>    - Build specific module locally"
	@echo "  publish-<module-name>  - Build and push specific module to registry"
	@echo ""
	@echo "Available modules:"
	@echo "  syslog-health-monitor, gpu-health-monitor, log-collector"
	@echo ""
	@echo "Utility targets:"
	@echo "  setup-buildx   - Setup Docker buildx builder"
	@echo "  clean          - Remove all nvsentinel Docker images"
	@echo "  list           - List all nvsentinel Docker images"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Notes:"
	@echo "  - Build targets are for local development"
	@echo "  - Publish targets use --push for direct registry push (standardized build)"
	@echo "  - All builds delegate to individual module Makefiles"
	@echo "  - Each module is the single source of truth for its Docker configuration"
	@echo "  - All builds support multi-platform: $(PLATFORMS)"
