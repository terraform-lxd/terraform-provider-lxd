package auth_test

import (
	"fmt"
	"regexp"
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
					resource.TestCheckResourceAttrSet("data.lxd_auth_identity.identity", "identifier"),
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
			{
				// Look up a client bearer identity as a devlxd identity.
				Config:      acctest.Provider() + testAccIdentity_DS_bearerAsDevlxd(identity, []string{"admins"}),
				ExpectError: regexp.MustCompile(`LXD identity type "Client token bearer"`),
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
					resource.TestCheckResourceAttrSet("data.lxd_auth_identity.identity", "identifier"),
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

func TestAccIdentity_DS_tlsPending(t *testing.T) {
	identity := acctest.GenerateName(2, "-")
	dataSourceName := "data.lxd_auth_identity.identity"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management", "access_management_tls")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// A pending identity has no certificate, and its identifier is a UUID that LXD replaces with
				// the certificate fingerprint once the trust token is redeemed.
				Config: acctest.Provider() + testAccIdentity_DS_tlsPending(identity),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", identity),
					resource.TestCheckResourceAttr(dataSourceName, "auth_method", "tls"),
					resource.TestCheckResourceAttr(dataSourceName, "tls_certificate", ""),
					resource.TestCheckResourceAttrSet(dataSourceName, "identifier"),
				),
			},
		},
	})
}

func TestAccIdentity_DS_devlxd(t *testing.T) {
	identity := acctest.GenerateName(2, "-")
	dataSourceName := "data.lxd_auth_identity.identity"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management", "auth_bearer_devlxd")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Look up by identity type.
				Config: acctest.Provider() + testAccIdentity_DS_devlxdByType(identity),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", identity),
					resource.TestCheckResourceAttr(dataSourceName, "type", "devlxd"),
					resource.TestCheckResourceAttr(dataSourceName, "auth_method", "bearer"),
				),
			},
			{
				// Look up by authentication method, which does not distinguish
				// client bearer identities from devlxd ones.
				Config: acctest.Provider() + testAccIdentity_DS_devlxdByAuthMethod(identity),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", identity),
					resource.TestCheckResourceAttr(dataSourceName, "type", "devlxd"),
					resource.TestCheckResourceAttr(dataSourceName, "auth_method", "bearer"),
				),
			},
			{
				// Look up a devlxd identity as a client bearer identity.
				Config:      acctest.Provider() + testAccIdentity_DS_devlxdAsBearer(identity),
				ExpectError: regexp.MustCompile(`LXD identity type "DevLXD token bearer"`),
			},
		},
	})
}

func TestAccIdentity_DS_typeValidation(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      acctest.Provider() + testAccIdentity_DS_noAttributes(identity),
				ExpectError: regexp.MustCompile(`Attribute "type" must be set`),
			},
			{
				Config:      acctest.Provider() + testAccIdentity_DS_bothAttributes(identity),
				ExpectError: regexp.MustCompile(`cannot both be set`),
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

func testAccIdentity_DS_tlsPending(name string) string {
	return testAccIdentity_tlsPending(name, []string{}) + `
		data "lxd_auth_identity" "identity" {
		  auth_method = "tls"
		  name        = lxd_auth_identity.identity.name
		}
	`
}

func testAccIdentity_DS_devlxdByType(name string) string {
	return testAccIdentity_type(name, "devlxd", []string{}) + `
                data "lxd_auth_identity" "identity" {
                  type = "devlxd"
                  name = lxd_auth_identity.identity.name
                }
        `
}

func testAccIdentity_DS_devlxdByAuthMethod(name string) string {
	return testAccIdentity_type(name, "devlxd", []string{}) + `
                data "lxd_auth_identity" "identity" {
                  auth_method = "bearer"
                  name        = lxd_auth_identity.identity.name
                }
        `
}

func testAccIdentity_DS_devlxdAsBearer(name string) string {
	return testAccIdentity_type(name, "devlxd", []string{}) + `
                data "lxd_auth_identity" "identity" {
                  type = "bearer"
                  name = lxd_auth_identity.identity.name
                }
        `
}

func testAccIdentity_DS_bearerAsDevlxd(name string, groups []string) string {
	return testAccIdentity_type(name, "bearer", groups) + `
                data "lxd_auth_identity" "identity" {
                  type = "devlxd"
                  name = lxd_auth_identity.identity.name
                }
        `
}

func testAccIdentity_DS_noAttributes(name string) string {
	return fmt.Sprintf(`
                data "lxd_auth_identity" "identity" {
                  name = %q
                }
        `,
		name,
	)
}

func testAccIdentity_DS_bothAttributes(name string) string {
	return fmt.Sprintf(`
                data "lxd_auth_identity" "identity" {
		  auth_method = "bearer"
		  type        = "bearer"
                  name        = %q
                }
        `,
		name,
	)
}
