package resources

import (
	"context"

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

var _ resource.Resource = (*containerResource)(nil)
var _ resource.ResourceWithConfigure = (*containerResource)(nil)
var _ resource.ResourceWithImportState = (*containerResource)(nil)

func NewContainerResource() resource.Resource { return &containerResource{} }

type containerResource struct {
	meta *providerdata.Meta
}

type containerModel struct {
	ID                  types.String `tfsdk:"id"`
	ProjectID           types.String `tfsdk:"project_id"`
	RegionID            types.String `tfsdk:"region_id"`
	FlavorID            types.String `tfsdk:"flavor_id"`
	Class               types.String `tfsdk:"class"`
	Serverless          types.Bool   `tfsdk:"serverless"`
	ImageID             types.String `tfsdk:"image_id"`
	Name                types.String `tfsdk:"name"`
	Command             types.List   `tfsdk:"command"`
	Args                types.List   `tfsdk:"args"`
	Env                 types.Map    `tfsdk:"env"`
	RestartPolicy       types.String `tfsdk:"restart_policy"`
	User                types.String `tfsdk:"user"`
	VolumeIDs           types.List   `tfsdk:"volume_ids"`
	NetworkInterfaceIDs types.List   `tfsdk:"network_interface_ids"`
	SSHKeyIDs           types.List   `tfsdk:"ssh_key_ids"`
	State               types.String `tfsdk:"state"`
	Status              types.String `tfsdk:"status"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

func (r *containerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container"
}

func (r *containerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Microsrv gVisor container running an OCI image. Updatable in place: name, flavor_id, ssh_key_ids, command, args, env, user (applied by Microsrv through a controlled sandbox restart — no resource recreate). image, restart policy, and attachments require replace. `serverless` enables scale-to-zero (checkpointed on idle); immutable. State may be READY or CHECKPOINTED.",
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
			"image_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{Required: true},
			"command": schema.ListAttribute{
				MarkdownDescription: "Container command. Empty inherits entrypoint/cmd from the image (when the image has process defaults). Updatable in place.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
			},
			"args": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"env": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"restart_policy": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "Override the image USER (empty inherits). Allowed: `root`/`root:root`/`0`/`0:0` or exactly the image's predefined user. Updatable in place.",
				Optional:            true,
				Computed:            true,
			},
			"volume_ids": schema.ListAttribute{
				ElementType:   types.StringType,
				Required:      true,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"network_interface_ids": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"ssh_key_ids": schema.ListAttribute{
				MarkdownDescription: "User SSH keys for console SSH. Unset defaults to every SSH key registered in the project (server-side keyring); an explicit non-empty list is PATCHable (1..N). An explicit empty list is not supported.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
			},
			"state":      schema.StringAttribute{Computed: true},
			"status":     schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *containerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (r *containerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan containerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.validateContainer(&plan, &resp.Diagnostics); err != nil {
		return
	}

	command, d := listToStrings(ctx, plan.Command)
	resp.Diagnostics.Append(d...)
	args, d := listToStrings(ctx, plan.Args)
	resp.Diagnostics.Append(d...)
	volIDs, d := listToStrings(ctx, plan.VolumeIDs)
	resp.Diagnostics.Append(d...)
	niIDs, d := listToStrings(ctx, plan.NetworkInterfaceIDs)
	resp.Diagnostics.Append(d...)
	sshIDs, d := listToStrings(ctx, plan.SSHKeyIDs)
	resp.Diagnostics.Append(d...)
	env, d := mapToStrings(ctx, plan.Env)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	user := ""
	if !plan.User.IsNull() && !plan.User.IsUnknown() {
		user = plan.User.ValueString()
	}
	out, err := r.meta.Client.CreateContainer(ctx, plan.ProjectID.ValueString(), client.CreateContainerRequest{
		RegionID:            plan.RegionID.ValueString(),
		FlavorID:            plan.FlavorID.ValueString(),
		Class:               plan.Class.ValueString(),
		Serverless:          plan.Serverless.ValueBool(),
		ImageID:             plan.ImageID.ValueString(),
		Name:                plan.Name.ValueString(),
		Command:             command,
		Args:                args,
		Env:                 env,
		RestartPolicy:       plan.RestartPolicy.ValueString(),
		User:                user,
		VolumeIDs:           volIDs,
		NetworkInterfaceIDs: niIDs,
		SSHKeyIDs:           sshIDs,
	})
	if err != nil {
		apiErrDiagnostics("Create container failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetContainer(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		// Grab the log tail BEFORE rollback: deleting the container destroys
		// the only evidence of why it never became ready.
		logs, _ := r.meta.Client.GetContainerLogs(ctx, out.ID, 50)
		rollbackAfterWaitFailure(ctx, r.meta, out.ID, r.meta.Client.DeleteContainer, func(ctx context.Context, id string) (string, error) {
			v, e := r.meta.Client.GetContainer(ctx, id)
			if e != nil {
				return "", e
			}
			return v.State, nil
		})
		apiErrDiagnostics("Wait container READY failed", withLogs(err, logs), &resp.Diagnostics)
		return
	}
	setContainerState(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *containerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state containerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.meta.Client.GetContainer(ctx, state.ID.ValueString())
	if err != nil {
		if notFoundRemove(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		apiErrDiagnostics("Read container failed", err, &resp.Diagnostics)
		return
	}
	setContainerState(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *containerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan containerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := plan.Name.ValueString()
	flavorID := plan.FlavorID.ValueString()
	patchReq := client.PatchContainerRequest{
		Name:     &name,
		FlavorID: &flavorID,
	}
	// Process config rides every patch (in-place update: Microsrv applies it
	// through a controlled sandbox restart, never a resource recreate).
	// null/unknown helpers return nil → pointer omitted → API keeps current.
	command, dCmd := listToStrings(ctx, plan.Command)
	resp.Diagnostics.Append(dCmd...)
	args, dArgs := listToStrings(ctx, plan.Args)
	resp.Diagnostics.Append(dArgs...)
	env, dEnv := mapToStrings(ctx, plan.Env)
	resp.Diagnostics.Append(dEnv...)
	if resp.Diagnostics.HasError() {
		return
	}
	if command != nil {
		patchReq.Command = &command
	}
	if args != nil {
		patchReq.Args = &args
	}
	if env != nil {
		patchReq.Env = &env
	}
	user := plan.User.ValueString()
	patchReq.User = &user
	sshIDs, d := listToStrings(ctx, plan.SSHKeyIDs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(sshIDs) == 0 && !plan.SSHKeyIDs.IsNull() && !plan.SSHKeyIDs.IsUnknown() {
		resp.Diagnostics.AddError("Cannot empty ssh_key_ids", "The API requires at least one key on update; omit the attribute (or list existing keys) instead of setting it empty.")
		return
	}
	if len(sshIDs) > 0 {
		patchReq.SSHKeyIDs = &sshIDs
	}
	out, err := r.meta.Client.PatchContainer(ctx, plan.ID.ValueString(), patchReq)
	if err != nil {
		apiErrDiagnostics("Update container failed", err, &resp.Diagnostics)
		return
	}
	if err := wait.UntilReady(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetContainer(ctx, out.ID)
		if err != nil {
			return "", err
		}
		out = v
		return v.State, nil
	}); err != nil {
		apiErrDiagnostics("Wait container READY failed", err, &resp.Diagnostics)
		return
	}
	setContainerState(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *containerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state containerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if err := r.meta.Client.DeleteContainer(ctx, id); err != nil {
		if !notFoundRemove(err) {
			apiErrDiagnostics("Delete container failed", err, &resp.Diagnostics)
			return
		}
	} else if err := wait.UntilGone(ctx, r.meta.Wait, func(ctx context.Context) (string, error) {
		v, err := r.meta.Client.GetContainer(ctx, id)
		if err != nil {
			return "", err
		}
		return v.State, nil
	}); err != nil {
		apiErrDiagnostics("Wait container deleted failed", err, &resp.Diagnostics)
	}
}

func (r *containerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *containerResource) validateContainer(plan *containerModel, diags *diag.Diagnostics) error {
	// Command may be empty when the image has entrypoint/cmd defaults (the
	// platform inherits them from the OCI config); allow empty here and let the API
	// validate against the image config.
	if plan.VolumeIDs.IsNull() || len(plan.VolumeIDs.Elements()) == 0 {
		diags.AddError("Missing volume_ids", "volume_ids is required and must contain at least one volume.")
	}
	// Unset ssh_key_ids defaults to the project keyring server-side, but an
	// explicit empty list would diverge from the default the API stores.
	if !plan.SSHKeyIDs.IsNull() && !plan.SSHKeyIDs.IsUnknown() && len(plan.SSHKeyIDs.Elements()) == 0 {
		diags.AddError("Empty ssh_key_ids", "Omit ssh_key_ids to default to the project SSH keyring, or list at least one key (the API rejects empty lists).")
	}
	return nil
}

func setContainerState(m *containerModel, out *client.Container) {
	m.ID = types.StringValue(out.ID)
	m.ProjectID = types.StringValue(out.ProjectID)
	m.RegionID = types.StringValue(out.RegionID)
	m.FlavorID = types.StringValue(out.FlavorID)
	m.Class = types.StringValue(out.Class)
	m.Serverless = types.BoolValue(out.Serverless)
	m.ImageID = types.StringValue(out.ImageID)
	m.Name = types.StringValue(out.Name)
	m.Command = stringsToList(out.Command)
	m.Args = stringsToList(out.Args)
	m.Env = mapToValue(out.Env)
	m.RestartPolicy = types.StringValue(out.RestartPolicy)
	if out.User == "" {
		m.User = types.StringNull()
	} else {
		m.User = types.StringValue(out.User)
	}
	m.VolumeIDs = stringsToList(out.VolumeIDs)
	m.NetworkInterfaceIDs = stringsToList(out.NetworkInterfaceIDs)
	m.SSHKeyIDs = stringsToList(out.SSHKeyIDs)
	m.State = types.StringValue(out.State)
	m.Status = types.StringValue(rawJSON(out.Status))
	m.CreatedAt = types.StringValue(out.CreatedAt)
	m.UpdatedAt = types.StringValue(out.UpdatedAt)
}

func mapToStrings(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if m.IsNull() || m.IsUnknown() {
		return nil, diags
	}
	var out map[string]string
	diags.Append(m.ElementsAs(ctx, &out, false)...)
	return out, diags
}

func mapToValue(vals map[string]string) types.Map {
	if vals == nil {
		vals = map[string]string{}
	}
	elems := make(map[string]attr.Value, len(vals))
	for k, v := range vals {
		elems[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, elems)
}
