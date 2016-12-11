package lxd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/lxc/lxd/shared"
)

func TestAccNetwork_basic(t *testing.T) {
	var network shared.NetworkConfig

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetwork_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkExists(t, "lxd_network.eth1", &network),
					testAccNetworkConfig(&network, "ipv4.address", "10.150.19.1/24"),
					resource.TestCheckResourceAttr("lxd_network.eth1", "name", "eth1"),
				),
			},
		},
	})
}

func TestAccNetwork_attach(t *testing.T) {
	var network shared.NetworkConfig
	var profile shared.ProfileConfig
	var container shared.ContainerInfo
	profileName := strings.ToLower(petname.Generate(2, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	device := shared.Device{
		"type":    "nic",
		"nictype": "bridged",
		"parent":  "eth1",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetwork_attach(profileName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkExists(t, "lxd_network.eth1", &network),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_network.eth1", "name", "eth1"),
					testAccProfileDevice(&profile, "eth1", device),
					testAccContainerExpandedDevice(&container, "eth1", device),
				),
			},
		},
	})
}

func testAccNetworkExists(t *testing.T, n string, network *shared.NetworkConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*LxdProvider).Client
		n, err := client.NetworkGet(rs.Primary.ID)
		if err != nil {
			return err
		}

		*network = n

		return nil
	}
}

func testAccNetworkConfig(network *shared.NetworkConfig, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if network.Config == nil {
			return fmt.Errorf("No config")
		}

		for key, value := range network.Config {
			if k != key {
				continue
			}

			if v == value {
				return nil
			}

			return fmt.Errorf("Bad value for %s: %s", k, value)
		}

		return fmt.Errorf("Config not found: %s", k)
	}
}

func testAccNetwork_basic() string {
	return fmt.Sprintf(`resource "lxd_network" "eth1" {
  name = "eth1"

	config {
		ipv4.address = "10.150.19.1/24"
		ipv4.nat = "true"
		ipv6.address = "fd42:474b:622d:259d::1/64"
		ipv6.nat = "true"
	}
}`)
}

func testAccNetwork_attach(profileName, containerName string) string {
	return fmt.Sprintf(`resource "lxd_network" "eth1" {
  name = "eth1"

	config {
		ipv4.address = "10.150.19.1/24"
		ipv4.nat = "true"
		ipv6.address = "fd42:474b:622d:259d::1/64"
		ipv6.nat = "true"
	}
}

resource "lxd_profile" "profile1" {
	name = "%s"

	device {
		name = "eth1"
		type = "nic"
		properties {
			nictype = "bridged"
			parent = "${lxd_network.eth1.name}"
		}
	}
}

resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default", "${lxd_profile.profile1.name}"]
}`, profileName, containerName)
}
