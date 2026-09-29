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

var _ resource.Resource = (*volumeResource)(nil)
var _ resource.ResourceWithConfigure = (*volumeResource)(nil)
var _ resource.ResourceWithImportState = (*volumeResource)(nil)

func NewVolumeResource() resource.Resource { return &volumeResource{} }

type volumeResource struct {
	meta *providerdata.Meta
}

type volumeModel struct {
	ID         types.String `tfsdk:"id"`
	ProjectID  types.String `tfsdk:"project_id"`
	RegionID   types.String `tfsdk:"region_id"`
	SizeMiB    types.Int64  `tfsdk:"size_mib"`
	Bootable   types.Bool   `tfsdk:"bootable"`
	CloneFrom  types.String `tfsdk:"clone_from"`
	AttachedTo types.String `tfsdk:"attached_to"`
	State      types.String `tfsdk:"state"`
	Spec       types.String `tfsdk:"spec"`
	Status     types.String `tfsdk:"status"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func (r *volumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (r *volumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Microsrv volume (512 MiB quantum; ≤8 attached per compute). Set `clone_from` (an image code) to create a bootable boot disk for a `microsrv_vm` referenced via `boot_volume_id`; without it the volume is a blank data disk. Immutable after create. `attached_to` shows owning vm/container (`vm/<id>` or `container/<id>`) when attached.",
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
			"size_mib": schema.Int64Attribute{
				MarkdownDescription: "Volume size in MiB (512 MiB quantum; updatable in place, grow-only).",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{growOnlySizeModifier{}},
			},
			"bootable": schema.BoolAttribute{Computed: true},
			"clone_from": schema.StringAttribute{
				MarkdownDescription: "Image code making this a bootable boot disk (implied `bootable=true`). Immutable; requires replace.",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"attached_to": schema.StringAttribute{Computed: true},
			"state":       schema.StringAttribute{Computed: true},
			"spec":        schema.StringAttribute{Computed: true},
			"status":      schema.StringAttribute{Computed: true},
			"created_at":  schema.StringAttribute{Computed: true},
			"updated_at":  schema.StringAttribute{Computed: true},
		},
	}
}

func (r *volumeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (r *volumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan volumeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cloneFrom := plan.CloneFrom.ValueString()
	out, err := r.meta.Client.CreateVolume(ctx, plan.ProjectID.ValueString(), client.CreateVolumeRequest{
		RegionID:  plan.RegionID.ValueString(),
		SizeMiB:   plan.SizeMiB.ValueInt64(),
		CloneFrom: cloneFrom,
		// The API requires clone_from and bootable together (bootable without a
		// source image is rejected); derive it so the pair can never diverge.
		Bootable: cloneFrom != "",
	})
	if err != nil {
		apiErrDiagnostics("Create volume failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetVolume(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		rollbackAfterWaitFailure(ctx, r.meta, out.ID, r.meta.Client.DeleteVolume, func(ctx context.Context, id string) (string, error) {
			v, e := r.meta.Client.GetVolume(ctx, id)
			if e != nil {
				return "", e
			}
			return v.State, nil
		})
		apiErrDiagnostics("Wait volume READY failed", err, &resp.Diagnostics)
		return
	}
	setVolumeState(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state volumeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.meta.Client.GetVolume(ctx, state.ID.ValueString())
	if err != nil {
		if notFoundRemove(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		apiErrDiagnostics("Read volume failed", err, &resp.Diagnostics)
		return
	}
	setVolumeState(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *volumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan volumeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.meta.Client.PatchVolume(ctx, plan.ID.ValueString(), client.PatchVolumeRequest{
		SizeMiB: plan.SizeMiB.ValueInt64(),
	})
	if err != nil {
		apiErrDiagnostics("Update volume failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetVolume(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		apiErrDiagnostics("Wait volume READY failed", err, &resp.Diagnostics)
		return
	}
	setVolumeState(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state volumeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if err := r.meta.Client.DeleteVolume(ctx, id); err != nil {
		if notFoundRemove(err) {
			return
		}
		apiErrDiagnostics("Delete volume failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilGone(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetVolume(ctx, id)
		if err != nil {
			return "", err
		}
		return v.State, nil
	}); err != nil {
		apiErrDiagnostics("Wait volume deleted failed", err, &resp.Diagnostics)
	}
}

func (r *volumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func setVolumeState(m *volumeModel, out *client.Volume) {
	m.ID = types.StringValue(out.ID)
	m.ProjectID = types.StringValue(out.ProjectID)
	m.RegionID = types.StringValue(out.RegionID)
	m.SizeMiB = types.Int64Value(out.SizeMiB)
	m.Bootable = types.BoolValue(out.Bootable)
	// clone_from is Optional, so an unset config must round-trip as null —
	// storing "" would make OpenTofu reject the apply as inconsistent.
	if out.CloneFrom == "" {
		m.CloneFrom = types.StringNull()
	} else {
		m.CloneFrom = types.StringValue(out.CloneFrom)
	}
	if out.AttachedTo == "" {
		m.AttachedTo = types.StringNull()
	} else {
		m.AttachedTo = types.StringValue(out.AttachedTo)
	}
	m.State = types.StringValue(out.State)
	m.Spec = types.StringValue(rawJSON(out.Spec))
	m.Status = types.StringValue(rawJSON(out.Status))
	m.CreatedAt = types.StringValue(out.CreatedAt)
	m.UpdatedAt = types.StringValue(out.UpdatedAt)
}
