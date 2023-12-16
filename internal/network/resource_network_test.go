package network_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccNetwork_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.eth1", "name", "eth1"),
					resource.TestCheckResourceAttr("incus_network.eth1", "type", "bridge"),
					resource.TestCheckResourceAttr("incus_network.eth1", "managed", "true"),
					resource.TestCheckResourceAttr("incus_network.eth1", "description", ""),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.%", "0"),
				),
			},
		},
	})
}

func TestAccNetwork_description(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_desc(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.eth1", "name", "eth1"),
					resource.TestCheckResourceAttr("incus_network.eth1", "type", "bridge"),
					resource.TestCheckResourceAttr("incus_network.eth1", "description", "My network"),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.%", "2"),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.ipv4.address", "10.150.19.1/24"),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.ipv6.address", "fd42:474b:622d:259d::1/64"),
				),
			},
		},
	})
}

func TestAccNetwork_attach(t *testing.T) {
	profileName := petname.Generate(2, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_attach(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.eth1", "name", "eth1"),
					resource.TestCheckResourceAttr("incus_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("incus_profile.profile1", "device.#", "1"),
					resource.TestCheckResourceAttr("incus_profile.profile1", "device.0.name", "eth1"),
					resource.TestCheckResourceAttr("incus_profile.profile1", "device.0.type", "nic"),
					resource.TestCheckResourceAttr("incus_profile.profile1", "device.0.properties.parent", "eth1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.#", "2"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.0", "default"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.1", profileName),
					resource.TestCheckResourceAttrSet("incus_instance.instance1", "ipv4_address"),
					resource.TestCheckResourceAttrSet("incus_instance.instance1", "ipv6_address"),
					resource.TestCheckResourceAttrSet("incus_instance.instance1", "mac_address"),
				),
			},
		},
	})
}

func TestAccNetwork_updateConfig(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_updateConfig_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.eth1", "name", "eth1"),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.ipv4.address", "10.150.19.1/24"),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.ipv4.nat", "true"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.parent", "eth1"),
					resource.TestCheckResourceAttrSet("incus_instance.instance1", "ipv4_address"),
					resource.TestCheckResourceAttrSet("incus_instance.instance1", "mac_address"),
				),
			},
			{
				Config: testAccNetwork_updateConfig_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.eth1", "name", "eth1"),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.ipv4.address", "10.150.21.1/24"),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.ipv4.nat", "false"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.parent", "eth1"),
				),
			},
		},
	})
}

func TestAccNetwork_typeMacvlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_typeMacvlan(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.eth1", "name", "eth1"),
					resource.TestCheckResourceAttr("incus_network.eth1", "type", "macvlan"),
					resource.TestCheckResourceAttr("incus_network.eth1", "config.parent", "incusbr0"),
				),
			},
		},
	})
}

func TestAccNetwork_target(t *testing.T) {
	networkName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_target(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.cluster_network_node1", "name", networkName),
					resource.TestCheckResourceAttr("incus_network.cluster_network_node1", "target", "node-1"),
					resource.TestCheckResourceAttr("incus_network.cluster_network_node1", "config.bridge.external_interfaces", "nosuchint"),
					resource.TestCheckResourceAttr("incus_network.cluster_network_node2", "name", networkName),
					resource.TestCheckResourceAttr("incus_network.cluster_network_node2", "target", "node-2"),
					resource.TestCheckResourceAttr("incus_network.cluster_network_node2", "config.bridge.external_interfaces", "nosuchint"),
					resource.TestCheckResourceAttr("incus_network.cluster_network", "name", networkName),
					resource.TestCheckResourceAttr("incus_network.cluster_network", "type", "bridge"),
					resource.TestCheckResourceAttr("incus_network.cluster_network", "config.ipv4.address", "10.150.19.1/24"),
				),
			},
		},
	})
}

func TestAccNetwork_project(t *testing.T) {
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.eth1", "name", "eth1"),
					resource.TestCheckResourceAttr("incus_network.eth1", "project", projectName),
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
				),
			},
		},
	})
}

func TestAccNetwork_importBasic(t *testing.T) {
	resourceName := "incus_network.eth1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_basic(),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        "eth1",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccNetwork_importDesc(t *testing.T) {
	resourceName := "incus_network.eth1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_desc(),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        "eth1",
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    false, // State of "config" will be always empty.
				ImportState:                          true,
			},
		},
	})
}

func TestAccNetwork_importProject(t *testing.T) {
	resourceName := "incus_network.eth1"
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_project(projectName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("%s/eth1", projectName),
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func testAccNetwork_basic() string {
	return `
resource "incus_network" "eth1" {
  name = "eth1"
}
`
}

func testAccNetwork_desc() string {
	return `
resource "incus_network" "eth1" {
  name        = "eth1"
  description = "My network"
  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
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
  }
}

resource "incus_profile" "profile1" {
  name = "%s"

  device {
    name = "eth1"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent  = incus_network.eth1.name
    }
  }
}

resource "incus_instance" "instance1" {
  name     = "%s"
  image    = "%s"
  profiles = ["default", incus_profile.profile1.name]
}
`, profileName, instanceName, acctest.TestImage)
}

func testAccNetwork_updateConfig_1(instanceName string) string {
	return fmt.Sprintf(`
resource "incus_network" "eth1" {
  name = "eth1"
  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat"     = true
  }
}

# We do need an instance here to ensure the network cannot
# be deleted, but must be updated in-place.
resource "incus_instance" "instance1" {
  name             = "%s"
  image            = "%s"
  wait_for_network = true

  device {
    name = "eth0"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent  = incus_network.eth1.name
    }
  }
}
  `, instanceName, acctest.TestImage)
}

func testAccNetwork_updateConfig_2(instanceName string) string {
	return fmt.Sprintf(`
resource "incus_network" "eth1" {
  name = "eth1"

  config = {
    "ipv4.address" = "10.150.21.1/24"
    "ipv4.nat"     = false
  }
}

# We do need an instance here to ensure the network cannot
# be deleted, but must be updated in-place.
resource "incus_instance" "instance1" {
  name             = "%s"
  image            = "%s"
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
  `, instanceName, acctest.TestImage)
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

func testAccNetwork_target(networkName string) string {
	return fmt.Sprintf(`
resource "incus_network" "cluster_network_node1" {
  name   = "%[1]s"
  target = "node-1"

  config = {
    "bridge.external_interfaces" = "nosuchint"
  }
}

resource "incus_network" "cluster_network_node2" {
  name   = "%[1]s"
  target = "node-2"

  config = {
    "bridge.external_interfaces" = "nosuchint"
  }
}

resource "incus_network" "cluster_network" {
  depends_on = [
    incus_network.cluster_network_node1,
    incus_network.cluster_network_node2,
  ]

  name = incus_network.cluster_network_node1.name
  config = {
    "ipv4.address" = "10.150.19.1/24"
  }
}
`, networkName)
}

func testAccNetwork_project(project string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
}

resource "incus_network" "eth1" {
  name    = "eth1"
  type    = "bridge"
  project = incus_project.project1.name
}
	`, project)
}
