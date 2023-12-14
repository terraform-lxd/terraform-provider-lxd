package incus

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/lxc/incus/shared/api"
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
					testAccNetworkExists(t, "incus_network.eth1", &network),
					testAccNetworkConfig(&network, "ipv4.address", "10.150.19.1/24"),
					resource.TestCheckResourceAttr("incus_network.eth1", "name", "eth1"),
					resource.TestCheckResourceAttr("incus_network.eth1", "type", "bridge"),
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
					testAccNetworkExists(t, "incus_network.eth1", &network),
					resource.TestCheckResourceAttr("incus_network.eth1", "description", "descriptive"),
				),
			},
		},
	})
}

func TestAccNetwork_attach(t *testing.T) {
	var network api.Network
	var profile api.Profile
	var instance api.Instance
	profileName := petname.Generate(2, "-")
	instanceName := petname.Generate(2, "-")

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
				Config: testAccNetwork_attach(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkExists(t, "incus_network.eth1", &network),
					testAccProfileRunning(t, "incus_profile.profile1", &profile),
					testAccInstanceRunning(t, "incus_instance.instance1", &instance),
					resource.TestCheckResourceAttr("incus_network.eth1", "name", "eth1"),
					testAccProfileDevice(&profile, "eth1", device),
					testAccInstanceExpandedDevice(&instance, "eth1", device),
				),
			},
		},
	})
}

func TestAccNetwork_updateConfig(t *testing.T) {
	var network api.Network
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_updateConfig_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkExists(t, "incus_network.eth1", &network),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.ipv4.address", "10.150.19.1/24"),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.ipv4.nat", "true"),
				),
			},
			{
				Config: testAccNetwork_updateConfig_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkExists(t, "incus_network.eth1", &network),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.ipv4.address", "10.150.21.1/24"),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.ipv4.nat", "false"),
				),
			},
		},
	})
}

func TestAccNetwork_typeMacvlan(t *testing.T) {
	var network api.Network

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_typeMacvlan(),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkExists(t, "incus_network.eth1", &network),
					resource.TestCheckResourceAttr("incus_network.eth1", "type", "macvlan"),
				),
			},
		},
	})
}

func TestAccNetwork_target(t *testing.T) {
	var network api.Network

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckClustering(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_target(),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkExists(t, "incus_network.cluster_network", &network),
					testAccNetworkConfig(&network, "ipv4.address", "10.150.19.1/24"),
					resource.TestCheckResourceAttr("incus_network.cluster_network_node1", "name", "cluster_network"),
					resource.TestCheckResourceAttr("incus_network.cluster_network_node2", "name", "cluster_network"),
					resource.TestCheckResourceAttr("incus_network.cluster_network", "name", "cluster_network"),
					resource.TestCheckResourceAttr("incus_network.cluster_network", "type", "bridge"),
				),
			},
		},
	})
}

func TestAccNetwork_project(t *testing.T) {
	var network api.Network
	var project api.Project
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "incus_project.project1", &project),
					testAccNetworkExistsInProject(t, "incus_network.eth1", &network, projectName),
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

		client, err := testAccProvider.Meta().(*incusProvider).GetInstanceServer("")
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

func testAccNetworkExistsInProject(t *testing.T, n string, network *api.Network, project string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := testAccProvider.Meta().(*incusProvider).GetInstanceServer("")
		if err != nil {
			return err
		}
		client = client.UseProject(project)
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
	return `
resource "incus_network" "eth1" {
  name = "eth1"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat" = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "true"
  }
}
`
}

func testAccNetwork_desc() string {
	return `
resource "incus_network" "eth1" {
  name        = "eth1"
  description = "descriptive"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat" = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "true"
  }
}
`
}

func testAccNetwork_attach(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_network" "eth1" {
  name = "eth1"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat" = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "true"
  }
}

resource "incus_profile" "profile1" {
  name = "%s"

  device {
    name = "eth1"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent = "${incus_network.eth1.name}"
    }
  }
}

resource "incus_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
  profiles = ["default", "${incus_profile.profile1.name}"]
}
`, profileName, instanceName)
}

func testAccNetwork_updateConfig_1(name string) string {
	return fmt.Sprintf(`
resource "incus_network" "eth1" {
  name = "eth1"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat" = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "true"
  }
}

# We do need a instance here to ensure the network cannot
# be deleted, but must be updated in-place.
resource "incus_instance" "c1" {
  name             = "%s"
  image            = "images:alpine/3.18"
  wait_for_network = false

  device {
    name = "eth0"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent  = incus_network.eth1.name
    }
  }
}
  `, name)
}

func testAccNetwork_updateConfig_2(name string) string {
	return fmt.Sprintf(`
resource "incus_network" "eth1" {
  name = "eth1"

  config = {
    "ipv4.address" = "10.150.21.1/24"
    "ipv4.nat" = "false"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "true"
  }
}

# We do need a instance here to ensure the network cannot
# be deleted, but must be updated in-place.
resource "incus_instance" "c1" {
  name             = "%s"
  image            = "images:alpine/3.18"
  wait_for_network = false

  device {
    name = "eth0"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent  = incus_network.eth1.name
    }
  }
}
  `, name)
}

func testAccNetwork_typeMacvlan() string {
	return `
resource "incus_network" "eth1" {
  name = "eth1"
  type = "macvlan"

  config = {
    "parent" = "incusbr0"
  }
}
`
}

func testAccNetwork_target() string {
	return `
resource "incus_network" "cluster_network_node1" {
  name = "cluster_network"
  target = "node1"

  config = {
    "bridge.external_interfaces" = "nosuchint"
  }
}

resource "incus_network" "cluster_network_node2" {
  name = "cluster_network"
  target = "node2"

  config = {
    "bridge.external_interfaces" = "nosuchint"
  }
}

resource "incus_network" "cluster_network" {
  depends_on = [
    "incus_network.cluster_network_node1",
    "incus_network.cluster_network_node2",
  ]

  name = incus_network.cluster_network_node1.name

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat" = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "true"
  }
}
`
}

func testAccNetwork_project(project string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
  config = {
	"features.storage.volumes" = false
	"features.images" = false
	"features.profiles" = false
	"features.storage.buckets" = false
  }
}
resource "incus_network" "eth1" {
  name = "eth1"
  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat" = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "true"
  }
  project = incus_project.project1.name
}
	`, project)
}
