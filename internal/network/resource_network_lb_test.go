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
	uplinkSubnet := acctest.GenerateSubnet()
	ovnSubnet := acctest.GenerateSubnet()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "network_load_balancer")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetworkLB_basic(uplinkSubnet, ovnSubnet),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("lxd_network.ovnbr", "type", "bridge"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "type", "ovn"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "config.ipv4.address", ovnSubnet.GatewayCIDRv4()),
					resource.TestCheckResourceAttr("lxd_network.ovn", "config.ipv6.address", ovnSubnet.GatewayCIDRv6()),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "description", "Load Balancer"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "listen_address", uplinkSubnet.HostIPv4(200)),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "config.%", "0"),
				),
			},
		},
	})
}

func TestAccNetworkLB_withConfig(t *testing.T) {
	uplinkSubnet := acctest.GenerateSubnet()
	ovnSubnet := acctest.GenerateSubnet()

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
				Config: acctest.Provider() + testAccNetworkLB_withConfig(lbConfig, uplinkSubnet, ovnSubnet),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "listen_address", uplinkSubnet.HostIPv4(200)),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "config.%", "1"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "config.user.test", "abcd"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.#", "0"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.#", "0"),
				),
			},
		},
	})
}

func TestAccNetworkLB_withBackendOnly(t *testing.T) {
	uplinkSubnet := acctest.GenerateSubnet()
	ovnSubnet := acctest.GenerateSubnet()
	backendName := acctest.GenerateName(2, "-")

	backend1 := api.NetworkLoadBalancerBackend{
		Name:          backendName,
		TargetAddress: ovnSubnet.HostIPv4(2),
		TargetPort:    "80",
	}

	backend2 := api.NetworkLoadBalancerBackend{
		Name:          backendName,
		TargetAddress: ovnSubnet.HostIPv4(2),
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
			acctest.PreCheckAPIExtensions(t, "network_load_balancer")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetworkLB_withBackend(backend1, uplinkSubnet, ovnSubnet),
				Check: resource.ComposeTestCheckFunc(
					acctest.PrintResourceState(t, "lxd_network_lb.test"),
					resource.TestCheckResourceAttr("lxd_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.#", "1"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.name", backend1.Name),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.target_address", backend1.TargetAddress),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.target_port", backend1.TargetPort),
				),
			},
			{
				Config: acctest.Provider() + testAccNetworkLB_withBackend(backend2, uplinkSubnet, ovnSubnet),
				Check: resource.ComposeTestCheckFunc(
					acctest.PrintResourceState(t, "lxd_network_lb.test"),
					resource.TestCheckResourceAttr("lxd_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.#", "1"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.name", backend2.Name),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.target_address", backend2.TargetAddress),
					resource.TestCheckNoResourceAttr("lxd_network_lb.test", "backend.0.target_port"),
				),
			},
		},
	})
}

func TestAccNetworkLB_withBackendAndPort(t *testing.T) {
	uplinkSubnet := acctest.GenerateSubnet()
	ovnSubnet := acctest.GenerateSubnet()
	instanceName := acctest.GenerateName(2, "")

	backend := api.NetworkLoadBalancerBackend{
		Name:          "backend",
		Description:   "Backend",
		TargetAddress: ovnSubnet.HostIPv4(2),
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
				Config: acctest.Provider() + testAccNetworkLB_withBackendAndPort(instanceName, backend, port, uplinkSubnet, ovnSubnet),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "listen_address", uplinkSubnet.HostIPv4(200)),
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
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.0.target_backend.0", backend.Name),
					resource.TestCheckResourceAttr("lxd_instance.instance", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance", "ipv4_address", backend.TargetAddress),
				),
			},
		},
	})
}

func TestAccNetworkLB_withBackendAndPort_noDescriptions(t *testing.T) {
	uplinkSubnet := acctest.GenerateSubnet()
	ovnSubnet := acctest.GenerateSubnet()

	backend := api.NetworkLoadBalancerBackend{
		Name:          "backend",
		TargetAddress: ovnSubnet.HostIPv4(2),
		TargetPort:    "80",
	}

	port := api.NetworkLoadBalancerPort{
		Protocol:   "tcp",
		ListenPort: "8080",
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "network_load_balancer")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetworkLB_withBackendAndPort_noDescription(backend, port, uplinkSubnet, ovnSubnet),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.#", "1"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.description", ""),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.#", "1"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.0.description", ""),
				),
			},
		},
	})
}

