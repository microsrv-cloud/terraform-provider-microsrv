package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/client"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/providerdata"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/wait"
)

var _ resource.Resource = (*wireGuardPeerResource)(nil)
var _ resource.ResourceWithConfigure = (*wireGuardPeerResource)(nil)
var _ resource.ResourceWithImportState = (*wireGuardPeerResource)(nil)

func NewWireGuardPeerResource() resource.Resource { return &wireGuardPeerResource{} }

type wireGuardPeerResource struct {
	meta *providerdata.Meta
}

type wireGuardPeerModel struct {
	ID                  types.String `tfsdk:"id"`
	ProjectID           types.String `tfsdk:"project_id"`
	RegionID            types.String `tfsdk:"region_id"`
	NetworkInterfaceID  types.String `tfsdk:"network_interface_id"`
	PublicKey           types.String `tfsdk:"public_key"`
	Comment             types.String `tfsdk:"comment"`
	PersistentKeepalive types.Int64  `tfsdk:"persistent_keepalive"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	State               types.String `tfsdk:"state"`
	Status              types.String `tfsdk:"status"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

func (r *wireGuardPeerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wireguard_peer"
}

func (r *wireGuardPeerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Microsrv WireGuard peer bound to a network interface. Submit only the public key of a locally generated X25519 keypair; the tunnel IP and gateway endpoint appear in `status` once READY. No update API: changes require replace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"network_interface_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"public_key": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"comment": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"persistent_keepalive": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"region_id":  schema.StringAttribute{Computed: true},
			"state":      schema.StringAttribute{Computed: true},
			"status":     schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *wireGuardPeerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (r *wireGuardPeerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan wireGuardPeerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	clientReq := client.CreateWireGuardPeerRequest{
		NetworkInterfaceID: plan.NetworkInterfaceID.ValueString(),
		PublicKey:          plan.PublicKey.ValueString(),
	}
	if !plan.Comment.IsNull() && !plan.Comment.IsUnknown() {
		clientReq.Comment = plan.Comment.ValueString()
	}
	if !plan.PersistentKeepalive.IsNull() && !plan.PersistentKeepalive.IsUnknown() {
		ka := int(plan.PersistentKeepalive.ValueInt64())
		clientReq.PersistentKeepalive = &ka
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		en := plan.Enabled.ValueBool()
		clientReq.Enabled = &en
	}
	out, err := r.meta.Client.CreateWireGuardPeer(ctx, plan.ProjectID.ValueString(), clientReq)
	if err != nil {
		apiErrDiagnostics("Create wireguard peer failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetWireGuardPeer(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		rollbackAfterWaitFailure(ctx, r.meta, out.ID, r.meta.Client.DeleteWireGuardPeer, func(ctx context.Context, id string) (string, error) {
			v, e := r.meta.Client.GetWireGuardPeer(ctx, id)
			if e != nil {
				return "", e
			}
			return v.State, nil
		})
		apiErrDiagnostics("Wait wireguard peer READY failed", err, &resp.Diagnostics)
		return
	}
	setWireGuardPeerState(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wireGuardPeerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state wireGuardPeerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.meta.Client.GetWireGuardPeer(ctx, state.ID.ValueString())
	if err != nil {
		if notFoundRemove(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		apiErrDiagnostics("Read wireguard peer failed", err, &resp.Diagnostics)
		return
	}
	setWireGuardPeerState(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *wireGuardPeerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "microsrv_wireguard_peer has no update API; changes require replace.")
}

func (r *wireGuardPeerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state wireGuardPeerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if err := r.meta.Client.DeleteWireGuardPeer(ctx, id); err != nil {
		if notFoundRemove(err) {
			return
		}
		apiErrDiagnostics("Delete wireguard peer failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilGone(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetWireGuardPeer(ctx, id)
		if err != nil {
			return "", err
		}
		return v.State, nil
	}); err != nil {
		apiErrDiagnostics("Wait wireguard peer deleted failed", err, &resp.Diagnostics)
	}
}

func (r *wireGuardPeerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func setWireGuardPeerState(m *wireGuardPeerModel, out *client.WireGuardPeer) {
	m.ID = types.StringValue(out.ID)
	m.ProjectID = types.StringValue(out.ProjectID)
	m.NetworkInterfaceID = types.StringValue(out.NetworkInterfaceID)
	m.PublicKey = types.StringValue(out.PublicKey)
	m.Comment = types.StringValue(out.Comment)
	if out.Comment == "" {
		m.Comment = types.StringNull()
	}
	m.PersistentKeepalive = types.Int64Value(int64(out.PersistentKeepalive))
	m.Enabled = types.BoolValue(out.Enabled)
	m.RegionID = types.StringValue(out.RegionID)
	m.State = types.StringValue(out.State)
	m.Status = types.StringValue(rawJSON(out.Status))
	m.CreatedAt = types.StringValue(out.CreatedAt)
	m.UpdatedAt = types.StringValue(out.UpdatedAt)
}
