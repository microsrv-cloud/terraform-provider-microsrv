package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/providerdata"
)

var _ datasource.DataSource = (*imagesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*imagesDataSource)(nil)

func NewImagesDataSource() datasource.DataSource { return &imagesDataSource{} }

type imagesDataSource struct {
	meta *providerdata.Meta
}

type imagesModel struct {
	Images types.List `tfsdk:"images"`
}

func (d *imagesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_images"
}

func imageAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                types.StringType,
		"code":              types.StringType,
		"name":              types.StringType,
		"min_boot_size_mib": types.Int64Type,
	}
}

func (d *imagesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List platform images (codes used as `microsrv_volume` `clone_from`).",
		Attributes: map[string]schema.Attribute{
			"images": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                schema.StringAttribute{Computed: true},
						"code":              schema.StringAttribute{Computed: true},
						"name":              schema.StringAttribute{Computed: true},
						"min_boot_size_mib": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *imagesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (d *imagesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state imagesModel
	out, err := d.meta.Client.ListImages(ctx)
	if err != nil {
		resp.Diagnostics.AddError("List images failed", err.Error())
		return
	}
	elems := make([]attr.Value, 0, len(out))
	for _, img := range out {
		elems = append(elems, types.ObjectValueMust(imageAttrTypes(), map[string]attr.Value{
			"id":                types.StringValue(img.ID),
			"code":              types.StringValue(img.Code),
			"name":              types.StringValue(img.Name),
			"min_boot_size_mib": types.Int64Value(img.MinBootSizeMiB),
		}))
	}
	state.Images = types.ListValueMust(types.ObjectType{AttrTypes: imageAttrTypes()}, elems)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

var _ datasource.DataSource = (*imageDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*imageDataSource)(nil)

func NewImageDataSource() datasource.DataSource { return &imageDataSource{} }

type imageDataSource struct {
	meta *providerdata.Meta
}

type imageModel struct {
	ID             types.String `tfsdk:"id"`
	Code           types.String `tfsdk:"code"`
	Name           types.String `tfsdk:"name"`
	MinBootSizeMiB types.Int64  `tfsdk:"min_boot_size_mib"`
}

func (d *imageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (d *imageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a platform image by code.",
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Computed: true},
			"code":              schema.StringAttribute{Required: true},
			"name":              schema.StringAttribute{Computed: true},
			"min_boot_size_mib": schema.Int64Attribute{Computed: true},
		},
	}
}

func (d *imageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.meta = castMeta(req.ProviderData, &resp.Diagnostics)
}

func (d *imageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config imageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	all, err := d.meta.Client.ListImages(ctx)
	if err != nil {
		resp.Diagnostics.AddError("List images failed", err.Error())
		return
	}
	code := config.Code.ValueString()
	for _, img := range all {
		if img.Code == code {
			config.ID = types.StringValue(img.ID)
			config.Code = types.StringValue(img.Code)
			config.Name = types.StringValue(img.Name)
			config.MinBootSizeMiB = types.Int64Value(img.MinBootSizeMiB)
			resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
			return
		}
	}
	resp.Diagnostics.AddError("Image not found", "No image with code "+code)
}
