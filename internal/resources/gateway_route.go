package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/client"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/providerdata"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/wait"
)

var _ resource.Resource = (*gatewayRouteResource)(nil)
var _ resource.ResourceWithConfigure = (*gatewayRouteResource)(nil)
var _ resource.ResourceWithImportState = (*gatewayRouteResource)(nil)

func NewGatewayRouteResource() resource.Resource { return &gatewayRouteResource{} }

type gatewayRouteResource struct {
	meta *providerdata.Meta
}

type gatewayRouteModel struct {
	ID            types.String `tfsdk:"id"`
	ProjectID     types.String `tfsdk:"project_id"`
	RegionID      types.String `tfsdk:"region_id"`
	ComputeID     types.String `tfsdk:"compute_id"`
	Protocol      types.String `tfsdk:"protocol"`
	SNIDomain     types.String `tfsdk:"sni_domain"`
	ProxyProtocol types.String `tfsdk:"proxy_protocol"`
	TargetPort    types.Int64  `tfsdk:"target_port"`
	Comment       types.String `tfsdk:"comment"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	State         types.String `tfsdk:"state"`
	Status        types.String `tfsdk:"status"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *gatewayRouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway_route"
}

func (r *gatewayRouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Microsrv gateway route granting console SSH/HTTP reachability for a VM or container, or TLS SNI passthrough to a backend (protocol `tls` requires `sni_domain`, cluster-unique among enabled routes; `proxy_protocol` v1/v2 optional for tls; `target_port` backend dial port for http routes, 0 = default 80). Updatable in place: comment, enabled, target_port, compute_id, protocol, sni_domain, proxy_protocol — the platform re-resolves them without recreating the route (SNI claims re-validated on every change).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"compute_id": schema.StringAttribute{
				Required: true,
			},
			"protocol": schema.StringAttribute{
				Required: true,
			},
			"sni_domain": schema.StringAttribute{
				MarkdownDescription: "FQDN for SNI passthrough (protocol `tls` only). Required when protocol is `tls`, forbidden otherwise. Mutable via update; claims are re-validated cluster-wide.",
				Optional:            true,
			},
			"proxy_protocol": schema.StringAttribute{
				MarkdownDescription: "PROXY protocol header for tls routes (`v1` text or `v2` binary). Only allowed when protocol is `tls`; mutable alongside protocol. Lets backends see the real client IP.",
				Optional:            true,
			},
			"target_port": schema.Int64Attribute{
				MarkdownDescription: "Backend port the console gateway dials for `http` routes (1..65535; 0 = console default 80). Only allowed when protocol is `http`; mutable via update.",
				Optional:            true,
				Computed:            true,
			},
			"comment": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"region_id":  schema.StringAttribute{Computed: true},
			"state":      schema.StringAttribute{Computed: true},
			"status":     schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *gatewayRouteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (r *gatewayRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gatewayRouteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientReq := client.CreateGatewayRouteRequest{
		ComputeID: plan.ComputeID.ValueString(),
		Protocol:  plan.Protocol.ValueString(),
	}
	if !plan.SNIDomain.IsNull() && !plan.SNIDomain.IsUnknown() {
		clientReq.SNIDomain = plan.SNIDomain.ValueString()
	}
	if !plan.ProxyProtocol.IsNull() && !plan.ProxyProtocol.IsUnknown() {
		clientReq.ProxyProtocol = plan.ProxyProtocol.ValueString()
	}
	if tp := int(plan.TargetPort.ValueInt64()); tp > 0 {
		clientReq.TargetPort = &tp
	}
	if !plan.Comment.IsNull() && !plan.Comment.IsUnknown() {
		clientReq.Comment = plan.Comment.ValueString()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		en := plan.Enabled.ValueBool()
		clientReq.Enabled = &en
	}

	out, err := r.meta.Client.CreateGatewayRoute(ctx, plan.ProjectID.ValueString(), clientReq)
	if err != nil {
		apiErrDiagnostics("Create gateway route failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetGatewayRoute(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		rollbackAfterWaitFailure(ctx, r.meta, out.ID, r.meta.Client.DeleteGatewayRoute, func(ctx context.Context, id string) (string, error) {
			v, e := r.meta.Client.GetGatewayRoute(ctx, id)
			if e != nil {
				return "", e
			}
			return v.State, nil
		})
		apiErrDiagnostics("Wait gateway route READY failed", err, &resp.Diagnostics)
		return
	}
	setGatewayRouteState(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gatewayRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gatewayRouteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.meta.Client.GetGatewayRoute(ctx, state.ID.ValueString())
	if err != nil {
		if notFoundRemove(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		apiErrDiagnostics("Read gateway route failed", err, &resp.Diagnostics)
		return
	}
	setGatewayRouteState(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *gatewayRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gatewayRouteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	comment := plan.Comment.ValueString()
	enabled := plan.Enabled.ValueBool()
	targetPort := int(plan.TargetPort.ValueInt64()) // 0 clears back to the default
	// Retarget fields ride every patch: sending the current plan values is
	// idempotent, and null Optional fields marshal as "" (clear/inherit) —
	// the API validates the merged combination on its side.
	computeID := plan.ComputeID.ValueString()
	protocol := plan.Protocol.ValueString()
	sniDomain := plan.SNIDomain.ValueString()
	proxyProtocol := plan.ProxyProtocol.ValueString()
	out, err := r.meta.Client.PatchGatewayRoute(ctx, plan.ID.ValueString(), client.PatchGatewayRouteRequest{
		Comment:       &comment,
		Enabled:       &enabled,
		TargetPort:    &targetPort,
		ComputeID:     &computeID,
		Protocol:      &protocol,
		SNIDomain:     &sniDomain,
		ProxyProtocol: &proxyProtocol,
	})
	if err != nil {
		apiErrDiagnostics("Update gateway route failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetGatewayRoute(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		apiErrDiagnostics("Wait gateway route READY failed", err, &resp.Diagnostics)
		return
	}
	setGatewayRouteState(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gatewayRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gatewayRouteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if err := r.meta.Client.DeleteGatewayRoute(ctx, id); err != nil {
		if !notFoundRemove(err) {
			apiErrDiagnostics("Delete gateway route failed", err, &resp.Diagnostics)
			return
		}
	} else if err := wait.UntilGone(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetGatewayRoute(ctx, id)
		if err != nil {
			return "", err
		}
		return v.State, nil
	}); err != nil {
		apiErrDiagnostics("Wait gateway route deleted failed", err, &resp.Diagnostics)
	}
}

func (r *gatewayRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func setGatewayRouteState(m *gatewayRouteModel, out *client.GatewayRoute) {
	m.ID = types.StringValue(out.ID)
	m.ProjectID = types.StringValue(out.ProjectID)
	m.RegionID = types.StringValue(out.RegionID)
	m.ComputeID = types.StringValue(out.ComputeID)
	m.Protocol = types.StringValue(out.Protocol)
	m.SNIDomain = types.StringValue(out.SNIDomain)
	if out.SNIDomain == "" {
		m.SNIDomain = types.StringNull()
	}
	m.ProxyProtocol = types.StringValue(out.ProxyProtocol)
	if out.ProxyProtocol == "" {
		m.ProxyProtocol = types.StringNull()
	}
	m.TargetPort = types.Int64Value(int64(out.TargetPort))
	m.Comment = types.StringValue(out.Comment)
	if out.Comment == "" {
		m.Comment = types.StringNull()
	}
	m.Enabled = types.BoolValue(out.Enabled)
	m.State = types.StringValue(out.State)
	m.Status = types.StringValue(rawJSON(out.Status))
	m.CreatedAt = types.StringValue(out.CreatedAt)
	m.UpdatedAt = types.StringValue(out.UpdatedAt)
}
