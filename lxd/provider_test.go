package lxd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	lxd "github.com/canonical/lxd/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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

// getTestLXDInstanceClient retrievs the client from the LXD server or stops the test execution
// otherwise.
func getTestLXDInstanceClient(t *testing.T) lxd.InstanceServer {
	testAccProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
	client, err := testAccProvider.Meta().(*lxdProvider).GetInstanceServer("")
	if err != nil {
		t.Fatalf("Failed to retrieve test client: %v", err)
	}

	return client
}

func testAccPreCheck(t *testing.T) {
	// NoOp
}

// testAccPreCheckLxdVersion skips the test if the server's version does not satisfy the provided
// version constraints. The version constraints are detailed at:
// https://pkg.go.dev/github.com/hashicorp/go-version#readme-version-constraints
func testAccPreCheckLxdVersion(t *testing.T, versionConstraint string) {
	server, _, err := getTestLXDInstanceClient(t).GetServer()
	if err != nil {
		t.Fatalf("Failed to retrieve the server: %v", err)
	}

	serverVersion := server.Environment.ServerVersion
	ok, err := CheckVersion(serverVersion, versionConstraint)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Skipf("Test %q skipped. LXD version %q does not satisfy the version constraint %q", t.Name(), serverVersion, versionConstraint)
	}
}

// testAccPreCheckAPIExtensions skips the test if the LXD server does not support the required
// extensions.
func testAccPreCheckAPIExtensions(t *testing.T, extensions []string) {
	client := getTestLXDInstanceClient(t)

	missing := []string{}
	for _, e := range extensions {
		if !client.HasExtension(e) {
			missing = append(missing, e)
		}
	}

	if len(missing) > 0 {
		t.Skipf("Test %q skipped. Missing required extensions: %v", t.Name(), missing)
	}
}

// testAccPreCheckVirtualization skips the test if the LXD server does not support virtualization.
func testAccPreCheckVirtualization(t *testing.T) {
	server, _, err := getTestLXDInstanceClient(t).GetServer()
	if err != nil {
		t.Fatalf("Failed to retrieve the server: %v", err)
	}

	// Ensure that LXD server supports qemu driver which is required for virtualization.
	if !strings.Contains(server.Environment.Driver, "qemu") {
		t.Skipf("Test %q skipped. Server does not support virtualization.", t.Name())
	}
}

// testAccPreCheckClustering skips the test if LXD is not running in clustered mode.
func testAccPreCheckClustering(t *testing.T) {
	client := getTestLXDInstanceClient(t)
	if !client.IsClustered() {
		t.Skipf("Test %q skipped. Server is not running in clustered mode.", t.Name())
	}
}
