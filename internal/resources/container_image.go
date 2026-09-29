package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/client"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/providerdata"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/wait"
)

var _ resource.Resource = (*containerImageResource)(nil)
var _ resource.ResourceWithConfigure = (*containerImageResource)(nil)
var _ resource.ResourceWithImportState = (*containerImageResource)(nil)

func NewContainerImageResource() resource.Resource { return &containerImageResource{} }

type containerImageResource struct {
	meta *providerdata.Meta
}

type containerImageModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	RegionID  types.String `tfsdk:"region_id"`
	Ref       types.String `tfsdk:"ref"`
	State     types.String `tfsdk:"state"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *containerImageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_image"
}

func (r *containerImageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A project-scoped OCI container image. Ref changes require replace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"region_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ref": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"state":      schema.StringAttribute{Computed: true},
			"status":     schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *containerImageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (r *containerImageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan containerImageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.meta.Client.CreateContainerImage(ctx, plan.ProjectID.ValueString(), client.CreateContainerImageRequest{
		RegionID: plan.RegionID.ValueString(),
		Ref:      plan.Ref.ValueString(),
	})
	if err != nil {
		apiErrDiagnostics("Create container image failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetContainerImage(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		rollbackAfterWaitFailure(ctx, r.meta, out.ID, r.meta.Client.DeleteContainerImage, func(ctx context.Context, id string) (string, error) {
			v, e := r.meta.Client.GetContainerImage(ctx, id)
			if e != nil {
				return "", e
			}
			return v.State, nil
		})
		apiErrDiagnostics("Wait container image READY failed", err, &resp.Diagnostics)
		return
	}
	setContainerImageState(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *containerImageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state containerImageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.meta.Client.GetContainerImage(ctx, state.ID.ValueString())
	if err != nil {
		if notFoundRemove(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		apiErrDiagnostics("Read container image failed", err, &resp.Diagnostics)
		return
	}
	setContainerImageState(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *containerImageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "microsrv_container_image has no update API; changes require replace.")
}

func (r *containerImageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state containerImageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if err := r.meta.Client.DeleteContainerImage(ctx, id); err != nil {
		if !notFoundRemove(err) {
			apiErrDiagnostics("Delete container image failed", err, &resp.Diagnostics)
			return
		}
	} else if err := wait.UntilGone(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetContainerImage(ctx, id)
		if err != nil {
			return "", err
		}
		return v.State, nil
	}); err != nil {
		apiErrDiagnostics("Wait container image deleted failed", err, &resp.Diagnostics)
	}
}

func (r *containerImageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func setContainerImageState(m *containerImageModel, out *client.ContainerImage) {
	m.ID = types.StringValue(out.ID)
	m.ProjectID = types.StringValue(out.ProjectID)
	m.RegionID = types.StringValue(out.RegionID)
	m.Ref = types.StringValue(out.Ref)
	m.State = types.StringValue(out.State)
	m.Status = types.StringValue(rawJSON(out.Status))
	m.CreatedAt = types.StringValue(out.CreatedAt)
	m.UpdatedAt = types.StringValue(out.UpdatedAt)
}
