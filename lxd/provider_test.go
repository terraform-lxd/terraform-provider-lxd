package lxd

import (
	"os"
	"testing"

	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProvider.ResourcesMap["lxd_noop"] = resourceLxdNoOp()

	testAccProviders = map[string]*schema.Provider{
		"lxd": testAccProvider,
	}

	testAccProvider.ConfigureFunc = testProviderConfigureWrapper
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testProviderConfigureWrapper(d *schema.ResourceData) (interface{}, error) {
	d.Set("refresh_interval", "5s")
	return providerConfigure(d)
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func TestAccLxdProvider_envRemote(t *testing.T) {
	envName := os.Getenv("LXD_REMOTE")

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLxdProvider_basic(envName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", envName),
				),
			},
		},
	})
}

func TestAccLxdProvider_imageRemotes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLxdProvider_basic("ubuntu"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", "ubuntu"),
				),
			},
			{
				Config: testAccLxdProvider_basic("ubuntu-daily"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", "ubuntu-daily"),
				),
			},
			{
				Config: testAccLxdProvider_basic("images"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", "images"),
				),
			},
		},
	})
}

func testAccPreCheck(t *testing.T) {
	// NoOp
}

func testAccLxdProvider_basic(remote string) string {
	return fmt.Sprintf(`
resource "lxd_noop" "noop1" {
	name = "noop1"
	remote = "%s"
}
`, remote)
}

// this NoOp resource allows us to invoke the Terraform testing framework to test the Provider
// without actually calling out to any LXD server's to create or destroy resources.
func resourceLxdNoOp() *schema.Resource {
	return &schema.Resource{
		Create: func(d *schema.ResourceData, meta interface{}) error {
			p := meta.(*lxdProvider)
			remote := p.selectRemote(d)
			_, err := meta.(*lxdProvider).GetServer(remote)
			if err != nil {
				return err
			}

			d.SetId(d.Get("name").(string))
			d.Set("name", d.Get("name"))
			d.Set("client_name", remote)
			d.Set("remote", d.Get("remote"))
			return nil
		},

		Delete: schema.RemoveFromState,

		Read: func(d *schema.ResourceData, meta interface{}) error {
			p := meta.(*lxdProvider)
			remote := p.selectRemote(d)
			_, err := meta.(*lxdProvider).GetServer(remote)
			if err != nil {
				return err
			}

			d.Set("name", d.Get("name"))
			d.Set("remote", d.Get("remote"))
			d.Set("client_name", remote)
			return nil
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"remote": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
			"client_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
