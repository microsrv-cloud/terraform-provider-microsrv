package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/providerdata"
)

var _ datasource.DataSource = (*flavorsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*flavorsDataSource)(nil)

func NewFlavorsDataSource() datasource.DataSource { return &flavorsDataSource{} }

type flavorsDataSource struct {
	meta *providerdata.Meta
}

type flavorsModel struct {
	Flavors types.List `tfsdk:"flavors"`
}

func (d *flavorsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flavors"
}

func (d *flavorsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List platform VM flavors.",
		Attributes: map[string]schema.Attribute{
			"flavors": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.StringAttribute{Computed: true},
						"code":       schema.StringAttribute{Computed: true},
						"name":       schema.StringAttribute{Computed: true},
						"vcpus":      schema.Int64Attribute{Computed: true},
						"memory_mib": schema.Int64Attribute{Computed: true},
						"available":  schema.BoolAttribute{Computed: true},
						"reason":     schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *flavorsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func flavorAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":         types.StringType,
		"code":       types.StringType,
		"name":       types.StringType,
		"vcpus":      types.Int64Type,
		"memory_mib": types.Int64Type,
		"available":  types.BoolType,
		"reason":     types.StringType,
	}
}

func (d *flavorsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state flavorsModel
	out, err := d.meta.Client.ListFlavors(ctx)
	if err != nil {
		resp.Diagnostics.AddError("List flavors failed", err.Error())
		return
	}
	elems := make([]attr.Value, 0, len(out))
	for _, f := range out {
		elems = append(elems, types.ObjectValueMust(flavorAttrTypes(), map[string]attr.Value{
			"id":         types.StringValue(f.ID),
			"code":       types.StringValue(f.Code),
			"name":       types.StringValue(f.Name),
			"vcpus":      types.Int64Value(int64(f.VCPUs)),
			"memory_mib": types.Int64Value(int64(f.MemoryMiB)),
			"available":  boolOrNull(f.Available),
			"reason":     types.StringValue(f.Reason),
		}))
	}
	state.Flavors = types.ListValueMust(types.ObjectType{AttrTypes: flavorAttrTypes()}, elems)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

var _ datasource.DataSource = (*flavorDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*flavorDataSource)(nil)

func NewFlavorDataSource() datasource.DataSource { return &flavorDataSource{} }

type flavorDataSource struct {
	meta *providerdata.Meta
}

type flavorModel struct {
	ID        types.String `tfsdk:"id"`
	Code      types.String `tfsdk:"code"`
	Name      types.String `tfsdk:"name"`
	VCPUs     types.Int64  `tfsdk:"vcpus"`
	MemoryMiB types.Int64  `tfsdk:"memory_mib"`
	Available types.Bool   `tfsdk:"available"`
	Reason    types.String `tfsdk:"reason"`
}

func (d *flavorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flavor"
}

func (d *flavorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a platform flavor by code.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Computed: true},
			"code":       schema.StringAttribute{Required: true},
			"name":       schema.StringAttribute{Computed: true},
			"vcpus":      schema.Int64Attribute{Computed: true},
			"memory_mib": schema.Int64Attribute{Computed: true},
			"available":  schema.BoolAttribute{Computed: true},
			"reason":     schema.StringAttribute{Computed: true},
		},
	}
}

func (d *flavorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (d *flavorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config flavorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	all, err := d.meta.Client.ListFlavors(ctx)
	if err != nil {
		resp.Diagnostics.AddError("List flavors failed", err.Error())
		return
	}
	code := config.Code.ValueString()
	for _, f := range all {
		if f.Code == code {
			config.ID = types.StringValue(f.ID)
			config.Code = types.StringValue(f.Code)
			config.Name = types.StringValue(f.Name)
			config.VCPUs = types.Int64Value(int64(f.VCPUs))
			config.MemoryMiB = types.Int64Value(int64(f.MemoryMiB))
			config.Available = boolOrNull(f.Available)
			config.Reason = types.StringValue(f.Reason)
			resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
			return
		}
	}
	resp.Diagnostics.AddError("Flavor not found", "No flavor with code "+code)
}
