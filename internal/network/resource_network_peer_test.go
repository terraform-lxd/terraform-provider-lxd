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

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t) // Due to standalone network creation.
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkPeer_basic(srcNetwork, dstNetwork),
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

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t) // Due to standalone network creation.
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkPeer_basic(srcNetwork, dstNetwork),
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

func testAccNetworkPeer_basic(srcNetwork string, dstNetwork string) string {
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

	return fmt.Sprintf("%s\n%s", ovnUplinkNetworkResource(), peerRes)
}

// ovnNetworkPreset returns configuration for OVN network and its parent bridge.
// Network resource "lxd_network.ovn" provides dhcp range "10.0.0.1/24".
func ovnUplinkNetworkResource() string {
	return `
resource "lxd_network" "ovnbr" {
  name = "ovn_uplink"
  type = "bridge"
  config = {
    "ipv4.address"     = "10.11.10.1/24"
    "ipv4.routes"      = "10.11.10.192/26"
    "ipv4.ovn.ranges"  = "10.11.10.193-10.11.10.254"
    "ipv4.dhcp.ranges" = "10.11.10.100-10.11.10.150"
    "ipv6.address"     = "fd42:1100:1000:1000::1/64"
    "ipv6.dhcp.ranges" = "fd42:1100:1000:1000:a::-fd42:1100:1000:1000:a::ffff"
    "ipv6.ovn.ranges"  = "fd42:1100:1000:1000:b::-fd42:1100:1000:1000:b::ffff"
  }
}
`
}
