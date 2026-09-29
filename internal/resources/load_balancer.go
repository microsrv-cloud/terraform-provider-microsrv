package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/client"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/providerdata"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/wait"
)

var _ resource.Resource = (*loadBalancerResource)(nil)
var _ resource.ResourceWithConfigure = (*loadBalancerResource)(nil)
var _ resource.ResourceWithImportState = (*loadBalancerResource)(nil)

func NewLoadBalancerResource() resource.Resource { return &loadBalancerResource{} }

type loadBalancerResource struct {
	meta *providerdata.Meta
}

type loadBalancerModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectID   types.String `tfsdk:"project_id"`
	RegionID    types.String `tfsdk:"region_id"`
	VPCID       types.String `tfsdk:"vpc_id"`
	VIP         types.String `tfsdk:"vip"`
	Listeners   types.List   `tfsdk:"listeners"`
	Backends    types.List   `tfsdk:"backends"`
	HealthCheck types.Object `tfsdk:"health_check"`
	State       types.String `tfsdk:"state"`
	Status      types.String `tfsdk:"status"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

type lbListenerModel struct {
	Protocol   types.String `tfsdk:"protocol"`
	ListenPort types.Int64  `tfsdk:"listen_port"`
	TargetPort types.Int64  `tfsdk:"target_port"`
	Algorithm  types.String `tfsdk:"algorithm"`
}

type lbHealthCheckModel struct {
	Protocol           types.String `tfsdk:"protocol"`
	Port               types.Int64  `tfsdk:"port"`
	HTTPPath           types.String `tfsdk:"http_path"`
	IntervalMS         types.Int64  `tfsdk:"interval_ms"`
	TimeoutMS          types.Int64  `tfsdk:"timeout_ms"`
	UnhealthyThreshold types.Int64  `tfsdk:"unhealthy_threshold"`
	HealthyThreshold   types.Int64  `tfsdk:"healthy_threshold"`
}

var lbListenerAttrTypes = map[string]attr.Type{
	"protocol":    types.StringType,
	"listen_port": types.Int64Type,
	"target_port": types.Int64Type,
	"algorithm":   types.StringType,
}

var lbHealthCheckAttrTypes = map[string]attr.Type{
	"protocol":            types.StringType,
	"port":                types.Int64Type,
	"http_path":           types.StringType,
	"interval_ms":         types.Int64Type,
	"timeout_ms":          types.Int64Type,
	"unhealthy_threshold": types.Int64Type,
	"healthy_threshold":   types.Int64Type,
}

func (r *loadBalancerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_load_balancer"
}

func (r *loadBalancerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Microsrv L4 load balancer for one VPC with an explicit VIP, TCP/UDP listeners, and VM/container backends. vpc_id and vip require replace; listeners, backends, and health_check are updatable.",
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
			"vip": schema.StringAttribute{
				MarkdownDescription: "Explicit IPv4 VIP: a usable host address in the VPC, not colliding with any NI IP or other LB VIP.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"listeners": schema.ListNestedAttribute{
				MarkdownDescription: "1..32 listeners; unique protocol+listen_port; algorithm is round_robin (default) or source_ip.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"protocol": schema.StringAttribute{Required: true},
						"listen_port": schema.Int64Attribute{
							Required: true,
						},
						"target_port": schema.Int64Attribute{
							Required: true,
						},
						"algorithm": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"backends": schema.ListAttribute{
				MarkdownDescription: "VM or container IDs (1..64) in the same project and region as the VPC.",
				ElementType:         types.StringType,
				Required:            true,
			},
			"health_check": schema.SingleNestedAttribute{
				MarkdownDescription: "Optional probe overrides; unset fields keep platform defaults (tcp probe on the first listener port).",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"protocol": schema.StringAttribute{
						MarkdownDescription: "tcp (default) or http.",
						Optional:            true,
						Computed:            true,
					},
					"port":                schema.Int64Attribute{Optional: true, Computed: true},
					"http_path":           schema.StringAttribute{Optional: true, Computed: true},
					"interval_ms":         schema.Int64Attribute{Optional: true, Computed: true},
					"timeout_ms":          schema.Int64Attribute{Optional: true, Computed: true},
					"unhealthy_threshold": schema.Int64Attribute{Optional: true, Computed: true},
					"healthy_threshold":   schema.Int64Attribute{Optional: true, Computed: true},
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

func (r *loadBalancerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (r *loadBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan loadBalancerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	listeners, hc, d := lbFromPlan(ctx, &plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	backends, d := listToStrings(ctx, plan.Backends)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientReq := client.CreateLoadBalancerRequest{
		VPCID:     plan.VPCID.ValueString(),
		VIP:       plan.VIP.ValueString(),
		Listeners: listeners,
		Backends:  backends,
	}
	if hc != nil {
		clientReq.HealthCheck = hc
	}

	out, err := r.meta.Client.CreateLoadBalancer(ctx, plan.ProjectID.ValueString(), clientReq)
	if err != nil {
		apiErrDiagnostics("Create load balancer failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetLoadBalancer(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		rollbackAfterWaitFailure(ctx, r.meta, out.ID, r.meta.Client.DeleteLoadBalancer, func(ctx context.Context, id string) (string, error) {
			v, e := r.meta.Client.GetLoadBalancer(ctx, id)
			if e != nil {
				return "", e
			}
			return v.State, nil
		})
		apiErrDiagnostics("Wait load balancer READY failed", err, &resp.Diagnostics)
		return
	}
	setLoadBalancerState(ctx, &plan, out, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *loadBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state loadBalancerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.meta.Client.GetLoadBalancer(ctx, state.ID.ValueString())
	if err != nil {
		if notFoundRemove(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		apiErrDiagnostics("Read load balancer failed", err, &resp.Diagnostics)
		return
	}
	setLoadBalancerState(ctx, &state, out, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *loadBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan loadBalancerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	listeners, hc, d := lbFromPlan(ctx, &plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	backends, d := listToStrings(ctx, plan.Backends)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	// listeners/backends are full-table replaces on the API; health_check
	// nil means "leave unchanged", so a removed block resets to defaults.
	patch := client.PatchLoadBalancerRequest{
		Listeners:   &listeners,
		Backends:    &backends,
		HealthCheck: &client.LBHealthCheck{},
	}
	if hc != nil {
		patch.HealthCheck = hc
	}
	out, err := r.meta.Client.PatchLoadBalancer(ctx, plan.ID.ValueString(), patch)
	if err != nil {
		apiErrDiagnostics("Update load balancer failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetLoadBalancer(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		apiErrDiagnostics("Wait load balancer READY failed", err, &resp.Diagnostics)
		return
	}
	setLoadBalancerState(ctx, &plan, out, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *loadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state loadBalancerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if err := r.meta.Client.DeleteLoadBalancer(ctx, id); err != nil {
		if !notFoundRemove(err) {
			apiErrDiagnostics("Delete load balancer failed", err, &resp.Diagnostics)
			return
		}
	} else if err := wait.UntilGone(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetLoadBalancer(ctx, id)
		if err != nil {
			return "", err
		}
		return v.State, nil
	}); err != nil {
		apiErrDiagnostics("Wait load balancer deleted failed", err, &resp.Diagnostics)
	}
}

func (r *loadBalancerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// lbFromPlan converts the listeners list and optional health_check object
// into client types. Backends stay as types.List (converted by callers).
func lbFromPlan(ctx context.Context, plan *loadBalancerModel) ([]client.LBListener, *client.LBHealthCheck, diag.Diagnostics) {
	var diags diag.Diagnostics
	var listenerModels []lbListenerModel
	diags.Append(plan.Listeners.ElementsAs(ctx, &listenerModels, false)...)
	if diags.HasError() {
		return nil, nil, diags
	}
	listeners := make([]client.LBListener, len(listenerModels))
	for i, l := range listenerModels {
		listeners[i] = client.LBListener{
			Protocol:   l.Protocol.ValueString(),
			ListenPort: uint32(l.ListenPort.ValueInt64()),
			TargetPort: uint32(l.TargetPort.ValueInt64()),
			Algorithm:  l.Algorithm.ValueString(),
		}
	}
	if plan.HealthCheck.IsNull() || plan.HealthCheck.IsUnknown() {
		return listeners, nil, diags
	}
	var hc lbHealthCheckModel
	diags.Append(plan.HealthCheck.As(ctx, &hc, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, nil, diags
	}
	return listeners, &client.LBHealthCheck{
		Protocol:           hc.Protocol.ValueString(),
		Port:               uint32(hc.Port.ValueInt64()),
		HTTPPath:           hc.HTTPPath.ValueString(),
		IntervalMS:         uint32(hc.IntervalMS.ValueInt64()),
		TimeoutMS:          uint32(hc.TimeoutMS.ValueInt64()),
		UnhealthyThreshold: uint32(hc.UnhealthyThreshold.ValueInt64()),
		HealthyThreshold:   uint32(hc.HealthyThreshold.ValueInt64()),
	}, diags
}

func setLoadBalancerState(ctx context.Context, m *loadBalancerModel, out *client.LoadBalancer, diags *diag.Diagnostics) {
	m.ID = types.StringValue(out.ID)
	m.ProjectID = types.StringValue(out.ProjectID)
	m.RegionID = types.StringValue(out.RegionID)
	m.VPCID = types.StringValue(out.VPCID)
	m.VIP = types.StringValue(out.VIP)
	listenerVals := make([]attr.Value, len(out.Listeners))
	for i, l := range out.Listeners {
		v, d := types.ObjectValueFrom(ctx, lbListenerAttrTypes, lbListenerModel{
			Protocol:   types.StringValue(l.Protocol),
			ListenPort: types.Int64Value(int64(l.ListenPort)),
			TargetPort: types.Int64Value(int64(l.TargetPort)),
			Algorithm:  types.StringValue(l.Algorithm),
		})
		diags.Append(d...)
		listenerVals[i] = v
	}
	if diags.HasError() {
		return
	}
	lst, d := types.ListValue(types.ObjectType{AttrTypes: lbListenerAttrTypes}, listenerVals)
	diags.Append(d...)
	m.Listeners = lst
	m.Backends = stringsToList(out.Backends)
	if hcZero(out.HealthCheck) {
		m.HealthCheck = types.ObjectNull(lbHealthCheckAttrTypes)
	} else {
		v, d := types.ObjectValueFrom(ctx, lbHealthCheckAttrTypes, lbHealthCheckModel{
			Protocol:           types.StringValue(out.HealthCheck.Protocol),
			Port:               intOrNull(out.HealthCheck.Port),
			HTTPPath:           types.StringValue(out.HealthCheck.HTTPPath),
			IntervalMS:         intOrNull(out.HealthCheck.IntervalMS),
			TimeoutMS:          intOrNull(out.HealthCheck.TimeoutMS),
			UnhealthyThreshold: intOrNull(out.HealthCheck.UnhealthyThreshold),
			HealthyThreshold:   intOrNull(out.HealthCheck.HealthyThreshold),
		})
		diags.Append(d...)
		m.HealthCheck = v
	}
	m.State = types.StringValue(out.State)
	m.Status = types.StringValue(rawJSON(out.Status))
	m.CreatedAt = types.StringValue(out.CreatedAt)
	m.UpdatedAt = types.StringValue(out.UpdatedAt)
}

func hcZero(hc client.LBHealthCheck) bool {
	return hc.Protocol == "" && hc.Port == 0 && hc.HTTPPath == "" &&
		hc.IntervalMS == 0 && hc.TimeoutMS == 0 &&
		hc.UnhealthyThreshold == 0 && hc.HealthyThreshold == 0
}

func intOrNull(v uint32) types.Int64 {
	if v == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(int64(v))
}
