package truststore_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccTrustToken_content(t *testing.T) {
	tokenName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrustToken(tokenName, "default"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_trust_token.token", "name", tokenName),
					resource.TestCheckResourceAttr("lxd_trust_token.token", "projects.#", "1"),
					resource.TestCheckResourceAttr("lxd_trust_token.token", "projects.0", "default"),
					resource.TestCheckResourceAttrSet("lxd_trust_token.token", "token"),
					resource.TestCheckResourceAttrSet("lxd_trust_token.token", "operation_id"),
				),
			},
			{
				Config: testAccTrustToken(tokenName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_trust_token.token", "name", tokenName),
					resource.TestCheckResourceAttr("lxd_trust_token.token", "projects.#", "0"),
					resource.TestCheckResourceAttrSet("lxd_trust_token.token", "token"),
					resource.TestCheckResourceAttrSet("lxd_trust_token.token", "operation_id"),
				),
			},
		},
	})
}

func testAccTrustToken(name string, projects ...string) string {
	return fmt.Sprintf(`
resource "lxd_trust_token" "token" {
  name     = "%s"
  projects = [%s]
}
	`, name, acctest.QuoteStrings(projects))
}
