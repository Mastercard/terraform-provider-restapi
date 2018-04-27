package main

import (
  "github.com/hashicorp/terraform/plugin"
  "github.com/hashicorp/terraform/terraform"
  "github.com/Mastercard/terraform-provider-restapi/restapi"
)

func main() {
  plugin.Serve(&plugin.ServeOpts{
    ProviderFunc: func() terraform.ResourceProvider {
      return restapi.Provider()
    },
  })
}
