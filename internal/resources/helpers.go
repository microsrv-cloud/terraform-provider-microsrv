package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/client"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/providerdata"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/wait"
)

type growOnlySizeModifier struct{}

func (m growOnlySizeModifier) Description(context.Context) string {
	return "Volume size cannot be decreased (volumes are grow-only)."
}

func (m growOnlySizeModifier) MarkdownDescription(context.Context) string {
	return "Volume size cannot be decreased (volumes are grow-only)."
}

func (m growOnlySizeModifier) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if req.PlanValue.ValueInt64() < req.StateValue.ValueInt64() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Cannot shrink volume",
			fmt.Sprintf("Volume size cannot be decreased from %d MiB to %d MiB (grow-only). To shrink a volume, create a new volume and copy data.", req.StateValue.ValueInt64(), req.PlanValue.ValueInt64()),
		)
	}
}

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
		diags.AddError("Provider not configured", "The provider must be configured before using this resource.")
		return false
	}
	return true
}

func apiErrDiagnostics(prefix string, err error, diags *diag.Diagnostics) {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		diags.AddError(prefix, apiErr.Error())
		return
	}
	diags.AddError(prefix, err.Error())
}

// withLogs appends a best-effort log tail to a wait-failure error so the
// diagnostic shows WHY the resource never became ready (e.g. a crashing
// app's fatal log lines) instead of a bare state timeout.
func withLogs(err error, logs string) error {
	logs = strings.TrimSpace(logs)
	if logs == "" {
		return err
	}
	if len(logs) > 4000 {
		logs = logs[:4000] + "\n... (truncated)"
	}
	return fmt.Errorf("%w\n\n--- logs (tail) ---\n%s", err, logs)
}

func notFoundRemove(err error) bool {
	var apiErr *client.APIError
	return errors.As(err, &apiErr) && apiErr.IsNotFound()
}

func rawJSON(b []byte) string {
	if len(b) == 0 || string(b) == "null" {
		return ""
	}
	return string(b)
}

// rollbackAfterWaitFailure best-effort deletes a resource that was created in
// this Create call but failed to reach READY. Without this the resource stays
// on the platform while missing from Terraform state, and (because it still holds
// attached volumes/NIs) blocks subsequent apply/destroy with 409 "in use".
func rollbackAfterWaitFailure(ctx context.Context, meta *providerdata.Meta, id string, del func(context.Context, string) error, get func(context.Context, string) (string, error)) {
	if err := del(ctx, id); err != nil {
		return
	}
	_ = wait.UntilGone(ctx, meta.Wait, func(ctx context.Context) (string, error) {
		state, err := get(ctx, id)
		if err != nil {
			return "", err
		}
		return state, nil
	})
}
