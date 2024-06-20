package network_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccNetworkACL_basic(t *testing.T) {
	aclName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACL(aclName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_acl.acl", "name", aclName),
					resource.TestCheckResourceAttr("incus_network_acl.acl", "description", "Network ACL"),
				),
			},
		},
	})
}

func TestAccNetworkACL_egress(t *testing.T) {
	aclName := petname.Generate(2, "-")

	entry1 := map[string]string{
		"action":           "allow",
		"destination":      "1.1.1.1,1.0.0.1",
		"destination_port": "53",
		"protocol":         "udp",
		"description":      "DNS to cloudflare public resolvers (UDP)",
		"state":            "enabled",
	}

	entry2 := map[string]string{
		"action":           "allow",
		"destination":      "1.1.1.1,1.0.0.1",
		"destination_port": "53",
		"protocol":         "tcp",
		"description":      "DNS to cloudflare public resolvers (TCP)",
		"state":            "enabled",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACL_withEgressRules(aclName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_acl.acl", "name", aclName),
					resource.TestCheckResourceAttr("incus_network_acl.acl", "description", "Network ACL"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_network_acl.acl", "egress.*", entry1),
					resource.TestCheckTypeSetElemNestedAttrs("incus_network_acl.acl", "egress.*", entry2),
				),
			},
		},
	})
}

func TestAccNetworkACL_ingress(t *testing.T) {
	aclName := petname.Generate(2, "-")

	entry := map[string]string{
		"action":           "allow",
		"source":           "@external",
		"destination_port": "22",
		"protocol":         "tcp",
		"description":      "Incoming SSH connections",
		"state":            "logged",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACL_withIngressRules(aclName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_acl.acl", "name", aclName),
					resource.TestCheckResourceAttr("incus_network_acl.acl", "description", "Network ACL"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_network_acl.acl", "ingress.*", entry),
				),
			},
		},
	})
}

func TestAccNetworkACL_egressAndIngress(t *testing.T) {
	aclName := petname.Generate(2, "-")

	ingresEntry := map[string]string{
		"action":           "allow",
		"source":           "@external",
		"destination_port": "22",
		"protocol":         "tcp",
		"description":      "Incoming SSH connections",
		"state":            "logged",
	}

	egressEntry1 := map[string]string{
		"action":           "allow",
		"destination":      "1.1.1.1,1.0.0.1",
		"destination_port": "53",
		"protocol":         "udp",
		"description":      "DNS to cloudflare public resolvers (UDP)",
		"state":            "enabled",
	}

	egressEntry2 := map[string]string{
		"action":           "allow",
		"destination":      "1.1.1.1,1.0.0.1",
		"destination_port": "53",
		"protocol":         "tcp",
		"description":      "DNS to cloudflare public resolvers (TCP)",
		"state":            "enabled",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACL_withTrafficRules(aclName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_acl.acl", "name", aclName),
					resource.TestCheckResourceAttr("incus_network_acl.acl", "description", "Network ACL"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_network_acl.acl", "egress.*", egressEntry1),
					resource.TestCheckTypeSetElemNestedAttrs("incus_network_acl.acl", "egress.*", egressEntry2),
					resource.TestCheckTypeSetElemNestedAttrs("incus_network_acl.acl", "ingress.*", ingresEntry),
				),
			},
		},
	})
}

func testAccNetworkACL(aclName string) string {
	return fmt.Sprintf(`
resource "incus_network_acl" "acl" {
  name        = "%s"
  description = "Network ACL"
}
`, aclName)
}

func testAccNetworkACL_withEgressRules(aclName string) string {
	return fmt.Sprintf(`
resource "incus_network_acl" "acl" {
  name        = "%[1]s"
  description = "Network ACL"

  egress = [
	{
	  action           = "allow"
	  destination      = "1.1.1.1,1.0.0.1"
	  destination_port = "53"
	  protocol         = "udp"
	  description      = "DNS to cloudflare public resolvers (UDP)"
	  state            = "enabled"
	},
	{
	  action           = "allow"
	  destination      = "1.1.1.1,1.0.0.1"
	  destination_port = "53"
	  protocol         = "tcp"
	  description      = "DNS to cloudflare public resolvers (TCP)"
	  state            = "enabled"
	}
  ]
}
`, aclName)
}

func testAccNetworkACL_withIngressRules(aclName string) string {
	return fmt.Sprintf(`
resource "incus_network_acl" "acl" {
  name        = "%[1]s"
  description = "Network ACL"

  ingress = [
    {
      action           = "allow"
      source           = "@external"
      destination_port = "22"
      protocol         = "tcp"
      description      = "Incoming SSH connections"
      state            = "logged"
    }
  ]
}
`, aclName)
}

func testAccNetworkACL_withTrafficRules(aclName string) string {
	return fmt.Sprintf(`
resource "incus_network_acl" "acl" {
  name        = "%[1]s"
  description = "Network ACL"

  egress = [
	{
	  action           = "allow"
	  destination      = "1.1.1.1,1.0.0.1"
	  destination_port = "53"
	  protocol         = "udp"
	  description      = "DNS to cloudflare public resolvers (UDP)"
	  state            = "enabled"
	},
	{
	  action           = "allow"
	  destination      = "1.1.1.1,1.0.0.1"
	  destination_port = "53"
	  protocol         = "tcp"
	  description      = "DNS to cloudflare public resolvers (TCP)"
	  state            = "enabled"
	}
  ]

  ingress = [
    {
      action           = "allow"
      source           = "@external"
      destination_port = "22"
      protocol         = "tcp"
      description      = "Incoming SSH connections"
      state            = "logged"
    }
  ]
}
`, aclName)
}
