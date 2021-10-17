package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/terraform-lxd/terraform-provider-lxd/lxd"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: lxd.Provider,
	})
}
