package network_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccNetworkForward_basic(t *testing.T) {
	networkName := acctest.GenerateName(1, "")
	subnet := acctest.GenerateSubnet()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t) // Due to standalone network creation.
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetworkForward(networkName, subnet),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network_forward.forward", "listen_address", subnet.HostIPv4(10)),
					resource.TestCheckResourceAttr("lxd_network_forward.forward", "description", "Network Forward"),
					resource.TestCheckResourceAttr("lxd_network_forward.forward", "config.target_address", subnet.HostIPv4(111)),
				),
			},
		},
	})
}

func TestAccNetworkForward_Ports(t *testing.T) {
	networkName := acctest.GenerateName(1, "")
	subnet := acctest.GenerateSubnet()

	entry1 := map[string]string{
		"description":    "SSH",
		"protocol":       "tcp",
		"listen_port":    "22",
		"target_port":    "2022",
		"target_address": subnet.HostIPv4(112),
	}

	entry2 := map[string]string{
		"description":    "HTTP",
		"protocol":       "tcp",
		"listen_port":    "80",
		"target_port":    "",
		"target_address": subnet.HostIPv4(112),
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t) // Due to standalone network creation.
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetworkForward_withPorts(networkName, subnet),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network_forward.forward", "listen_address", subnet.HostIPv4(10)),
					resource.TestCheckResourceAttr("lxd_network_forward.forward", "description", "Network Forward"),
					resource.TestCheckResourceAttr("lxd_network_forward.forward", "config.target_address", subnet.HostIPv4(111)),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_network_forward.forward", "ports.*", entry1),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_network_forward.forward", "ports.*", entry2),
				),
			},
		},
	})
}

func testAccNetworkForward(networkName string, subnet acctest.Subnet) string {
	return fmt.Sprintf(`
resource "lxd_network" "forward" {
  name = "%s"

  config = {
    "ipv4.address" = "%s"
    "ipv4.nat"     = "true"
    "ipv6.address" = "%s"
    "ipv6.nat"     = "true"
  }
}

resource "lxd_network_forward" "forward" {
  network        = lxd_network.forward.name
  description    = "Network Forward"
  listen_address = "%s"

  config = {
    target_address = "%s"
  }
}
  `, networkName, subnet.GatewayCIDRv4(), subnet.GatewayCIDRv6(), subnet.HostIPv4(10), subnet.HostIPv4(111))
}

func testAccNetworkForward_withPorts(networkName string, subnet acctest.Subnet) string {
	return fmt.Sprintf(`
resource "lxd_network" "forward" {
  name = "%s"

  config = {
    "ipv4.address" = "%s"
    "ipv4.nat"     = "true"
    "ipv6.address" = "%s"
    "ipv6.nat"     = "true"
  }
}

resource "lxd_network_forward" "forward" {
  network        = lxd_network.forward.name
  description    = "Network Forward"
  listen_address = "%s"

  config = {
    target_address = "%s"
  }

  ports = [
    {
      description    = "SSH"
      protocol       = "tcp"
      listen_port    = "22"
      target_port    = "2022"
      target_address = "%[6]s"
    },
    {
      description    = "HTTP"
      protocol       = "tcp"
      listen_port    = "80"
      target_address = "%[6]s"
    }
  ]
}
  `, networkName, subnet.GatewayCIDRv4(), subnet.GatewayCIDRv6(), subnet.HostIPv4(10), subnet.HostIPv4(111), subnet.HostIPv4(112))
}
