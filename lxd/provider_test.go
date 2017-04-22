package lxd

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"os/exec"
	"testing"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"lxd": testAccProvider,
	}

	testAccProvider.ConfigureFunc = testProviderConfigureWrapper
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testProviderConfigureWrapper(d *schema.ResourceData) (interface{}, error) {
	d.Set("refresh_interval", "5s")
	return providerConfigure(d)
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func testAccPreCheck(t *testing.T) {
	cmd := exec.Command("lxc", "--version")
	if err := cmd.Run(); err != nil {
		t.Fatalf("LXD client must be available: %s", err)
	}
}
