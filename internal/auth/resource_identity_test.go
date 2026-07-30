package auth_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

func TestAccIdentity_tlsPending(t *testing.T) {
	identity := acctest.GenerateName(2, "-")
	resourceName := "lxd_auth_identity.identity"

	clientCert, clientKey, cleanupCert := acctest.GenerateClientCertificate(t)
	defer cleanupCert()

	var token string

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management", "access_management_tls")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Omitting the certificate creates a pending identity and issues a trust token.
				Config: acctest.Provider() + testAccIdentity_tlsPending(identity, []string{"admins"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", identity),
					resource.TestCheckResourceAttr(resourceName, "auth_method", "tls"),
					resource.TestCheckResourceAttr(resourceName, "groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "groups.0", "admins"),
					resource.TestCheckNoResourceAttr(resourceName, "tls_certificate"),
					resource.TestCheckResourceAttrSet(resourceName, "trust_token"),
					testAccIdentity_captureAttr(resourceName, "trust_token", &token),
				),
			},
			{
				// Updating an identity that is still pending must preserve its trust token, which is
				// issued once and cannot be retrieved from the server.
				Config: acctest.Provider() + testAccIdentity_tlsPending(identity, []string{}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "groups.#", "0"),
					resource.TestCheckResourceAttrPtr(resourceName, "trust_token", &token),
				),
			},
			{
				// Redeeming the token enrolls the client certificate, which moves the identity out of the
				// pending state.
				// The consumed token is cleared and the identity must converge without producing a diff.
				PreConfig: func() {
					acctest.RedeemTrustToken(t, token, clientCert, clientKey)
				},
				Config: acctest.Provider() + testAccIdentity_tlsPending(identity, []string{"admins"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", identity),
					resource.TestCheckResourceAttr(resourceName, "groups.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "tls_certificate"),
					resource.TestCheckNoResourceAttr(resourceName, "trust_token"),
					resource.TestCheckNoResourceAttr(resourceName, "expires_at"),
				),
			},
			{
				// Updating a redeemed identity must not clear its certificate.
				Config: acctest.Provider() + testAccIdentity_tlsPending(identity, []string{}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", identity),
					resource.TestCheckResourceAttr(resourceName, "groups.#", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "tls_certificate"),
				),
			},
		},
	})
}

func TestAccIdentity_tlsPendingCertificate(t *testing.T) {
	identity := acctest.GenerateName(2, "-")
	resourceName := "lxd_auth_identity.identity"

	clientCert, _, cleanupCert := acctest.GenerateClientCertificate(t)
	defer cleanupCert()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management", "access_management_tls")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Omitting the certificate creates a pending identity and issues a trust token.
				Config: acctest.Provider() + testAccIdentity_tlsPending(identity, []string{"admins"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", identity),
					resource.TestCheckNoResourceAttr(resourceName, "tls_certificate"),
					resource.TestCheckResourceAttrSet(resourceName, "trust_token"),
				),
			},
			{
				// LXD refuses a certificate on an identity that is still pending, so configuring one replaces
				// the identity instead of updating it.
				// The outstanding trust token is invalidated along with the old identity, and the replacement
				// is active from the start.
				Config: acctest.Provider() + testAccIdentity_tlsPendingWithCertificate(identity, []string{"admins"}, clientCert),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", identity),
					resource.TestCheckResourceAttr(resourceName, "groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tls_certificate", clientCert),
					resource.TestCheckNoResourceAttr(resourceName, "trust_token"),
					resource.TestCheckNoResourceAttr(resourceName, "expires_at"),
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

// testAccIdentity_captureAttr stores the value of the given resource attribute in target, so that
// it can be used by a subsequent test step.
func testAccIdentity_captureAttr(resourceName string, attrName string, target *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		res, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource %q not found", resourceName)
		}

		value, ok := res.Primary.Attributes[attrName]
		if !ok {
			return fmt.Errorf("Attribute %q not found in resource %q", attrName, resourceName)
		}

		*target = value

		return nil
	}
}

func testAccIdentity_tlsPending(name string, groups []string) string {
	return fmt.Sprintf(`
		resource "lxd_auth_identity" "identity" {
		  auth_method = "tls"
		  name        = %q
		  groups      = [%s]
		}
	`,
		name,
		acctest.QuoteStrings(groups),
	)
}

func testAccIdentity_tlsPendingWithCertificate(name string, groups []string, certificate string) string {
	return fmt.Sprintf(`
		resource "lxd_auth_identity" "identity" {
		  auth_method = "tls"
		  name        = %q
		  groups      = [%s]

		  tls_certificate = %q
		}
	`,
		name,
		acctest.QuoteStrings(groups),
		certificate,
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
