package auth_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccIdentity_bearer(t *testing.T) {
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
				Config: acctest.Provider() + testAccIdentity_bearer(identity, []string{}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "name", identity),
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "auth_method", "bearer"),
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "groups.#", "0"),
				),
			},
			{
				// Update groups.
				Config: acctest.Provider() + testAccIdentity_bearer(identity, []string{"admins"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "name", identity),
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "auth_method", "bearer"),
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "groups.#", "1"),
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "groups.0", "admins"),
				),
			},
		},
	})
}

func TestAccIdentity_tls(t *testing.T) {
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
				Config: acctest.Provider() + testAccIdentity_tls(identity, []string{}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "name", identity),
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "auth_method", "tls"),
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "groups.#", "0"),
					resource.TestCheckResourceAttrSet("lxd_auth_identity.identity", "tls_certificate"),
				),
			},
			{
				// Update groups.
				Config: acctest.Provider() + testAccIdentity_tls(identity, []string{"admins"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "name", identity),
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "auth_method", "tls"),
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "groups.#", "1"),
					resource.TestCheckResourceAttr("lxd_auth_identity.identity", "groups.0", "admins"),
					resource.TestCheckResourceAttrSet("lxd_auth_identity.identity", "tls_certificate"),
				),
			},
		},
	})
}

func TestAccIdentity_importEmpty(t *testing.T) {
	resourceName := "lxd_auth_identity.identity"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management", "auth_bearer")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccIdentity_bearer("tf-auth-identity", []string{}),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        "/bearer/tf-auth-identity",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccIdentity_importWithGroups(t *testing.T) {
	resourceName := "lxd_auth_identity.identity"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"tls": {
				VersionConstraint: "~> 4.0",
				Source:            "hashicorp/tls",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccIdentity_tls("tf-auth-identity", []string{"admins"}),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        "/tls/tf-auth-identity",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func testAccIdentity_bearer(name string, groups []string) string {
	return fmt.Sprintf(`
                resource "lxd_auth_identity" "identity" {
		  auth_method = "bearer"
                  name        = %q
                  groups      = [%s]
                }
        `,
		name,
		acctest.QuoteStrings(groups),
	)
}

func testAccIdentity_tls(name string, groups []string) string {
	return fmt.Sprintf(`
		resource "tls_private_key" "key" {
		  algorithm = "ED25519"
		}

		resource "tls_self_signed_cert" "cert" {
		  private_key_pem       = tls_private_key.key.private_key_pem
		  validity_period_hours = 1

		  subject {
		    common_name = "localhost"
		  }

		  allowed_uses = [
		    "digital_signature"
		  ]
		}

                resource "lxd_auth_identity" "identity" {
		  auth_method = "tls"
                  name        = %q
		  groups      = [%s]

                  tls_certificate = tls_self_signed_cert.cert.cert_pem
                }
        `,
		name,
		acctest.QuoteStrings(groups),
	)
}
