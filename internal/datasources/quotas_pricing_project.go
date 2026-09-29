package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/providerdata"
)

var _ datasource.DataSource = (*quotasDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*quotasDataSource)(nil)

func NewQuotasDataSource() datasource.DataSource { return &quotasDataSource{} }

type quotasDataSource struct {
	meta *providerdata.Meta
}

type quotasModel struct {
	Quotas types.List `tfsdk:"quotas"`
}

func quotaAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":            types.StringType,
		"resource_kind": types.StringType,
		"hard_limit":    types.Int64Type,
		"usage":         types.Int64Type,
	}
}

func (d *quotasDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_quotas"
}

func (d *quotasDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Current user quota limits and usage.",
		Attributes: map[string]schema.Attribute{
			"quotas": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.StringAttribute{Computed: true},
						"resource_kind": schema.StringAttribute{Computed: true},
						"hard_limit":    schema.Int64Attribute{Computed: true},
						"usage":         schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *quotasDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (d *quotasDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state quotasModel
	out, err := d.meta.Client.ListQuotas(ctx)
	if err != nil {
		resp.Diagnostics.AddError("List quotas failed", err.Error())
		return
	}
	elems := make([]attr.Value, 0, len(out))
	for _, q := range out {
		elems = append(elems, types.ObjectValueMust(quotaAttrTypes(), map[string]attr.Value{
			"id":            types.StringValue(q.ID),
			"resource_kind": types.StringValue(q.ResourceKind),
			"hard_limit":    types.Int64Value(q.HardLimit),
			"usage":         types.Int64Value(q.Usage),
		}))
	}
	state.Quotas = types.ListValueMust(types.ObjectType{AttrTypes: quotaAttrTypes()}, elems)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

var _ datasource.DataSource = (*pricingDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*pricingDataSource)(nil)

func NewPricingDataSource() datasource.DataSource { return &pricingDataSource{} }

type pricingDataSource struct {
	meta *providerdata.Meta
}

type pricingModel struct {
	RegionID types.String `tfsdk:"region_id"`
	Prices   types.List   `tfsdk:"prices"`
}

func priceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":             types.StringType,
		"region_id":      types.StringType,
		"resource_kind":  types.StringType,
		"price_per_hour": types.Float64Type,
	}
}

func (d *pricingDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pricing"
}

func (d *pricingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Region pricing catalog. Optional region_id filter.",
		Attributes: map[string]schema.Attribute{
			"region_id": schema.StringAttribute{Optional: true},
			"prices": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":             schema.StringAttribute{Computed: true},
						"region_id":      schema.StringAttribute{Computed: true},
						"resource_kind":  schema.StringAttribute{Computed: true},
						"price_per_hour": schema.Float64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *pricingDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (d *pricingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config pricingModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	regionID := ""
	if !config.RegionID.IsNull() {
		regionID = config.RegionID.ValueString()
	}
	out, err := d.meta.Client.ListPricing(ctx, regionID)
	if err != nil {
		resp.Diagnostics.AddError("List pricing failed", err.Error())
		return
	}
	elems := make([]attr.Value, 0, len(out))
	for _, p := range out {
		elems = append(elems, types.ObjectValueMust(priceAttrTypes(), map[string]attr.Value{
			"id":             types.StringValue(p.ID),
			"region_id":      types.StringValue(p.RegionID),
			"resource_kind":  types.StringValue(p.ResourceKind),
			"price_per_hour": types.Float64Value(p.PricePerHour),
		}))
	}
	config.Prices = types.ListValueMust(types.ObjectType{AttrTypes: priceAttrTypes()}, elems)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

var _ datasource.DataSource = (*projectDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*projectDataSource)(nil)

func NewProjectDataSource() datasource.DataSource { return &projectDataSource{} }

type projectDataSource struct {
	meta *providerdata.Meta
}

type projectDSModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	OwnerUserID types.String `tfsdk:"owner_user_id"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
}

func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a platform project by id or name.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Optional: true, Computed: true},
			"name":          schema.StringAttribute{Optional: true, Computed: true},
			"owner_user_id": schema.StringAttribute{Computed: true},
			"is_default":    schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *projectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config projectDSModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	hasID := !config.ID.IsNull() && config.ID.ValueString() != ""
	hasName := !config.Name.IsNull() && config.Name.ValueString() != ""
	if hasID == hasName {
		resp.Diagnostics.AddError("Invalid lookup", "Provide exactly one of id or name.")
		return
	}
	if hasID {
		out, err := d.meta.Client.GetProject(ctx, config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Get project failed", err.Error())
			return
		}
		config.ID = types.StringValue(out.ID)
		config.Name = types.StringValue(out.Name)
		config.OwnerUserID = types.StringValue(out.OwnerUserID)
		config.IsDefault = types.BoolValue(out.IsDefault)
		resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
		return
	}
	all, err := d.meta.Client.ListProjects(ctx)
	if err != nil {
		resp.Diagnostics.AddError("List projects failed", err.Error())
		return
	}
	name := config.Name.ValueString()
	for _, p := range all {
		if p.Name == name {
			config.ID = types.StringValue(p.ID)
			config.Name = types.StringValue(p.Name)
			config.OwnerUserID = types.StringValue(p.OwnerUserID)
			config.IsDefault = types.BoolValue(p.IsDefault)
			resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
			return
		}
	}
	resp.Diagnostics.AddError("Project not found", "No project named "+name)
}
