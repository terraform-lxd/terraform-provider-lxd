package network_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccNetworkACL_DS_basic(t *testing.T) {
	aclName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetworkACL_DS_basic(aclName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_network_acl.acl", "name", aclName),
					resource.TestCheckResourceAttr("data.lxd_network_acl.acl", "description", "Network ACL"),
				),
			},
		},
	})
}

func TestAccNetworkACL_DS_withRules(t *testing.T) {
	aclName := acctest.GenerateName(2, "-")

	egressEntry := map[string]string{
		"action":      "allow",
		"destination": "1.1.1.1",
		"protocol":    "udp",
		"state":       "enabled",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetworkACL_DS_withRules(aclName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_network_acl.acl", "name", aclName),
					resource.TestCheckResourceAttr("data.lxd_network_acl.acl", "egress.#", "1"),
					resource.TestCheckResourceAttr("data.lxd_network_acl.acl", "ingress.#", "0"),
					resource.TestCheckTypeSetElemNestedAttrs("data.lxd_network_acl.acl", "egress.*", egressEntry),
				),
			},
		},
	})
}

func TestAccNetworkACL_DS_withIngressRules(t *testing.T) {
	aclName := acctest.GenerateName(2, "-")

	ingressEntry := map[string]string{
		"action":   "allow",
		"source":   "@external",
		"protocol": "tcp",
		"state":    "enabled",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetworkACL_DS_withIngressRules(aclName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_network_acl.acl", "name", aclName),
					resource.TestCheckResourceAttr("data.lxd_network_acl.acl", "egress.#", "0"),
					resource.TestCheckResourceAttr("data.lxd_network_acl.acl", "ingress.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("data.lxd_network_acl.acl", "ingress.*", ingressEntry),
				),
			},
		},
	})
}

func testAccNetworkACL_DS_basic(aclName string) string {
	return fmt.Sprintf(`
resource "lxd_network_acl" "acl" {
  name        = %q
  description = "Network ACL"
}

data "lxd_network_acl" "acl" {
  name = lxd_network_acl.acl.name
}
`, aclName)
}

func testAccNetworkACL_DS_withRules(aclName string) string {
	return fmt.Sprintf(`
resource "lxd_network_acl" "acl" {
  name = %q

  egress = [
    {
      action      = "allow"
      destination = "1.1.1.1"
      protocol    = "udp"
      state       = "enabled"
    }
  ]
}

data "lxd_network_acl" "acl" {
  name = lxd_network_acl.acl.name
}
`, aclName)
}

func testAccNetworkACL_DS_withIngressRules(aclName string) string {
	return fmt.Sprintf(`
resource "lxd_network_acl" "acl" {
  name = %q

  ingress = [
    {
      action   = "allow"
      source   = "@external"
      protocol = "tcp"
      state    = "enabled"
    }
  ]
}

data "lxd_network_acl" "acl" {
  name = lxd_network_acl.acl.name
}
`, aclName)
}