func TestAccNetworkLB_withBackendAndPort_noTargetPort(t *testing.T) {
	uplinkSubnet := acctest.GenerateSubnet()
	ovnSubnet := acctest.GenerateSubnet()

	backend := api.NetworkLoadBalancerBackend{
		Name:          "backend",
		TargetAddress: ovnSubnet.HostIPv4(2),
	}

	port := api.NetworkLoadBalancerPort{
		Protocol:   "tcp",
		ListenPort: "8080",
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
			acctest.PreCheckAPIExtensions(t, "network_load_balancer")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetworkLB_withBackendAndPort_noTargetPort(backend, port, uplinkSubnet, ovnSubnet),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("lxd_network.ovn", "name", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "network", "ovn"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.#", "1"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.name", "backend"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.description", ""),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "backend.0.target_address", backend.TargetAddress),
					resource.TestCheckNoResourceAttr("lxd_network_lb.test", "backend.0.target_port"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.#", "1"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.0.description", ""),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.0.listen_port", "8080"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.0.target_backend.0", "backend"),
					resource.TestCheckResourceAttr("lxd_network_lb.test", "port.0.protocol", "tcp"),
				),
			},
		},
	})
}

func testAccNetworkLB_basic(uplinkSubnet acctest.Subnet, ovnSubnet acctest.Subnet) string {
	lbRes := fmt.Sprintf(`
resource "lxd_network_lb" "test" {
  network        = lxd_network.ovn.name
  listen_address = "%s"
  description    = "Load Balancer"
}
`, uplinkSubnet.HostIPv4(200))

	return fmt.Sprintf("%s\n%s", ovnNetworkResource(uplinkSubnet, ovnSubnet), lbRes)
}

func testAccNetworkLB_withConfig(config map[string]string, uplinkSubnet acctest.Subnet, ovnSubnet acctest.Subnet) string {
	entries := strings.Builder{}
	for k, v := range config {
		entry := fmt.Sprintf("%q = %q\n", k, v)
		entries.WriteString(entry)
	}

	lbRes := fmt.Sprintf(`
resource "lxd_network_lb" "test" {
  network        = lxd_network.ovn.name
  listen_address = "%s"
  description    = "Load Balancer with Config"

  config = {
    %s
  }
}
`, uplinkSubnet.HostIPv4(200), entries.String())

	return fmt.Sprintf("%s\n%s", ovnNetworkResource(uplinkSubnet, ovnSubnet), lbRes)
}

func testAccNetworkLB_withBackend(backend api.NetworkLoadBalancerBackend, uplinkSubnet acctest.Subnet, ovnSubnet acctest.Subnet) string {
	targetPort := ""
	if backend.TargetPort != "" {
		targetPort = fmt.Sprintf("target_port = %q", backend.TargetPort)
	}

	args := []any{
		uplinkSubnet.HostIPv4(200), // 1
		backend.Name,               // 2
		backend.TargetAddress,      // 3
		targetPort,                 // 4
	}

	lbRes := fmt.Sprintf(`
resource "lxd_network_lb" "test" {
  network        = lxd_network.ovn.name
  listen_address = "%[1]s"

  backend {
    name           = "%[2]s"
    target_address = "%[3]s"
    %[4]s
  }
}
`, args...)

	return fmt.Sprintf("%s\n%s", ovnNetworkResource(uplinkSubnet, ovnSubnet), lbRes)
}

