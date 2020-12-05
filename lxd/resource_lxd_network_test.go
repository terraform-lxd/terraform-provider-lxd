package lxd

import (
	"fmt"
	"strings"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"github.com/lxc/lxd/shared/api"
)

func TestAccNetwork_basic(t *testing.T) {
	var network api.Network

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
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

func TestAccNetwork_description(t *testing.T) {
	var network api.Network

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_desc(),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkExists(t, "lxd_network.eth1", &network),
					resource.TestCheckResourceAttr("lxd_network.eth1", "description", "descriptive"),
				),
			},
		},
	})
}

func TestAccNetwork_attach(t *testing.T) {
	var network api.Network
	var profile api.Profile
	var container api.Container
	profileName := strings.ToLower(petname.Generate(2, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	device := map[string]string{
		"type":    "nic",
		"nictype": "bridged",
		"parent":  "eth1",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
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

func TestAccNetwork_updateConfig(t *testing.T) {
	var network api.Network
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_updateConfig_1(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkExists(t, "lxd_network.eth1", &network),
					resource.TestCheckResourceAttr("lxd_network.eth1", "config.ipv4.address", "10.150.19.1/24"),
					resource.TestCheckResourceAttr("lxd_network.eth1", "config.ipv4.nat", "true"),
				),
			},
			{
				Config: testAccNetwork_updateConfig_2(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkExists(t, "lxd_network.eth1", &network),
					resource.TestCheckResourceAttr("lxd_network.eth1", "config.ipv4.address", "10.150.21.1/24"),
					resource.TestCheckResourceAttr("lxd_network.eth1", "config.ipv4.nat", "false"),
				),
			},
		},
	})
}

func testAccNetworkExists(t *testing.T, n string, network *api.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := testAccProvider.Meta().(*lxdProvider).GetInstanceServer("")
		if err != nil {
			return err
		}
		n, _, err := client.GetNetwork(rs.Primary.ID)
		if err != nil {
			return err
		}

		*network = *n

		return nil
	}
}

func testAccNetworkConfig(network *api.Network, k, v string) resource.TestCheckFunc {
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
	return fmt.Sprintf(`
resource "lxd_network" "eth1" {
  name = "eth1"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat" = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "true"
  }
}
`)
}

func testAccNetwork_desc() string {
	return fmt.Sprintf(`
resource "lxd_network" "eth1" {
  name        = "eth1"
  description = "descriptive"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat" = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "true"
  }
}
`)
}

func testAccNetwork_attach(profileName, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "eth1" {
  name = "eth1"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat" = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "true"
  }
}

resource "lxd_profile" "profile1" {
  name = "%s"

  device {
    name = "eth1"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent = "${lxd_network.eth1.name}"
    }
  }
}

resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.12"
  profiles = ["default", "${lxd_profile.profile1.name}"]
}
`, profileName, containerName)
}

func testAccNetwork_updateConfig_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_network" "eth1" {
  name = "eth1"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat" = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "true"
  }
}

# We do need a container here to ensure the network cannot
# be deleted, but must be updated in-place.
resource "lxd_container" "c1" {
  name             = "%s"
  image            = "images:alpine/3.12"
  wait_for_network = false

  device {
    name = "eth0"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent  = lxd_network.eth1.name
    }
  }
}
  `, name)
}

func testAccNetwork_updateConfig_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_network" "eth1" {
  name = "eth1"

  config = {
    "ipv4.address" = "10.150.21.1/24"
    "ipv4.nat" = "false"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "true"
  }
}

# We do need a container here to ensure the network cannot
# be deleted, but must be updated in-place.
resource "lxd_container" "c1" {
  name             = "%s"
  image            = "images:alpine/3.12"
  wait_for_network = false

  device {
    name = "eth0"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent  = lxd_network.eth1.name
    }
  }
}
  `, name)
}
