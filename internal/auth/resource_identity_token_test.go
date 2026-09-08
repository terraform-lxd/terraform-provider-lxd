package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccIdentityToken_bearer(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue token.
				Config: acctest.Provider() + testAccIdentityToken(identity, "bearer", "1H"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_token.token", "identity", identity),
					resource.TestCheckResourceAttr("lxd_auth_identity_token.token", "expiry", "1H"),
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
			{
				// Change expiry to issue a new token.
				Config: acctest.Provider() + testAccIdentityToken(identity, "bearer", "2H"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_token.token", "expiry", "2H"),
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
		},
	})
}

func TestAccIdentityToken_revoked(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue token.
				Config: acctest.Provider() + testAccIdentityToken(identity, "bearer", "1H"),
			},
			{
				// A token that is revoked outside of Terraform is no longer
				// reported by the server and must be reissued.
				PreConfig: func() {
					server := acctest.InstanceServer(t)

					err := server.RevokeBearerIdentityToken(identity)
					if err != nil {
						t.Fatalf("Failed to revoke token of identity %q: %v", identity, err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				RefreshPlanChecks: resource.RefreshPlanChecks{
					PostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("lxd_auth_identity_token.token", plancheck.ResourceActionCreate),
					},
				},
			},
			{
				// Reissue token.
				Config: acctest.Provider() + testAccIdentityToken(identity, "bearer", "1H"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
		},
	})
}

func TestAccIdentityToken_expired(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue a short lived token.
				Config: acctest.Provider() + testAccIdentityToken(identity, "bearer", "10S"),
			},
			{
				// An expired token must be reissued, even though the server
				// keeps reporting its expiry.
				PreConfig: func() {
					time.Sleep(11 * time.Second)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				RefreshPlanChecks: resource.RefreshPlanChecks{
					PostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("lxd_auth_identity_token.token", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccIdentityToken_destroyed(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue token and read the identity that bears it. The identity
				// must report the expiry of the issued token.
				Config: acctest.Provider() + testAccIdentityToken_identityDataSource(identity, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "name", identity),
					resource.TestCheckResourceAttrWith("data.lxd_auth_identity.identity", "expires_at", checkExpiresInFuture),
					resource.TestCheckResourceAttrPair("data.lxd_auth_identity.identity", "expires_at", "lxd_auth_identity_token.token", "expires_at"),
				),
			},
			{
				// Remove just the token resource, which revokes the token while
				// the identity that bears it remains.
				Config: acctest.Provider() + testAccIdentityToken_identityDataSource(identity, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "name", identity),
				),
			},
			{
				// Read the identity once more, now that its token is revoked.
				// The identity must report no expiry.
				Config: acctest.Provider() + testAccIdentityToken_identityDataSource(identity, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_identity.identity", "name", identity),
					resource.TestCheckNoResourceAttr("data.lxd_auth_identity.identity", "expires_at"),
				),
			},
		},
	})
}

func TestAccIdentityToken_serverExpiry(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue token without an explicit expiry, which makes the
				// server apply its default.
				Config: acctest.Provider() + testAccIdentityToken(identity, "bearer", ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_token.token", "identity", identity),
					resource.TestCheckNoResourceAttr("lxd_auth_identity_token.token", "expiry"),
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
		},
	})
}

func TestAccIdentityToken_createBeforeDestroy(t *testing.T) {
	identity := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management_expiry")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Issue token.
				Config: acctest.Provider() + testAccIdentityToken_createBeforeDestroy(identity, "1H"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
			{
				// Ensure token created with create-before-destroy (or outside the Terraform)
				// is not revoked.
				Config: acctest.Provider() + testAccIdentityToken_createBeforeDestroy(identity, "2H"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_token.token", "expiry", "2H"),
					resource.TestCheckResourceAttrSet("lxd_auth_identity_token.token", "token"),
					resource.TestCheckResourceAttrWith("lxd_auth_identity_token.token", "expires_at", checkExpiresInFuture),
				),
			},
		},
	})
}

// checkExpiresInFuture ensures the expiry is an RFC3339 timestamp in the future.
func checkExpiresInFuture(value string) error {
	expiresAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return fmt.Errorf("Expected an RFC3339 timestamp, got %q: %w", value, err)
	}

	if !expiresAt.After(time.Now()) {
		return fmt.Errorf("Expected expiry %q to be in the future", value)
	}

	return nil
}

func testAccIdentityToken_identityDataSource(name string, withToken bool) string {
	token := ""
	dependsOn := ""

	if withToken {
		token = `
		resource "lxd_auth_identity_token" "token" {
		  identity = lxd_auth_identity.identity.name
		  expiry   = "1H"
		}
		`

		// Without an explicit dependency the identity is read before the
		// token is issued, and therefore reports no expiry.
		dependsOn = "depends_on = [lxd_auth_identity_token.token]"
	}

	return fmt.Sprintf(`
		resource "lxd_auth_identity" "identity" {
		  type = "bearer"
		  name = %q
		}
		%s
		data "lxd_auth_identity" "identity" {
		  type = "bearer"
		  name = lxd_auth_identity.identity.name

		  %s
		}
	`,
		name,
		token,
		dependsOn,
	)
}

func testAccIdentityToken_createBeforeDestroy(name string, expiry string) string {
	return fmt.Sprintf(`
		resource "lxd_auth_identity" "identity" {
		  type = "bearer"
		  name = %q
		}

		resource "lxd_auth_identity_token" "token" {
		  identity = lxd_auth_identity.identity.name
		  expiry   = %q

		  lifecycle {
		    create_before_destroy = true
		  }
		}
	`,
		name,
		expiry,
	)
}

func testAccIdentityToken(name string, identityType string, expiry string) string {
	tokenExpiry := ""
	if expiry != "" {
		tokenExpiry = fmt.Sprintf("expiry = %q", expiry)
	}

	return fmt.Sprintf(`
		resource "lxd_auth_identity" "identity" {
		  type = %q
		  name = %q
		}

		resource "lxd_auth_identity_token" "token" {
		  identity = lxd_auth_identity.identity.name
		  %s
		}
	`,
		identityType,
		name,
		tokenExpiry,
	)
}