func testAccNetworkLB_withBackendAndPort(instanceName string, backend api.NetworkLoadBalancerBackend, port api.NetworkLoadBalancerPort, uplinkSubnet acctest.Subnet, ovnSubnet acctest.Subnet) string {
	args := []any{
		instanceName,               // 1
		acctest.TestImage,          // 2
		backend.Name,               // 3
		backend.Description,        // 4
		backend.TargetAddress,      // 5
		backend.TargetPort,         // 6
		port.Description,           // 7
		port.Protocol,              // 8
		port.ListenPort,            // 9
		uplinkSubnet.HostIPv4(200), // 10
	}

	lbRes := fmt.Sprintf(`
resource "lxd_instance" "instance" {
  name      = "%[1]s"
  image     = "%[2]s"
  ephemeral = false

  wait_for {
    type = "ipv4"
  }

  device {
    name = "eth0"
    type = "nic"
    properties = {
      "network"      = lxd_network.ovn.name
      "ipv4.address" = "%[5]s"
    }
  }
}

resource "lxd_network_lb" "test" {
  network        = lxd_network.ovn.name
  listen_address = "%[10]s"
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

	return fmt.Sprintf("%s\n%s", ovnNetworkResource(uplinkSubnet, ovnSubnet), lbRes)
}

func testAccNetworkLB_withBackendAndPort_noDescription(backend api.NetworkLoadBalancerBackend, port api.NetworkLoadBalancerPort, uplinkSubnet acctest.Subnet, ovnSubnet acctest.Subnet) string {
	args := []any{
		backend.Name,               // 1
		backend.TargetAddress,      // 2
		backend.TargetPort,         // 3
		port.Protocol,              // 4
		port.ListenPort,            // 5
		uplinkSubnet.HostIPv4(200), // 6
	}

	lbRes := fmt.Sprintf(`
resource "lxd_network_lb" "test" {
  network        = lxd_network.ovn.name
  listen_address = "%[6]s"
  description    = "Load Balancer with Backend and Port"

  backend {
    name           = "%[1]s"
    target_address = "%[2]s"
    target_port    = "%[3]s"
  }

  port {
    protocol       = "%[4]s"
    listen_port    = "%[5]s"
    target_backend = ["%[1]s"]
  }
}
`, args...)

	return fmt.Sprintf("%s\n%s", ovnNetworkResource(uplinkSubnet, ovnSubnet), lbRes)
}

func testAccNetworkLB_withBackendAndPort_noTargetPort(backend api.NetworkLoadBalancerBackend, port api.NetworkLoadBalancerPort, uplinkSubnet acctest.Subnet, ovnSubnet acctest.Subnet) string {
	args := []any{
		backend.Name,               // 1
		backend.TargetAddress,      // 2
		port.ListenPort,            // 3
		uplinkSubnet.HostIPv4(200), // 4
	}

	lbRes := fmt.Sprintf(`
resource "lxd_network_lb" "test" {
  network        = lxd_network.ovn.name
  listen_address = "%[4]s"

  backend {
    name           = "%[1]s"
    target_address = "%[2]s"
  }

  port {
    listen_port    = "%[3]s"
    target_backend = ["%[1]s"]
  }
}
`, args...)

	return fmt.Sprintf("%s\n%s", ovnNetworkResource(uplinkSubnet, ovnSubnet), lbRes)
}

// ovnNetworkPreset returns configuration for OVN network and its parent bridge.
func ovnNetworkResource(uplinkSubnet acctest.Subnet, ovnSubnet acctest.Subnet) string {
	return fmt.Sprintf(`
resource "lxd_network" "ovnbr" {
  name = "ovnbr"
  type = "bridge"
  config = {
    "ipv4.address"     = "%[1]s"
    "ipv4.routes"      = "%[2]s/26"
    "ipv4.ovn.ranges"  = "%[3]s"
    "ipv4.dhcp.ranges" = "%[4]s"
    "ipv6.address"     = "%[5]s"
    "ipv6.dhcp.ranges" = "%[6]s"
    "ipv6.ovn.ranges"  = "%[7]s"
  }
}

resource "lxd_network" "ovn" {
  name = "ovn"
  type = "ovn"
  config = {
    "network"      = lxd_network.ovnbr.name
    "ipv4.address" = "%[8]s"
    "ipv4.nat"     = "true"
    "ipv6.address" = "%[9]s"
    "ipv6.nat"     = "true"
  }
}
`,
		uplinkSubnet.GatewayCIDRv4(),
		uplinkSubnet.HostIPv4(192),
		uplinkSubnet.SubRangeV4(224, 254),
		uplinkSubnet.SubRangeV4(100, 150),
		uplinkSubnet.GatewayCIDRv6(),
		uplinkSubnet.SubRangeV6(0xa),
		uplinkSubnet.SubRangeV6(0xb),
		ovnSubnet.GatewayCIDRv4(),
		ovnSubnet.GatewayCIDRv6(),
	)
}
