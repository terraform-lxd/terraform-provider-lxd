package lxd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccNetworkLB_basic(t *testing.T) {
	var lb api.NetworkLoadBalancer

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckAPIExtensions(t, []string{"network_load_balancer"}) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkLB_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkLBExists(t, "lxd_network_lb.test", &lb),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "description", "Load Balancer"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "listen_address", "10.10.10.200"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "network", "ovn"),
				),
			},
		},
	})
}

func TestAccNetworkLB_withConfig(t *testing.T) {
	var lb api.NetworkLoadBalancer

	lbConfig := map[string]string{
		"user.test": "abcd",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckAPIExtensions(t, []string{"network_load_balancer"}) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkLB_withConfig(lbConfig),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkLBExists(t, "lxd_network_lb.test", &lb),
					testAccNetworkLBConfig(t, &lb, lbConfig),
				),
			},
		},
	})
}

func TestAccNetworkLB_withBackend(t *testing.T) {
	var lb api.NetworkLoadBalancer

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
		PreCheck:  func() { testAccPreCheckAPIExtensions(t, []string{"network_load_balancer"}) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkLB_withBackendAndPort(backend, port),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkLBExists(t, "lxd_network_lb.test", &lb),
					testAccNetworkLBBackend(t, &lb, backend),
					testAccNetworkLBPort(t, &lb, port),
				),
			},
		},
	})
}

func testAccNetworkLBExists(t *testing.T, name string, loadBalancer *api.NetworkLoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Resource %q not found", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("Resource %q has no ID", name)
		}

		split := strings.SplitN(id, "/", 2)
		if len(split) != 2 {
			return fmt.Errorf("Resource %q has invalid ID: %q", name, id)
		}

		server, err := testAccProvider.Meta().(*lxdProvider).GetInstanceServer("")
		if err != nil {
			return err
		}

		network, listenAddr := split[0], split[1]
		lb, _, err := server.GetNetworkLoadBalancer(network, listenAddr)
		if err != nil {
			return err
		}

		*loadBalancer = *lb

		return nil
	}
}

func testAccNetworkLBConfig(t *testing.T, lb *api.NetworkLoadBalancer, config map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(lb.Config) == 0 {
			return fmt.Errorf("Load balancer's config is empty")
		}

		for k, v := range config {
			val, ok := lb.Config[k]
			if !ok {
				return fmt.Errorf("Load balancer's config has no entry for key %q", k)
			}

			if val != v {
				return fmt.Errorf("Load balancer's config values mismatch for key %q: %q != %q", k, val, v)
			}
		}

		return nil
	}
}

func testAccNetworkLBBackend(t *testing.T, lb *api.NetworkLoadBalancer, backend api.NetworkLoadBalancerBackend) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(lb.Backends) == 0 {
			return fmt.Errorf("Load balancer has no backends")
		}

		for _, b := range lb.Backends {
			if reflect.DeepEqual(b, backend) {
				return nil // Found matching backend
			}
		}

		return fmt.Errorf("Load balancer has no matching backend")
	}
}

func testAccNetworkLBPort(t *testing.T, lb *api.NetworkLoadBalancer, port api.NetworkLoadBalancerPort) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(lb.Ports) == 0 {
			return fmt.Errorf("Load balancer has no ports")
		}

		for _, p := range lb.Ports {
			if reflect.DeepEqual(p, port) {
				return nil // Found matching port
			}
		}

		return fmt.Errorf("Load balancer has no matching port")
	}
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
  name      = "c1"
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
#
# Intentionally not formatted for readability.
#
resource "lxd_network" "ovnbr" {
  name = "ovnbr"
  type = "bridge"
  config = {
    "ipv4.address":     "10.10.10.1/24"
    "ipv4.routes":      "10.10.10.192/26"
    "ipv4.ovn.ranges":  "10.10.10.193-10.10.10.254"
    "ipv4.dhcp.ranges": "10.10.10.100-10.10.10.150"
    "ipv6.address":     "fd42:1000:1000:1000::1/64"
    "ipv6.dhcp.ranges": "fd42:1000:1000:1000:a::-fd42:1000:1000:1000:a::ffff"
    "ipv6.ovn.ranges":  "fd42:1000:1000:1000:b::-fd42:1000:1000:1000:b::ffff"
  }
}

resource "lxd_network" "ovn" {
  name = "ovn"
  type = "ovn"
  config = {
    "network" : lxd_network.ovnbr.name
    "bridge.mtu" : "1500"
    "ipv4.nat" : "true"
    "ipv6.nat" : "true"
    "ipv4.address" : "10.0.0.1/24"
    "ipv6.address" : "fd42::1/64"
    "volatile.network.ipv4.address" : "10.10.10.200"
    "volatile.network.ipv6.address" : "fd42:1000:1000:1000:b::1"
  }
}
`
}
