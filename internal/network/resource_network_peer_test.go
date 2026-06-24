package network_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccNetworkPeer_basic(t *testing.T) {
	srcNetwork := acctest.GenerateName(2, "-")
	dstNetwork := acctest.GenerateName(2, "-")
	subnet := acctest.GenerateSubnet()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetworkPeer_basic(srcNetwork, dstNetwork, subnet),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network_1", "name", srcNetwork),
					resource.TestCheckResourceAttr("lxd_network.network_2", "name", dstNetwork),
					resource.TestCheckResourceAttr("lxd_network_peer.peer-1-2", "name", fmt.Sprintf("peer-%s", dstNetwork)),
					resource.TestCheckResourceAttr("lxd_network_peer.peer-1-2", "description", "Network peer"),
					resource.TestCheckResourceAttr("lxd_network_peer.peer-1-2", "source_project", "default"),
					resource.TestCheckResourceAttr("lxd_network_peer.peer-1-2", "source_network", srcNetwork),
					resource.TestCheckResourceAttr("lxd_network_peer.peer-1-2", "target_project", "default"),
					resource.TestCheckResourceAttr("lxd_network_peer.peer-1-2", "target_network", dstNetwork),
					resource.TestCheckResourceAttr("lxd_network_peer.peer-2-1", "name", fmt.Sprintf("peer-%s", srcNetwork)),
					resource.TestCheckResourceAttr("lxd_network_peer.peer-2-1", "description", "Network peer"),
					resource.TestCheckResourceAttr("lxd_network_peer.peer-2-1", "source_project", "default"),
					resource.TestCheckResourceAttr("lxd_network_peer.peer-2-1", "source_network", dstNetwork),
					resource.TestCheckResourceAttr("lxd_network_peer.peer-2-1", "target_project", "default"),
					resource.TestCheckResourceAttr("lxd_network_peer.peer-2-1", "target_network", srcNetwork),
				),
			},
		},
	})
}

func TestAccNetworkPeer_import(t *testing.T) {
	resourceName := "lxd_network_peer.peer-1-2"
	srcNetwork := acctest.GenerateName(2, "-")
	dstNetwork := acctest.GenerateName(2, "-")
	subnet := acctest.GenerateSubnet()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetworkPeer_basic(srcNetwork, dstNetwork, subnet),
			},
			{
				ResourceName:  resourceName,
				ImportStateId: fmt.Sprintf("/peer-%s/default/%s/default/%s", dstNetwork, srcNetwork, dstNetwork),
				ImportState:   true,
				// FIXME: When creating mutual network peers, the first that
				// is applied may report "pending" state because it gets stored
				// in the terraform state and is not refreshed once the other
				// peer is configured.
				ImportStateVerify:                    false,
				ImportStateVerifyIdentifierAttribute: "source_network",
			},
		},
	})
}

func testAccNetworkPeer_basic(srcNetwork string, dstNetwork string, subnet acctest.Subnet) string {
	peerRes := fmt.Sprintf(`
resource "lxd_network" "network_1" {
  name = "%s"
  type = "ovn"
  config = {
    "network" = lxd_network.ovnbr.name
  }
}

resource "lxd_network" "network_2" {
  name = "%s"
  type = "ovn"
  config = {
    "network" = lxd_network.ovnbr.name
  }
}

resource "lxd_network_peer" "peer-1-2" {
  name           = "peer-${lxd_network.network_2.name}"
  description    = "Network peer"
  source_network = lxd_network.network_1.name
  target_network = lxd_network.network_2.name
}

resource "lxd_network_peer" "peer-2-1" {
  name           = "peer-${lxd_network.network_1.name}"
  description    = "Network peer"
  source_network = lxd_network.network_2.name
  target_network = lxd_network.network_1.name
}
`, srcNetwork, dstNetwork)

	return fmt.Sprintf("%s\n%s", ovnUplinkNetworkResource(subnet), peerRes)
}

// ovnUplinkNetworkResource returns configuration for an OVN uplink bridge network.
// Addressing (routes/DHCP/OVN ranges) is derived from the provided subnet.
func ovnUplinkNetworkResource(subnet acctest.Subnet) string {
	return fmt.Sprintf(`
resource "lxd_network" "ovnbr" {
  name = "ovn_uplink"
  type = "bridge"
  config = {
    "ipv4.address"     = "%s"
    "ipv4.routes"      = "%s/26"
    "ipv4.ovn.ranges"  = "%s"
    "ipv4.dhcp.ranges" = "%s"
    "ipv6.address"     = "%s"
    "ipv6.dhcp.ranges" = "%s"
    "ipv6.ovn.ranges"  = "%s"
  }
}
`,
		subnet.GatewayCIDRv4(),
		subnet.HostIPv4(192),
		subnet.SubRangeV4(193, 254),
		subnet.SubRangeV4(100, 150),
		subnet.GatewayCIDRv6(),
		subnet.SubRangeV6(0xa),
		subnet.SubRangeV6(0xb),
	)
}
