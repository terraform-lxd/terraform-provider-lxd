package server_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccServer_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckServerTests(t)
			acctest.PreCheckAPIExtensions(t, "metadata_configuration")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"user.terraform-provider-test": "foo",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test", "foo"),
				),
			},
			{
				// Revert the managed key to its default.
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"user.terraform-provider-test": "",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test", ""),
				),
			},
		},
	})
}

func TestAccServer_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckServerTests(t)
			acctest.PreCheckAPIExtensions(t, "metadata_configuration")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"user.terraform-provider-test": "foo",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test", "foo"),
				),
			},
			{
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"user.terraform-provider-test":  "bar",
					"user.terraform-provider-test2": "baz",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test", "bar"),
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test2", "baz"),
				),
			},
			{
				// Revert the managed keys to their defaults.
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"user.terraform-provider-test":  "",
					"user.terraform-provider-test2": "",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test", ""),
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test2", ""),
				),
			},
		},
	})
}

func TestAccServer_localKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckServerTests(t)
			acctest.PreCheckAPIExtensions(t, "metadata_configuration")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"core.bgp_routerid": "127.0.0.1",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.core.bgp_routerid", "127.0.0.1"),
				),
			},
			{
				// Revert the managed key to its default.
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"core.bgp_routerid": "",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.core.bgp_routerid", ""),
				),
			},
		},
	})
}

func TestAccServer_invalidKeyRejected(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckServerTests(t)
			acctest.PreCheckAPIExtensions(t, "metadata_configuration")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"not.a.real.key": "value",
				}),
				ExpectError: regexp.MustCompile("is not a valid server configuration key"),
			},
		},
	})
}

func testAccServer_config(config map[string]string) string {
	var b strings.Builder
	b.WriteString(`
resource "lxd_server" "global" {
  config = {
`)

	for key, value := range config {
		fmt.Fprintf(&b, "    %q = %q\n", key, value)
	}

	b.WriteString("  }\n}\n")

	return b.String()
}
