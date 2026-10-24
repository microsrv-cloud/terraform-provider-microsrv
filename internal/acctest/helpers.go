// Package acctest provides shared helpers for acceptance tests.
// All acceptance tests require TF_ACC=1 and MICROSRV_TOKEN (or MICROSRV_API_KEY).
package acctest

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"golang.org/x/crypto/ssh"

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
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// RandSSHPublicKey generates a cryptographically fresh Ed25519 SSH public key.
// Generating a new key per test avoids 409 conflict errors from duplicate key
// fingerprints on the user account across parallel or rerun tests.
func RandSSHPublicKey() string {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("failed to generate ed25519 key: %v", err))
	}
	k, err := ssh.NewPublicKey(pub)
	if err != nil {
		panic(fmt.Sprintf("failed to format ssh public key: %v", err))
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(k)))
}

// Legacy static key kept for backwards compatibility if needed.
const TestSSHPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIE7pFNt09gFCjm8g1eF9zl+sCVYfMC49P/J98BBWKZ8P tf-acc-test-key"

// ProviderConfig returns the minimal HCL provider block.
// Credentials are taken from env vars (MICROSRV_TOKEN / MICROSRV_API_KEY).
const ProviderConfig = `
provider "microsrv" {}
`
