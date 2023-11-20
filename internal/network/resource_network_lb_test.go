package network_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccNetworkLB_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkLB_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("lxd_network.ovnbr", "type", "bridge"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "type", "ovn"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "config.ipv4.address", "10.0.0.1/24"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "config.ipv6.address", "fd42::1/64"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "description", "Load Balancer"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "listen_address", "10.10.10.200"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "config.%", "0"),
				),
			},
		},
	})
}

func TestAccNetworkLB_withConfig(t *testing.T) {
	lbConfig := map[string]string{
		"user.test": "abcd",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkLB_withConfig(lbConfig),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "listen_address", "10.10.10.200"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "config.%", "1"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "config.user.test", "abcd"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.#", "0"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.#", "0"),
				),
			},
		},
	})
}

func TestAccNetworkLB_withBackend(t *testing.T) {
	backend := api.NetworkLoadBalancerBackend{
		Name:          "backend",
		Description:   "Backend",
		TargetAddress: "10.0.0.2",
		TargetPort:    "80",
	}

	port := api.NetworkLoadBalancerPort{
		Description:   "Port 8080/tcp",
		Protocol:      "tcp",
		ListenPort:    "8080",
		TargetBackend: []string{"backend"},
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkLB_withBackendAndPort(backend, port),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "listen_address", "10.10.10.200"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.#", "1"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.name", backend.Name),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.description", backend.Description),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.target_address", backend.TargetAddress),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.target_port", backend.TargetPort),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.#", "1"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.0.description", port.Description),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.0.protocol", port.Protocol),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.0.listen_port", port.ListenPort),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.0.target_backend.#", "1"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.0.target_backend.0", port.TargetBackend[0]),
					resource.TestCheckResourceAttr("lxd_instance.instance", "name", "tfc1"),
					resource.TestCheckResourceAttr("lxd_instance.instance", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance", "ipv4_address", backend.TargetAddress),
				),
			},
		},
	})
}

func testAccNetworkLB_basic() string {
	lbRes := `
resource "lxd_network_lb" "test" {
  network        = lxd_network.ovn.name
  listen_address = "10.10.10.200"
  description    = "Load Balancer"
}
`

	return fmt.Sprintf("%s\n%s", ovnNetworkResource(), lbRes)
}

func testAccNetworkLB_withConfig(config map[string]string) string {
	entries := strings.Builder{}
	for k, v := range config {
		entry := fmt.Sprintf("%q = %q\n", k, v)
		entries.WriteString(entry)
	}

	lbRes := fmt.Sprintf(`
resource "lxd_network_lb" "test" {
  network        = lxd_network.ovn.name
  listen_address = "10.10.10.200"
  description    = "Load Balancer with Config"

  config = {
    %s
  }
}
`, entries.String())

	return fmt.Sprintf("%s\n%s", ovnNetworkResource(), lbRes)
}

func testAccNetworkLB_withBackendAndPort(backend api.NetworkLoadBalancerBackend, port api.NetworkLoadBalancerPort) string {
	portTargetBackend := make([]string, 0, len(port.TargetBackend))
	for _, b := range port.TargetBackend {
		portTargetBackend = append(portTargetBackend, fmt.Sprintf("%q", b))
	}

	args := []any{
		backend.TargetAddress,                 // 1
		backend.Name,                          // 2
		backend.Description,                   // 3
		backend.TargetAddress,                 // 4
		backend.TargetPort,                    // 5
		port.Description,                      // 6
		port.Protocol,                         // 7
		port.ListenPort,                       // 8
		strings.Join(portTargetBackend, ", "), // 9
	}

	lbRes := fmt.Sprintf(`
resource "lxd_instance" "instance" {
  name      = "tfc1"
  image     = "images:alpine/3.18"
  ephemeral = false

  device {
    name = "eth0"
    type = "nic"
    properties = {
      "network"      = lxd_network.ovn.name
      "ipv4.address" = %[1]q
    }
  }
}

resource "lxd_network_lb" "test" {
  network        = lxd_network.ovn.name
  listen_address = "10.10.10.200"
  description    = "Load Balancer with Backend and Port"

  backend {
    name           = %[2]q
    description    = %[3]q
    target_address = %[4]q
    target_port    = %[5]q
  }

  port {
    description    = %[6]q
    protocol       = %[7]q
    listen_port    = %[8]q
    target_backend = [%[9]s]
  }
}
`, args...)

	return fmt.Sprintf("%s\n%s", ovnNetworkResource(), lbRes)
}

// ovnNetworkPreset returns configuration for OVN network and its parent bridge.
// Network resource "lxd_network.ovn" provides dhcp range "10.0.0.1/24".
func ovnNetworkResource() string {
	return `
resource "lxd_network" "ovnbr" {
  name = "ovnbr"
  type = "bridge"
  config = {
    "ipv4.address"     = "10.10.10.1/24"
    "ipv4.routes"      = "10.10.10.192/26"
    "ipv4.ovn.ranges"  = "10.10.10.193-10.10.10.254"
    "ipv4.dhcp.ranges" = "10.10.10.100-10.10.10.150"
    "ipv6.address"     = "fd42:1000:1000:1000::1/64"
    "ipv6.dhcp.ranges" = "fd42:1000:1000:1000:a::-fd42:1000:1000:1000:a::ffff"
    "ipv6.ovn.ranges"  = "fd42:1000:1000:1000:b::-fd42:1000:1000:1000:b::ffff"
  }
}

resource "lxd_network" "ovn" {
  name = "ovn"
  type = "ovn"
  config = {
    "network"      = lxd_network.ovnbr.name
    "ipv4.address" = "10.0.0.1/24"
    "ipv4.nat" : "true"
    "ipv6.address" = "fd42::1/64"
    "ipv6.nat" : "true"
  }
}
`
}
