package incus

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	incus "github.com/lxc/incus/client"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProvider.ResourcesMap["incus_noop"] = resourceIncusNoOp()

	testAccProviders = map[string]*schema.Provider{
		"incus": testAccProvider,
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

func TestAccIncusProvider_envRemote(t *testing.T) {
	envName := os.Getenv("Incus_REMOTE")

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccIncusProvider_basic(envName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_noop.noop1", "remote", envName),
				),
			},
		},
	})
}

func TestAccIncusProvider_imageRemotes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccIncusProvider_basic("ubuntu"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_noop.noop1", "remote", "ubuntu"),
				),
			},
			{
				Config: testAccIncusProvider_basic("ubuntu-daily"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_noop.noop1", "remote", "ubuntu-daily"),
				),
			},
			{
				Config: testAccIncusProvider_basic("images"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_noop.noop1", "remote", "images"),
				),
			},
		},
	})
}

func testAccIncusProvider_basic(remote string) string {
	return fmt.Sprintf(`
resource "incus_noop" "noop1" {
	name = "noop1"
	remote = "%s"
}
`, remote)
}

// this NoOp resource allows us to invoke the Terraform testing framework to test the Provider
// without actually calling out to any Incus server's to create or destroy resources.
func resourceIncusNoOp() *schema.Resource {
	return &schema.Resource{
		Create: func(d *schema.ResourceData, meta interface{}) error {
			p := meta.(*incusProvider)
			remote := p.selectRemote(d)
			_, err := meta.(*incusProvider).GetServer(remote)
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
			p := meta.(*incusProvider)
			remote := p.selectRemote(d)
			_, err := meta.(*incusProvider).GetServer(remote)
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

// getTestIncusInstanceClient retrievs the client from the Incus server or stops the test execution
// otherwise.
func getTestIncusInstanceClient(t *testing.T) incus.InstanceServer {
	testAccProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
	client, err := testAccProvider.Meta().(*incusProvider).GetInstanceServer("")
	if err != nil {
		t.Fatalf("Failed to retrieve test client: %v", err)
	}

	return client
}

func testAccPreCheck(t *testing.T) {
	// NoOp
}

// testAccPreCheckIncusVersion skips the test if the server's version does not satisfy the provided
// version constraints. The version constraints are detailed at:
// https://pkg.go.dev/github.com/hashicorp/go-version#readme-version-constraints
func testAccPreCheckIncusVersion(t *testing.T, versionConstraint string) {
	server, _, err := getTestIncusInstanceClient(t).GetServer()
	if err != nil {
		t.Fatalf("Failed to retrieve the server: %v", err)
	}

	serverVersion := server.Environment.ServerVersion
	ok, err := CheckVersion(serverVersion, versionConstraint)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Skipf("Test %q skipped. Incus version %q does not satisfy the version constraint %q", t.Name(), serverVersion, versionConstraint)
	}
}

// testAccPreCheckAPIExtensions skips the test if the Incus server does not support the required
// extensions.
func testAccPreCheckAPIExtensions(t *testing.T, extensions []string) {
	client := getTestIncusInstanceClient(t)

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

// testAccPreCheckVirtualization skips the test if the Incus server does not support virtualization.
func testAccPreCheckVirtualization(t *testing.T) {
	server, _, err := getTestIncusInstanceClient(t).GetServer()
	if err != nil {
		t.Fatalf("Failed to retrieve the server: %v", err)
	}

	// Ensure that Incus server supports qemu driver which is required for virtualization.
	if !strings.Contains(server.Environment.Driver, "qemu") {
		t.Skipf("Test %q skipped. Server does not support virtualization.", t.Name())
	}
}

// testAccPreCheckClustering skips the test if Incus is not running in clustered mode.
func testAccPreCheckClustering(t *testing.T) {
	client := getTestIncusInstanceClient(t)
	if !client.IsClustered() {
		t.Skipf("Test %q skipped. Server is not running in clustered mode.", t.Name())
	}
}
