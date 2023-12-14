package network_test

import (
	"fmt"
	"strings"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/lxc/incus/shared/api"
	"github.com/maveonair/terraform-provider-incus/internal/acctest"
)

func TestAccNetworkLB_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "network_load_balancer")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkLB_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("incus_network.ovnbr", "type", "bridge"),
					resource.TestCheckResourceAttr("incus_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("incus_network.ovn", "type", "ovn"),
					resource.TestCheckResourceAttr("incus_network.ovn", "config.ipv4.address", "10.0.0.1/24"),
					resource.TestCheckResourceAttr("incus_network.ovn", "config.ipv6.address", "fd42::1/64"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "description", "Load Balancer"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "listen_address", "10.10.10.200"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "config.%", "0"),
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
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "network_load_balancer")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkLB_withConfig(lbConfig),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("incus_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "listen_address", "10.10.10.200"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "config.%", "1"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "config.user.test", "abcd"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "port.#", "0"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "backend.#", "0"),
				),
			},
		},
	})
}

func TestAccNetworkLB_withBackend(t *testing.T) {
	instanceName := petname.Generate(2, "")

	backend := api.NetworkLoadBalancerBackend{
		Name:          "backend",
		Description:   "Backend",
		TargetAddress: "10.0.0.2",
		TargetPort:    "80",
	}

	port := api.NetworkLoadBalancerPort{
		Description: "Port 8080/tcp",
		Protocol:    "tcp",
		ListenPort:  "8080",
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "network_load_balancer")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkLB_withBackendAndPort(instanceName, backend, port),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("incus_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "listen_address", "10.10.10.200"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "backend.#", "1"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "backend.0.name", backend.Name),
					resource.TestCheckResourceAttr("incus_network_lb.test", "backend.0.description", backend.Description),
					resource.TestCheckResourceAttr("incus_network_lb.test", "backend.0.target_address", backend.TargetAddress),
					resource.TestCheckResourceAttr("incus_network_lb.test", "backend.0.target_port", backend.TargetPort),
					resource.TestCheckResourceAttr("incus_network_lb.test", "port.#", "1"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "port.0.description", port.Description),
					resource.TestCheckResourceAttr("incus_network_lb.test", "port.0.protocol", port.Protocol),
					resource.TestCheckResourceAttr("incus_network_lb.test", "port.0.listen_port", port.ListenPort),
					resource.TestCheckResourceAttr("incus_network_lb.test", "port.0.target_backend.#", "1"),
					resource.TestCheckResourceAttr("incus_network_lb.test", "port.0.target_backend.0", backend.Name),
					resource.TestCheckResourceAttr("incus_instance.instance", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance", "ipv4_address", backend.TargetAddress),
				),
			},
		},
	})
}

func testAccNetworkLB_basic() string {
	lbRes := `
resource "incus_network_lb" "test" {
  network        = incus_network.ovn.name
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
resource "incus_network_lb" "test" {
  network        = incus_network.ovn.name
  listen_address = "10.10.10.200"
  description    = "Load Balancer with Config"

  config = {
    %s
  }
}
`, entries.String())

	return fmt.Sprintf("%s\n%s", ovnNetworkResource(), lbRes)
}

func testAccNetworkLB_withBackendAndPort(instanceName string, backend api.NetworkLoadBalancerBackend, port api.NetworkLoadBalancerPort) string {
	args := []any{
		instanceName,          // 1
		acctest.TestImage,     // 2
		backend.Name,          // 3
		backend.Description,   // 4
		backend.TargetAddress, // 5
		backend.TargetPort,    // 6
		port.Description,      // 7
		port.Protocol,         // 8
		port.ListenPort,       // 9
	}

	lbRes := fmt.Sprintf(`
resource "incus_instance" "instance" {
  name      = "%[1]s"
  image     = "%[2]s"
  ephemeral = false

  device {
    name = "eth0"
    type = "nic"
    properties = {
      "network"      = incus_network.ovn.name
      "ipv4.address" = "%[5]s"
    }
  }
}

resource "incus_network_lb" "test" {
  network        = incus_network.ovn.name
  listen_address = "10.10.10.200"
  description    = "Load Balancer with Backend and Port"

  backend {
    name           = "%[3]s"
    description    = "%[4]s"
    target_address = "%[5]s"
    target_port    = "%[6]s"
  }

  port {
    description    = "%[7]s"
    protocol       = "%[8]s"
    listen_port    = "%[9]s"
    target_backend = ["%[3]s"]
  }
}
`, args...)

	return fmt.Sprintf("%s\n%s", ovnNetworkResource(), lbRes)
}

// ovnNetworkPreset returns configuration for OVN network and its parent bridge.
// Network resource "incus_network.ovn" provides dhcp range "10.0.0.1/24".
func ovnNetworkResource() string {
	return `
resource "incus_network" "ovnbr" {
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

resource "incus_network" "ovn" {
  name = "ovn"
  type = "ovn"
  config = {
    "network"      = incus_network.ovnbr.name
    "ipv4.address" = "10.0.0.1/24"
    "ipv4.nat"     = "true"
    "ipv6.address" = "fd42::1/64"
    "ipv6.nat"     = "true"
  }
}
`
}
