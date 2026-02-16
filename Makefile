.PHONY: help build clean docker docker-push

APP_DIR := app/achat
CMD_PKG := ./cmd/achat

DIST_DIR := dist
BIN_NAME := achat
BIN := $(DIST_DIR)/$(BIN_NAME)

IMAGE ?= achat
TAG ?= dev
PLATFORMS ?= linux/amd64,linux/arm64

help:
	@echo "Targets:"
	@echo "  make build        Build ./dist/achat for host"
	@echo "  make clean        Remove dist output"
	@echo "  make docker       Build docker image ($(IMAGE):$(TAG))"
	@echo "  make docker-push  Buildx multi-arch push (requires IMAGE_REPO)"
	@echo ""
	@echo "Vars:"
	@echo "  TAG=...           Image tag (default: dev)"
	@echo "  IMAGE=...         Local image name for make docker (default: achat)"
	@echo "  IMAGE_REPO=...    Remote repo for make docker-push (e.g. ghcr.io/you/achat)"
	@echo "  PLATFORMS=...     Buildx platforms (default: linux/amd64,linux/arm64)"

build:
	@mkdir -p "$(DIST_DIR)"
	@echo "==> building $(BIN)"
	@cd "$(APP_DIR)" && go build -trimpath -o "../..//$(BIN)" "$(CMD_PKG)"
	@echo "==> done: $(BIN)"

clean:
	@rm -rf "$(DIST_DIR)"

docker:
	@echo "==> docker build $(IMAGE):$(TAG)"
	@docker build -t "$(IMAGE):$(TAG)" -f Dockerfile .

docker-push:
	@if [ -z "$(IMAGE_REPO)" ]; then \
		echo "ERROR: IMAGE_REPO is required (e.g. IMAGE_REPO=ghcr.io/you/achat)" >&2; \
		exit 1; \
	fi
	@echo "==> docker buildx build --platform $(PLATFORMS) -t $(IMAGE_REPO):$(TAG) --push"
	@docker buildx build --platform "$(PLATFORMS)" -t "$(IMAGE_REPO):$(TAG)" --push -f Dockerfile .
