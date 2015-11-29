package lxd

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/gpmgo/gopm/modules/log"
	"github.com/lxc/lxd"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {

	// The actual provider
	return &schema.Provider{
		Schema: map[string]*schema.Schema{

			"address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["lxd_address"],
				Default:     "/var/lib/lxd/unix.socket",
			},

			"scheme": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "unix",
				Description: descriptions["lxd_scheme"],
			},

			"port": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "8443",
				Description: descriptions["lxd_port"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"lxd_container": resourceLxdContainer(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"lxd_address": "The FQDN or IP where the LXD daemon can be contacted. (default = /var/lib/lxd/unix.socket)",
		"lxd_scheme":  "unix or https (default = unix)",
		"lxd_port":    "Port LXD Daemon is listening on (default 8443).",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	daemon_addr := ""
	scheme := d.Get("scheme")
	switch scheme {
	case "unix":
		daemon_addr = fmt.Sprintf("unix:%s", d.Get("address"))
	case "https":
		daemon_addr = fmt.Sprintf("https://%s:%s", d.Get("address"), d.Get("port"))
	default:
		log.Fatal("Invalid scheme: %s", scheme)
	}

	config := lxd.Config{
		Remotes: map[string]lxd.RemoteConfig{
			"terraform": lxd.RemoteConfig{Addr: daemon_addr},
		},
	}

	client, err := lxd.NewClient(&config, "terraform")
	if err != nil {
		log.Error("Could not create LXD client: %s", err)
		return nil, err
	}

	if err := validateClient(client); err != nil {
		return nil, err
	}

	return client, nil
}

func validateClient(client *lxd.Client) error {
	return client.Finger()
}
