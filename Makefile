.PHONY: build test tidy install lint package testacc testacc-fast sweep

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

# Run the full acceptance test suite against the real Microsrv API.
# Requires: MICROSRV_TOKEN (or MICROSRV_API_KEY) environment variable.
testacc:
	TF_ACC=1 go test ./internal/... -run "TestAcc" -v -timeout 80m

# Run only the fast subset (datasources + cheap resources, ~5 min).
testacc-fast:
	TF_ACC=1 go test ./internal/... \
		-run "TestAcc(DataRegion|DataFlavor|DataImage|DataQuota|DataPricing|DataProject|Project_|SSHKey_|VPC_|Volume_blank|NetworkInterface_)" \
		-v -timeout 15m

# Sweep and delete orphaned acceptance test resources (prefix: tf-acc-).
# Requires: MICROSRV_TOKEN (or MICROSRV_API_KEY) environment variable.
sweep:
	go run ./scripts/sweep.go

