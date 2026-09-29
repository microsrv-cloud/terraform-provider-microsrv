.PHONY: build test tidy install lint package

VERSION ?= dev
BINARY  := terraform-provider-microsrv
OS_ARCH ?= $(shell go env GOOS)_$(shell go env GOARCH)
INSTALL_PATH ?= $(HOME)/.terraform.d/plugins/registry.terraform.io/microsrv-cloud/microsrv/$(VERSION)/$(OS_ARCH)

build:
	go build -ldflags="-X main.version=$(VERSION)" -o bin/$(BINARY) .

test:
	go test ./...

tidy:
	go mod tidy

install: build
	mkdir -p "$(INSTALL_PATH)"
	cp bin/$(BINARY) "$(INSTALL_PATH)/$(BINARY)"

lint:
	golangci-lint run ./...

# Registry-format .zip packages (one per platform) for OCI publishing.
package:
	./scripts/package.sh "$(VERSION)"
