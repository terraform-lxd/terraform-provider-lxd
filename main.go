package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/sl1pm4t/terraform-provider-lxd/lxd"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: lxd.Provider,
	})
}
