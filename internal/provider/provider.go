package provider

import (
	"context"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/client"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/datasources"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/providerdata"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/resources"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/wait"
)

var _ provider.Provider = (*MicrosrvProvider)(nil)

type MicrosrvProvider struct {
	version string
}

type providerModel struct {
	Endpoint       types.String `tfsdk:"endpoint"`
	Token          types.String `tfsdk:"token"`
	APIKey         types.String `tfsdk:"api_key"`
	ReadyTimeout   types.String `tfsdk:"ready_timeout"`
	PollInterval   types.String `tfsdk:"poll_interval"`
	RequestTimeout types.String `tfsdk:"request_timeout"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MicrosrvProvider{version: version}
	}
}

func (p *MicrosrvProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "microsrv"
	resp.Version = p.version
}

func (p *MicrosrvProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Microsrv infrastructure via the platform control plane.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Microsrv API base URL (default `https://api.microsrv.ru`). May also be set via MICROSRV_ENDPOINT.",
			},
			"token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Microsrv access token (PAT, `sel_…`), shown once at creation: console → Access → Tokens or `POST /api/v1/tokens`. May also be set via MICROSRV_TOKEN. Mutually exclusive with api_key.",
			},
			"api_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Alias for token: the same Microsrv access token (PAT, `sel_…`). May also be set via MICROSRV_API_KEY. Mutually exclusive with token.",
			},
			"ready_timeout": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Max time to wait for READY / deletion (Go duration, default 15m).",
			},
			"poll_interval": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Polling interval while waiting (Go duration, default 2s).",
			},
			"request_timeout": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "HTTP client timeout per request (Go duration, default 30s).",
			},
		},
	}
}

func (p *MicrosrvProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("MICROSRV_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.microsrv.ru"
	}
	if !config.Endpoint.IsNull() && !config.Endpoint.IsUnknown() {
		endpoint = config.Endpoint.ValueString()
	}

	token := os.Getenv("MICROSRV_TOKEN")
	if !config.Token.IsNull() && !config.Token.IsUnknown() {
		token = config.Token.ValueString()
	}

	apiKey := os.Getenv("MICROSRV_API_KEY")
	if !config.APIKey.IsNull() && !config.APIKey.IsUnknown() {
		apiKey = config.APIKey.ValueString()
	}

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(path.Root("endpoint"), "Missing endpoint", "Set endpoint or MICROSRV_ENDPOINT.")
		return
	}
	if token != "" && apiKey != "" {
		resp.Diagnostics.AddError("Invalid auth", "Provide exactly one of token or api_key (not both).")
		return
	}
	auth := token
	if apiKey != "" {
		auth = apiKey
	}
	if auth == "" {
		resp.Diagnostics.AddError("Missing credentials", "Set token (MICROSRV_TOKEN) or api_key (MICROSRV_API_KEY).")
		return
	}

	reqTimeout := 30 * time.Second
	if !config.RequestTimeout.IsNull() && !config.RequestTimeout.IsUnknown() && config.RequestTimeout.ValueString() != "" {
		d, err := time.ParseDuration(config.RequestTimeout.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("request_timeout"), "Invalid duration", err.Error())
			return
		}
		reqTimeout = d
	}

	waitCfg := wait.Config{Timeout: 15 * time.Minute, Interval: 2 * time.Second}
	if !config.ReadyTimeout.IsNull() && !config.ReadyTimeout.IsUnknown() && config.ReadyTimeout.ValueString() != "" {
		d, err := time.ParseDuration(config.ReadyTimeout.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("ready_timeout"), "Invalid duration", err.Error())
			return
		}
		waitCfg.Timeout = d
	}
	if !config.PollInterval.IsNull() && !config.PollInterval.IsUnknown() && config.PollInterval.ValueString() != "" {
		d, err := time.ParseDuration(config.PollInterval.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("poll_interval"), "Invalid duration", err.Error())
			return
		}
		waitCfg.Interval = d
	}

	c, err := client.New(client.Config{
		Endpoint:       endpoint,
		Token:          auth,
		RequestTimeout: reqTimeout,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client error", err.Error())
		return
	}

	meta := &providerdata.Meta{Client: c, Wait: waitCfg}
	resp.DataSourceData = meta
	resp.ResourceData = meta
}

func (p *MicrosrvProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewProjectResource,
		resources.NewSSHKeyResource,
		resources.NewVPCResource,
		resources.NewVolumeResource,
		resources.NewNetworkInterfaceResource,
		resources.NewVMResource,
		resources.NewContainerImageResource,
		resources.NewContainerResource,
		resources.NewWireGuardPeerResource,
		resources.NewGatewayRouteResource,
		resources.NewLoadBalancerResource,
	}
}

func (p *MicrosrvProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewRegionsDataSource,
		datasources.NewRegionDataSource,
		datasources.NewFlavorsDataSource,
		datasources.NewFlavorDataSource,
		datasources.NewImagesDataSource,
		datasources.NewImageDataSource,
		datasources.NewQuotasDataSource,
		datasources.NewPricingDataSource,
		datasources.NewProjectDataSource,
	}
}
