package resources

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/client"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/providerdata"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/wait"
)

var _ resource.Resource = (*vmResource)(nil)
var _ resource.ResourceWithConfigure = (*vmResource)(nil)
var _ resource.ResourceWithImportState = (*vmResource)(nil)

func NewVMResource() resource.Resource { return &vmResource{} }

type vmResource struct {
	meta *providerdata.Meta
}

type vmModel struct {
	ID                  types.String `tfsdk:"id"`
	ProjectID           types.String `tfsdk:"project_id"`
	RegionID            types.String `tfsdk:"region_id"`
	FlavorID            types.String `tfsdk:"flavor_id"`
	Class               types.String `tfsdk:"class"`
	Serverless          types.Bool   `tfsdk:"serverless"`
	Name                types.String `tfsdk:"name"`
	NetworkInterfaceIDs types.List   `tfsdk:"network_interface_ids"`
	SSHKeyIDs           types.List   `tfsdk:"ssh_key_ids"`
	VolumeIDs           types.List   `tfsdk:"volume_ids"`
	BootVolumeID        types.String `tfsdk:"boot_volume_id"`
	State               types.String `tfsdk:"state"`
	Status              types.String `tfsdk:"status"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

func (r *vmResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (r *vmResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Microsrv VM. The boot disk is an explicit `microsrv_volume` resource referenced by `boot_volume_id` (never created inline by this resource). Destroying a VM detaches and leaves attached volumes to their own resources. `serverless` enables transport-layer scale-to-zero (checkpointed on idle, RAM on disk); immutable after create. State may be READY or CHECKPOINTED (paused).",
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
			"flavor_id": schema.StringAttribute{Required: true},
			"class": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"serverless": schema.BoolAttribute{
				MarkdownDescription: "Enable scale-to-zero (serverless) with default policy knobs. Immutable after create; requires replace.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{Required: true},
			"network_interface_ids": schema.ListAttribute{
				ElementType:   types.StringType,
				Required:      true,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"ssh_key_ids": schema.ListAttribute{
				ElementType:   types.StringType,
				Required:      true,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"volume_ids": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"boot_volume_id": schema.StringAttribute{
				MarkdownDescription: "ID of an explicit bootable `microsrv_volume` (created with `clone_from`) to boot from.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"state":      schema.StringAttribute{Computed: true},
			"status":     schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *vmResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (r *vmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vmModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bootVol := plan.BootVolumeID.ValueString()
	if bootVol == "" {
		resp.Diagnostics.AddAttributeError(path.Root("boot_volume_id"), "Missing boot_volume_id", "Reference an explicit bootable volume (microsrv_volume created with clone_from).")
		return
	}

	niIDs, d := listToStrings(ctx, plan.NetworkInterfaceIDs)
	resp.Diagnostics.Append(d...)
	sshIDs, d := listToStrings(ctx, plan.SSHKeyIDs)
	resp.Diagnostics.Append(d...)
	volIDs, d := listToStrings(ctx, plan.VolumeIDs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateVMRequest{
		RegionID:            plan.RegionID.ValueString(),
		FlavorID:            plan.FlavorID.ValueString(),
		Class:               plan.Class.ValueString(),
		Name:                plan.Name.ValueString(),
		NetworkInterfaceIDs: niIDs,
		SSHKeyIDs:           sshIDs,
		VolumeIDs:           volIDs,
		BootVolumeID:        bootVol,
		Serverless:          plan.Serverless.ValueBool(),
	}

	out, err := r.meta.Client.CreateVM(ctx, plan.ProjectID.ValueString(), createReq)
	if err != nil {
		apiErrDiagnostics("Create VM failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetVM(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		// Console tail BEFORE rollback: the evidence dies with the VM.
		logs, _ := r.meta.Client.GetVMLogs(ctx, out.ID, 50)
		// Roll back only the VM; the boot/data volumes are explicit resources
		// owned by their own microsrv_volume blocks.
		rollbackAfterWaitFailure(ctx, r.meta, out.ID, r.meta.Client.DeleteVM, func(ctx context.Context, id string) (string, error) {
			v, e := r.meta.Client.GetVM(ctx, id)
			if e != nil {
				return "", e
			}
			return v.State, nil
		})
		apiErrDiagnostics("Wait VM READY failed", withLogs(err, logs), &resp.Diagnostics)
		return
	}
	setVMState(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.meta.Client.GetVM(ctx, state.ID.ValueString())
	if err != nil {
		if notFoundRemove(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		apiErrDiagnostics("Read VM failed", err, &resp.Diagnostics)
		return
	}
	setVMState(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *vmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vmModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.meta.Client.PatchVM(ctx, plan.ID.ValueString(), client.PatchVMRequest{
		Name:     plan.Name.ValueString(),
		FlavorID: plan.FlavorID.ValueString(),
	})
	if err != nil {
		apiErrDiagnostics("Update VM failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetVM(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		logs, _ := r.meta.Client.GetVMLogs(ctx, out.ID, 50)
		apiErrDiagnostics("Wait VM READY failed", withLogs(err, logs), &resp.Diagnostics)
		return
	}
	setVMState(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vmModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Volumes (boot + data) are explicit resources; destroy only detaches them
	// by deleting the VM and leaves them for their own microsrv_volume blocks.
	id := state.ID.ValueString()
	if err := r.meta.Client.DeleteVM(ctx, id); err != nil {
		if !notFoundRemove(err) {
			apiErrDiagnostics("Delete VM failed", err, &resp.Diagnostics)
			return
		}
	} else if err := wait.UntilGone(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetVM(ctx, id)
		if err != nil {
			return "", err
		}
		return v.State, nil
	}); err != nil {
		apiErrDiagnostics("Wait VM deleted failed", err, &resp.Diagnostics)
		return
	}
}

func (r *vmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func setVMState(m *vmModel, out *client.VM) {
	m.ID = types.StringValue(out.ID)
	m.ProjectID = types.StringValue(out.ProjectID)
	m.RegionID = types.StringValue(out.RegionID)
	m.FlavorID = types.StringValue(out.FlavorID)
	m.Class = types.StringValue(out.Class)
	m.Serverless = types.BoolValue(out.Serverless)
	m.Name = types.StringValue(out.Name)
	m.NetworkInterfaceIDs = stringsToList(out.NetworkInterfaceIDs)
	// The API prepends the explicit boot volume to volume_ids; state must
	// mirror config exactly (sorted key list, data disks only) or Tofu rejects
	// the apply with "inconsistent result after apply".
	keys := append([]string(nil), out.SSHKeyIDs...)
	sort.Strings(keys)
	m.SSHKeyIDs = stringsToList(keys)
	vols := append([]string(nil), out.VolumeIDs...)
	if len(vols) > 0 {
		vols = vols[1:]
	}
	m.VolumeIDs = stringsToList(vols)
	m.State = types.StringValue(out.State)
	m.Status = types.StringValue(rawJSON(out.Status))
	m.CreatedAt = types.StringValue(out.CreatedAt)
	m.UpdatedAt = types.StringValue(out.UpdatedAt)
}

func listToStrings(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	var elems []types.String
	diags.Append(list.ElementsAs(ctx, &elems, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]string, 0, len(elems))
	for _, e := range elems {
		out = append(out, e.ValueString())
	}
	return out, diags
}

func stringsToList(vals []string) types.List {
	if vals == nil {
		vals = []string{}
	}
	elems := make([]attr.Value, len(vals))
	for i, v := range vals {
		elems[i] = types.StringValue(v)
	}
	return types.ListValueMust(types.StringType, elems)
}
