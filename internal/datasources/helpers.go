package datasources

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/providerdata"
)

func castMeta(data any, diags *diag.Diagnostics) *providerdata.Meta {
	// ProviderData is nil during Validate*Config before provider Configure runs.
	if data == nil {
		return nil
	}
	m, ok := data.(*providerdata.Meta)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("Expected *providerdata.Meta, got %T", data))
		return nil
	}
	return m
}

func requireMeta(meta *providerdata.Meta, diags *diag.Diagnostics) bool {
	if meta == nil {
		diags.AddError("Provider not configured", "The provider must be configured before using this data source.")
		return false
	}
	return true
}

// boolOrNull maps a *bool to a types.Bool, keeping the attribute null
// (unknown) when the API did not annotate it.
func boolOrNull(v *bool) types.Bool {
	if v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}
