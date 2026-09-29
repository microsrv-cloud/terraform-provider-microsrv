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

var _ resource.Resource = (*networkInterfaceResource)(nil)
var _ resource.ResourceWithConfigure = (*networkInterfaceResource)(nil)
var _ resource.ResourceWithImportState = (*networkInterfaceResource)(nil)

func NewNetworkInterfaceResource() resource.Resource { return &networkInterfaceResource{} }

type networkInterfaceResource struct {
	meta *providerdata.Meta
}

type networkInterfaceModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	VPCID     types.String `tfsdk:"vpc_id"`
	RegionID  types.String `tfsdk:"region_id"`
	IPAddress types.String `tfsdk:"ip_address"`
	PrefixLen types.Int64  `tfsdk:"prefix_len"`
	State     types.String `tfsdk:"state"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *networkInterfaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_interface"
}

// reassignIPOnReplace stops an omitted ip_address from being carried into a
// replacement that targets another VPC or project: the framework's proposed
// plan keeps the prior address for a null Optional+Computed config, which
// would pin the new NI to an address from the old VPC (API 400) instead of
// auto-allocating. Marking the plan value unknown tells Create to send no
// ip_address so the platform assigns a fresh one.
type reassignIPOnReplace struct{}

func (reassignIPOnReplace) Description(context.Context) string {
	return "Reassigns the IP when a replacement targets another VPC or project."
}

func (d reassignIPOnReplace) MarkdownDescription(ctx context.Context) string {
	return d.Description(ctx)
}

func (reassignIPOnReplace) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() || req.StateValue.IsNull() {
		return // explicit pin: RequiresReplace handles changes; no state: first create
	}
	var plan, state networkInterfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.VPCID.Equal(state.VPCID) || !plan.ProjectID.Equal(state.ProjectID) {
		resp.PlanValue = types.StringUnknown()
	}
}

func (r *networkInterfaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Microsrv network interface bound to a VPC. Set `ip_address` to pin a specific address, or omit it and the platform assigns the first free address in the VPC range.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"vpc_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"region_id": schema.StringAttribute{Computed: true},
			"ip_address": schema.StringAttribute{
				MarkdownDescription: "Optional pinned IPv4 inside the VPC range (not network/gateway/broadcast, not already used by an NI or LB VIP). Omit to let the platform assign the first free address. Changing it forces replacement.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{reassignIPOnReplace{}, stringplanmodifier.RequiresReplace()},
			},
			"prefix_len": schema.Int64Attribute{Computed: true},
			"state":      schema.StringAttribute{Computed: true},
			"status":     schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *networkInterfaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (r *networkInterfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkInterfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.meta.Client.CreateNetworkInterface(ctx, plan.ProjectID.ValueString(), client.CreateNetworkInterfaceRequest{
		VPCID:     plan.VPCID.ValueString(),
		IPAddress: plan.IPAddress.ValueString(),
	})
	if err != nil {
		apiErrDiagnostics("Create network interface failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetNetworkInterface(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		rollbackAfterWaitFailure(ctx, r.meta, out.ID, r.meta.Client.DeleteNetworkInterface, func(ctx context.Context, id string) (string, error) {
			v, e := r.meta.Client.GetNetworkInterface(ctx, id)
			if e != nil {
				return "", e
			}
			return v.State, nil
		})
		apiErrDiagnostics("Wait network interface READY failed", err, &resp.Diagnostics)
		return
	}
	setNetworkInterfaceState(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *networkInterfaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkInterfaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.meta.Client.GetNetworkInterface(ctx, state.ID.ValueString())
	if err != nil {
		if notFoundRemove(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		apiErrDiagnostics("Read network interface failed", err, &resp.Diagnostics)
		return
	}
	setNetworkInterfaceState(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *networkInterfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "microsrv_network_interface has no update API; changes require replace.")
}

func (r *networkInterfaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkInterfaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if err := r.meta.Client.DeleteNetworkInterface(ctx, id); err != nil {
		if notFoundRemove(err) {
			return
		}
		apiErrDiagnostics("Delete network interface failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilGone(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetNetworkInterface(ctx, id)
		if err != nil {
			return "", err
		}
		return v.State, nil
	}); err != nil {
		apiErrDiagnostics("Wait network interface deleted failed", err, &resp.Diagnostics)
	}
}

func (r *networkInterfaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func setNetworkInterfaceState(m *networkInterfaceModel, out *client.NetworkInterface) {
	m.ID = types.StringValue(out.ID)
	m.ProjectID = types.StringValue(out.ProjectID)
	m.VPCID = types.StringValue(out.VPCID)
	m.RegionID = types.StringValue(out.RegionID)
	m.IPAddress = types.StringValue(out.IPAddress)
	m.PrefixLen = types.Int64Value(int64(out.PrefixLen))
	m.State = types.StringValue(out.State)
	m.Status = types.StringValue(rawJSON(out.Status))
	m.CreatedAt = types.StringValue(out.CreatedAt)
	m.UpdatedAt = types.StringValue(out.UpdatedAt)
}
