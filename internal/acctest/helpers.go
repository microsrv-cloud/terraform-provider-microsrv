// Package acctest provides shared helpers for acceptance tests.
// All acceptance tests require TF_ACC=1 and MICROSRV_TOKEN (or MICROSRV_API_KEY).
package acctest

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/provider"
)

// ProviderFactories is an in-process provider factory used by every TestCase.
var ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"microsrv": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// SkipIfNoAccEnv skips the test when TF_ACC is unset or when no auth token is available.
func SkipIfNoAccEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	if os.Getenv("MICROSRV_TOKEN") == "" && os.Getenv("MICROSRV_API_KEY") == "" {
		t.Fatal("MICROSRV_TOKEN or MICROSRV_API_KEY must be set for acceptance tests")
	}
}

// RandSuffix returns a short random hex suffix for unique resource names.
func RandSuffix() string {
	return fmt.Sprintf("%06x", rand.Int63n(1<<24)) //nolint:gosec
}

// TestSSHPublicKey is a static Ed25519 public key used as the SSH key in every
// acceptance test.  Using a fixed key eliminates dependency on the CI runner's
// ~/.ssh directory and keeps test configs self-contained.
const TestSSHPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIE7pFNt09gFCjm8g1eF9zl+sCVYfMC49P/J98BBWKZ8P tf-acc-test-key"

// ProviderConfig returns the minimal HCL provider block.
// Credentials are taken from env vars (MICROSRV_TOKEN / MICROSRV_API_KEY).
const ProviderConfig = `
provider "microsrv" {}
`
