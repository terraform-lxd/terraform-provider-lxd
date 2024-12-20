package network_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccNetwork_basic(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_basic(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "type", "bridge"),
					resource.TestCheckResourceAttr("lxd_network.network", "managed", "true"),
					resource.TestCheckResourceAttr("lxd_network.network", "description", ""),
					resource.TestCheckResourceAttr("lxd_network.network", "config.%", "0"),
				),
			},
		},
	})
}

func TestAccNetwork_description(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_desc(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "type", "bridge"),
					resource.TestCheckResourceAttr("lxd_network.network", "description", "My network"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.%", "2"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv4.address", "10.150.10.1/24"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv6.address", "fd42:474b:622d:259d::1/64"),
				),
			},
		},
	})
}

func TestAccNetwork_nullable(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_nullable(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "type", "bridge"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.%", "2"),
					resource.TestCheckNoResourceAttr("lxd_network.network", "config.ipv4.address"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv6.address", "none"),
				),
			},
		},
	})
}

func TestAccNetwork_attach(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")
	profileName := acctest.GenerateName(2, "-")
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_attach(networkName, profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.name", "eth1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.type", "nic"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.parent", networkName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.#", "2"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.0", "default"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.1", profileName),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "ipv4_address"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "ipv6_address"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "mac_address"),
				),
			},
		},
	})
}

func TestAccNetwork_updateConfig(t *testing.T) {
	networkName := acctest.GenerateName(1, "-")
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_updateConfig_1(networkName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv4.address", "10.150.30.1/24"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv4.nat", "true"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.parent", networkName),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "ipv4_address"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "mac_address"),
				),
			},
			{
				Config: testAccNetwork_updateConfig_2(networkName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv4.address", "10.150.40.1/24"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv4.nat", "false"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.parent", networkName),
				),
			},
		},
	})
}

func TestAccNetwork_typeMacvlan(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_typeMacvlan(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "type", "macvlan"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.parent", "lxdbr0"),
				),
			},
		},
	})
}

func TestAccNetwork_target(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")

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
					resource.TestCheckResourceAttr("lxd_network.cluster_network_node1", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.cluster_network_node1", "target", "node-1"),
					resource.TestCheckResourceAttr("lxd_network.cluster_network_node1", "config.bridge.external_interfaces", "nosuchint"),
					resource.TestCheckResourceAttr("lxd_network.cluster_network_node2", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.cluster_network_node2", "target", "node-2"),
					resource.TestCheckResourceAttr("lxd_network.cluster_network_node2", "config.bridge.external_interfaces", "nosuchint"),
					resource.TestCheckResourceAttr("lxd_network.cluster_network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.cluster_network", "type", "bridge"),
					resource.TestCheckResourceAttr("lxd_network.cluster_network", "config.ipv4.address", "10.150.50.1/24"),
				),
			},
		},
	})
}

func TestAccNetwork_project(t *testing.T) {
	projectName := acctest.GenerateName(2, "-")
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_project(networkName, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "project", projectName),
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
				),
			},
		},
	})
}

func TestAccNetwork_importBasic(t *testing.T) {
	resourceName := "lxd_network.network"
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_basic(networkName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        networkName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccNetwork_importDesc(t *testing.T) {
	resourceName := "lxd_network.network"
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_desc(networkName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        networkName,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    false, // State of "config" will be always empty.
				ImportState:                          true,
			},
		},
	})
}

func TestAccNetwork_importProject(t *testing.T) {
	resourceName := "lxd_network.network"
	networkName := acctest.GenerateName(2, "-")
	projectName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_project(networkName, projectName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("%s/%s", projectName, networkName),
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func testAccNetwork_basic(networkName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name = "%s"
}
`, networkName)
}

func testAccNetwork_desc(networkName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name        = "%s"
  description = "My network"
  config = {
    "ipv4.address" = "10.150.10.1/24"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
  }
}
`, networkName)
}

func testAccNetwork_nullable(networkName string) string {
	return fmt.Sprintf(`
locals {
  foo = "bar"
}

resource "lxd_network" "network" {
  name = "%s"

  config = {
    "ipv4.address" = local.foo == "bar" ? null : "10.0.0.1/24"
    "ipv6.address" = "none"
  }
}
`, networkName)
}

func testAccNetwork_attach(networkName string, profileName string, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name = "%s"
  config = {
    "ipv4.address" = "10.150.20.1/24"
  }
}

resource "lxd_profile" "profile1" {
  name = "%s"

  device {
    name = "eth1"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent  = lxd_network.network.name
    }
  }
}

resource "lxd_instance" "instance1" {
  name     = "%s"
  image    = "%s"
  profiles = ["default", lxd_profile.profile1.name]
}
`, networkName, profileName, instanceName, acctest.TestImage)
}

func testAccNetwork_updateConfig_1(networkName string, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name = "%s"
  config = {
    "ipv4.address" = "10.150.30.1/24"
    "ipv4.nat"     = true
  }
}

# We do need an instance here to ensure the network cannot
# be deleted, but must be updated in-place.
resource "lxd_instance" "instance1" {
  name             = "%s"
  image            = "%s"
  wait_for_network = true

  device {
    name = "eth0"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent  = lxd_network.network.name
    }
  }
}
`, networkName, instanceName, acctest.TestImage)
}

func testAccNetwork_updateConfig_2(networkName string, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name = "%s"

  config = {
    "ipv4.address" = "10.150.40.1/24"
    "ipv4.nat"     = false
  }
}

# We do need an instance here to ensure the network cannot
# be deleted, but must be updated in-place.
resource "lxd_instance" "instance1" {
  name             = "%s"
  image            = "%s"
  wait_for_network = false

  device {
    name = "eth0"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent  = lxd_network.network.name
    }
  }
}
`, networkName, instanceName, acctest.TestImage)
}

func testAccNetwork_typeMacvlan(networkName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name = "%s"
  type = "macvlan"

  config = {
    "parent" = "lxdbr0"
  }
}
`, networkName)
}

func testAccNetwork_target(networkName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "cluster_network_node1" {
  name   = "%[1]s"
  target = "node-1"

  config = {
    "bridge.external_interfaces" = "nosuchint"
  }
}

resource "lxd_network" "cluster_network_node2" {
  name   = "%[1]s"
  target = "node-2"

  config = {
    "bridge.external_interfaces" = "nosuchint"
  }
}

resource "lxd_network" "cluster_network" {
  depends_on = [
    lxd_network.cluster_network_node1,
    lxd_network.cluster_network_node2,
  ]

  name = lxd_network.cluster_network_node1.name
  config = {
    "ipv4.address" = "10.150.50.1/24"
  }
}
`, networkName)
}

func testAccNetwork_project(networkName string, projectName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
}

resource "lxd_network" "network" {
  name    = "%s"
  type    = "bridge"
  project = lxd_project.project1.name
}
`, projectName, networkName)
}
