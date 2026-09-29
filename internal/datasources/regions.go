package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/providerdata"
)

var _ datasource.DataSource = (*regionsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*regionsDataSource)(nil)

func NewRegionsDataSource() datasource.DataSource { return &regionsDataSource{} }

type regionsDataSource struct {
	meta *providerdata.Meta
}

type regionsModel struct {
	Regions types.List `tfsdk:"regions"`
}

func (d *regionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_regions"
}

func (d *regionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List platform regions.",
		Attributes: map[string]schema.Attribute{
			"regions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: regionAttrs(),
				},
			},
		},
	}
}

func regionAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":      schema.StringAttribute{Computed: true},
		"code":    schema.StringAttribute{Computed: true},
		"name":    schema.StringAttribute{Computed: true},
		"enabled": schema.BoolAttribute{Computed: true},
	}
}

func (d *regionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (d *regionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state regionsModel
	out, err := d.meta.Client.ListRegions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("List regions failed", err.Error())
		return
	}
	elems := make([]attr.Value, 0, len(out))
	for _, r := range out {
		elems = append(elems, types.ObjectValueMust(regionAttrTypes(), map[string]attr.Value{
			"id":      types.StringValue(r.ID),
			"code":    types.StringValue(r.Code),
			"name":    types.StringValue(r.Name),
			"enabled": types.BoolValue(r.Enabled),
		}))
	}
	state.Regions = types.ListValueMust(types.ObjectType{AttrTypes: regionAttrTypes()}, elems)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func regionAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":      types.StringType,
		"code":    types.StringType,
		"name":    types.StringType,
		"enabled": types.BoolType,
	}
}

// Single region lookup

var _ datasource.DataSource = (*regionDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*regionDataSource)(nil)

func NewRegionDataSource() datasource.DataSource { return &regionDataSource{} }

type regionDataSource struct {
	meta *providerdata.Meta
}

type regionModel struct {
	ID      types.String `tfsdk:"id"`
	Code    types.String `tfsdk:"code"`
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func (d *regionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_region"
}

func (d *regionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a platform region by id or code.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Optional: true, Computed: true},
			"code":    schema.StringAttribute{Optional: true, Computed: true},
			"name":    schema.StringAttribute{Computed: true},
			"enabled": schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *regionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (d *regionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config regionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	hasID := !config.ID.IsNull() && config.ID.ValueString() != ""
	hasCode := !config.Code.IsNull() && config.Code.ValueString() != ""
	if hasID == hasCode {
		resp.Diagnostics.AddError("Invalid lookup", "Provide exactly one of id or code.")
		return
	}
	if hasID {
		out, err := d.meta.Client.GetRegion(ctx, config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Get region failed", err.Error())
			return
		}
		config.ID = types.StringValue(out.ID)
		config.Code = types.StringValue(out.Code)
		config.Name = types.StringValue(out.Name)
		config.Enabled = types.BoolValue(out.Enabled)
		resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
		return
	}
	all, err := d.meta.Client.ListRegions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("List regions failed", err.Error())
		return
	}
	code := config.Code.ValueString()
	for _, r := range all {
		if r.Code == code {
			config.ID = types.StringValue(r.ID)
			config.Code = types.StringValue(r.Code)
			config.Name = types.StringValue(r.Name)
			config.Enabled = types.BoolValue(r.Enabled)
			resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
			return
		}
	}
	resp.Diagnostics.AddError("Region not found", "No region with code "+code)
}
