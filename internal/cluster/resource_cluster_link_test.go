package cluster_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
)

func TestAccClusterLink_pending(t *testing.T) {
	linkName := acctest.GenerateName(2, "-")
	resourceName := "lxd_cluster_link.link"

	var token string

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "cluster_links")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Omitting the trust token creates a pending link and issues a token for the linked cluster.
				Config: acctest.Provider() + testAccClusterLink_pending(linkName, "Backup cluster", []string{"admins"}, "test"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", linkName),
					resource.TestCheckResourceAttr(resourceName, "type", "bidirectional"),
					resource.TestCheckResourceAttr(resourceName, "description", "Backup cluster"),
					resource.TestCheckResourceAttr(resourceName, "auth_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "auth_groups.0", "admins"),
					resource.TestCheckResourceAttr(resourceName, "config.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "config.user.env", "test"),
					resource.TestCheckResourceAttr(resourceName, "addresses.#", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "trust_token"),
					testAccClusterLink_checkTrustToken(resourceName, linkName),
					testAccClusterLink_captureAttr(resourceName, "trust_token", &token),
				),
			},
			{
				// Updating a link that is still pending must preserve its trust token, which is issued
				// once and cannot be retrieved from the server.
				Config: acctest.Provider() + testAccClusterLink_pending(linkName, "Backup cluster in DC2", []string{}, "prod"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Backup cluster in DC2"),
					resource.TestCheckResourceAttr(resourceName, "auth_groups.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "config.user.env", "prod"),
					resource.TestCheckResourceAttrPtr(resourceName, "trust_token", &token),
				),
			},
			{
				// The trust token cannot be read back from the server.
				ResourceName:                         resourceName,
				ImportStateId:                        linkName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"trust_token", "expires_at"},
			},
		},
	})
}

func TestAccClusterLink_validation(t *testing.T) {
	linkName := acctest.GenerateName(2, "-")

	// Every combination is rejected before anything is created, most of them while planning.
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Only a bidirectional link has an identity on this cluster.
				Config:      acctest.Provider() + testAccClusterLink_attrs(linkName, `type = "public"`, `remote_address = "127.0.0.1:8443"`, `auth_groups = ["admins"]`),
				ExpectError: testAccClusterLink_errorRegexp(`Auth groups cannot be set for "public" cluster links`),
			},
			{
				// Without a token, LXD would create a pending identity instead of a link.
				Config:      acctest.Provider() + testAccClusterLink_attrs(linkName, `type = "unidirectional"`),
				ExpectError: testAccClusterLink_errorRegexp(`Unidirectional cluster links require a trust token`),
			},
			{
				// An empty token would make LXD issue one that the provider could not store.
				Config:      acctest.Provider() + testAccClusterLink_attrs(linkName, `trust_token = ""`),
				ExpectError: testAccClusterLink_errorRegexp(`string length must be at least 1`),
			},
			{
				// Volatile keys are managed by LXD.
				Config:      acctest.Provider() + testAccClusterLink_attrs(linkName, `config = { "volatile.addresses" = "127.0.0.1:8443" }`),
				ExpectError: testAccClusterLink_errorRegexp(`must be a user.* key`),
			},
			{
				Config:      acctest.Provider() + testAccClusterLink_attrs(linkName, `remote_address = "127.0.0.1:8443"`),
				ExpectError: testAccClusterLink_errorRegexp(`Remote address and fingerprint can only be set for "public" cluster links`),
			},
			{
				Config:      acctest.Provider() + testAccClusterLink_attrs(linkName, `type = "public"`, `remote_address = "127.0.0.1:8443"`, `trust_token = "token"`),
				ExpectError: testAccClusterLink_errorRegexp(`Trust token cannot be set for public cluster links`),
			},
			{
				Config:      acctest.Provider() + testAccClusterLink_attrs(linkName, `type = "public"`),
				ExpectError: testAccClusterLink_errorRegexp(`Public cluster links require the remote address`),
			},
			{
				Config:      acctest.Provider() + testAccClusterLink_attrs(linkName, `type = "public"`, `remote_address = "127.0.0.1:8443"`),
				ExpectError: testAccClusterLink_errorRegexp(`Public cluster links require the fingerprint`),
			},
		},
	})
}

