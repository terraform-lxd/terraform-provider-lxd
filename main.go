package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/provider"
)

// version indicates provider's version. The appropriate value
// for the compiled binary will be set by the goreleaser.
// See: https://goreleaser.com/cookbooks/using-main.version/
var version = "dev"

// Old main for SDKv2.
//
// func main() {
// 	plugin.Serve(&plugin.ServeOpts{
// 		ProviderFunc: lxd.Provider,
// 	})
// }

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider in debug mode")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address:         "registry.terraform.io/terraform-lxd/lxd",
		Debug:           debug,
		ProtocolVersion: 6,
	}

	err := providerserver.Serve(context.Background(), provider.NewLxdProvider(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
