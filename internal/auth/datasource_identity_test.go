package auth_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccIdentity_DS_bearer(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management", "auth_bearer")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create identity.
				Config: acctest.Provider() + testAccIdentity_DS_bearer(identity, []string{}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "name", identity),
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "auth_method", "bearer"),
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "groups.#", "0"),
				),
			},
			{
				// Update groups.
				Config: acctest.Provider() + testAccIdentity_DS_bearer(identity, []string{"admins"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "name", identity),
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "auth_method", "bearer"),
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "groups.#", "1"),
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "groups.0", "admins"),
				),
			},
		},
	})
}

func TestAccIdentity_DS_tls(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"tls": {
				VersionConstraint: "~> 4.0",
				Source:            "hashicorp/tls",
			},
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create identity.
				Config: acctest.Provider() + testAccIdentity_DS_tls(identity, []string{}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "name", identity),
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "auth_method", "tls"),
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "groups.#", "0"),
					resource.TestCheckResourceAttrSet("data.lxd_auth_identity.identity", "tls_certificate"),
				),
			},
			{
				// Update groups.
				Config: acctest.Provider() + testAccIdentity_DS_tls(identity, []string{"admins"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "name", identity),
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "auth_method", "tls"),
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "groups.#", "1"),
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "groups.0", "admins"),
					resource.TestCheckResourceAttrSet("data.lxd_auth_identity.identity", "tls_certificate"),
				),
			},
		},
	})
}

func testAccIdentity_DS_bearer(name string, groups []string) string {
	return testAccIdentity_bearer(name, groups) + `
                data "lxd_auth_identity" "identity" {
                  auth_method = "bearer"
                  name        = lxd_auth_identity.identity.name
                }
        `
}

func testAccIdentity_DS_tls(name string, groups []string) string {
	return testAccIdentity_tls(name, groups) + `
                data "lxd_auth_identity" "identity" {
                  auth_method = "tls"
                  name        = lxd_auth_identity.identity.name
                }
        `
}