func TestAccClusterLink_public(t *testing.T) {
	// A server can act as both ends of a public link, because no identity is involved.
	// The expected fingerprint is read from the same address that LXD is told to contact.
	acctest.PreCheckAPIExtensions(t, "cluster_links_public")
	serverURL := acctest.PreCheckLocalServerHTTPS(t)
	address := strings.TrimPrefix(serverURL, "https://")
	fingerprint := acctest.GetServerCertificateFingerprint(t, serverURL)

	linkName := acctest.GenerateName(2, "-")
	resourceName := "lxd_cluster_link.link"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      acctest.Provider() + testAccClusterLink_public(linkName, address, "0000"),
				ExpectError: testAccClusterLink_errorRegexp(`does not match fingerprint`),
			},
			{
				// A pending public link of the same name that Terraform does not manage must not be
				// refreshed and then removed by the provider.
				PreConfig: func() {
					leftover := api.ClusterLinksPost{
						Name:          linkName,
						Type:          api.ClusterLinkTypePublic,
						RemoteAddress: address,
					}

					_, err := acctest.InstanceServer(t).CreateClusterLinkPendingPublic(leftover)
					if err != nil {
						t.Fatalf("Failed to create leftover pending cluster link: %v", err)
					}
				},
				Config:      acctest.Provider() + testAccClusterLink_public(linkName, address, fingerprint),
				ExpectError: testAccClusterLink_errorRegexp(fmt.Sprintf(`Cluster link "%s" already exists`, linkName)),
			},
			{
				PreConfig: func() {
					err := acctest.InstanceServer(t).DeleteClusterLink(linkName)
					if err != nil && !errors.IsNotFoundError(err) {
						t.Fatalf("Failed to remove leftover pending cluster link: %v", err)
					}
				},
				Config: acctest.Provider() + testAccClusterLink_public(linkName, address, fingerprint),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", linkName),
					resource.TestCheckResourceAttr(resourceName, "type", "public"),
					resource.TestCheckResourceAttr(resourceName, "remote_address", address),
					resource.TestCheckResourceAttr(resourceName, "fingerprint", fingerprint),
					resource.TestCheckResourceAttr(resourceName, "auth_groups.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "addresses.#", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "trust_token"),
				),
			},
			{
				// The address and fingerprint are known only at creation and cannot be read back.
				ResourceName:                         resourceName,
				ImportStateId:                        linkName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"remote_address", "fingerprint"},
			},
		},
	})
}

// testAccClusterLink_errorRegexp returns a pattern that matches the given error text even where
// Terraform wraps it over several lines.
func testAccClusterLink_errorRegexp(text string) *regexp.Regexp {
	return regexp.MustCompile(strings.ReplaceAll(regexp.QuoteMeta(text), " ", `\s+`))
}

// testAccClusterLink_checkTrustToken verifies that the trust token in state decodes into a cluster
// link token issued for the given link.
func testAccClusterLink_checkTrustToken(resourceName string, linkName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		res, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource %q not found", resourceName)
		}

		token, err := shared.CertificateTokenDecode(res.Primary.Attributes["trust_token"])
		if err != nil {
			return fmt.Errorf("Failed to decode trust token: %w", err)
		}

		if token.Type != api.IdentityTypeCertificateClusterLink {
			return fmt.Errorf("Expected trust token of type %q, got %q", api.IdentityTypeCertificateClusterLink, token.Type)
		}

		if token.ClientName != linkName {
			return fmt.Errorf("Expected trust token for %q, got %q", linkName, token.ClientName)
		}

		return nil
	}
}

// testAccClusterLink_captureAttr stores the value of the given resource attribute in target, so
// that it can be used by a subsequent test step.
func testAccClusterLink_captureAttr(resourceName string, attrName string, target *string) resource.TestCheckFunc {
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

func testAccClusterLink_pending(name string, description string, authGroups []string, env string) string {
	return fmt.Sprintf(`
resource "lxd_cluster_link" "link" {
  name        = %q
  description = %q
  auth_groups = [%s]
  config = {
    "user.env" = %q
  }
}
	`, name, description, acctest.QuoteStrings(authGroups), env)
}

func testAccClusterLink_public(name string, address string, fingerprint string) string {
	fingerprintAttr := ""
	if fingerprint != "" {
		fingerprintAttr = fmt.Sprintf("fingerprint = %q", fingerprint)
	}

	return fmt.Sprintf(`
resource "lxd_cluster_link" "link" {
  name           = %q
  type           = "public"
  remote_address = %q
  %s
}
	`, name, address, fingerprintAttr)
}

// testAccClusterLink_attrs returns a cluster link resource with the given attribute lines.
func testAccClusterLink_attrs(name string, attrs ...string) string {
	return fmt.Sprintf(`
resource "lxd_cluster_link" "link" {
  name = %q
  %s
}
	`, name, strings.Join(attrs, "\n  "))
}
