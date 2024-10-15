# Variables
GO := go
PROJECT_NAME := oblik
VERSION := $(shell git describe --tags)
DOCKER_IMAGE := ghcr.io/socialgouv/$(PROJECT_NAME)

# Phony targets
.PHONY: all build build-static preflight deps update-deps docker-build docker-push docker install-lint-tools install-release-tools setup-test-env test next-tag changelog clean lint

# Default target
all: build

# Build the project
build: preflight
	$(GO) build -mod=vendor -ldflags="-X 'github.com/SocialGouv/oblik/pkg/cli.Version=${VERSION}'"

# Build a static binary
build-static: preflight
	CGO_ENABLED=0 $(GO) build -a -installsuffix cgo -mod=vendor -ldflags="-X 'github.com/SocialGouv/oblik/pkg/cli.Version=${VERSION}'" -o $(PROJECT_NAME) .

# Preflight checks
preflight: deps
	$(GO) fmt github.com/SocialGouv/$(PROJECT_NAME)

# Manage dependencies
deps:
	$(GO) mod tidy
	$(GO) mod vendor

# Update dependencies
update-deps:
	$(GO) get -u ./...

# Build Docker image
docker-build:
	docker build . --build-arg=VERSION=$(VERSION) -t $(DOCKER_IMAGE):$(VERSION).dev

# Push Docker image
docker-push:
	docker push $(DOCKER_IMAGE)

# Build and push Docker image
docker: docker-build docker-push

# Install lint tools
install-lint-tools:
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install release tools
install-release-tools:
	$(GO) install github.com/git-chglog/git-chglog/cmd/git-chglog@v0.15.4

# Set up test environment
setup-test-env:
	./tests/kind-with-registry.sh
	./tests/install-dependencies.sh

# Run tests
test:
	./tests/deploy-oblik.sh
	./tests/test-oblik.sh $(ARGS)

# Generate next tag
next-tag:
	@$(GO) run ./scripts/next-tag.go

# Generate changelog
changelog:
	git-chglog --next-tag $(shell make next-tag) -o CHANGELOG.md

# Clean build artifacts
clean:
	rm -f $(PROJECT_NAME)
	$(GO) clean

# Run linter
lint:
	golangci-lint run
