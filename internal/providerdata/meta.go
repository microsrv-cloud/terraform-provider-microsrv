package providerdata

import (
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/client"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/wait"
)

// Meta is passed from the provider to resources and data sources.
type Meta struct {
	Client *client.Client
	Wait   wait.Config
}
