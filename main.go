package main

import (
	"github.com/Mastercard/terraform-provider-restapi/restapi"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return restapi.Provider()
		},
	})
}
