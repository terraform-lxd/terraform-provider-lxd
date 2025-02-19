package instance_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccInstance_DS_basic(t *testing.T) {
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_DS_basic(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Stopped"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "running", "false"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "ephemeral", "false"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "profiles.#", "1"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "profiles.0", "default"),
				),
			},
		},
	})
}

func TestAccInstance_DS_ephemeral(t *testing.T) {
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_DS_ephemeral(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Running"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "running", "true"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "ephemeral", "true"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "profiles.#", "1"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "profiles.0", "default"),
				),
			},
		},
	})
}

func TestAccInstance_DS_config(t *testing.T) {
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_DS_config(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Stopped"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "config.boot.autostart", "1"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "limits.%", "2"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "limits.cpu", "2"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "limits.memory", "128MiB"),
				),
			},
		},
	})
}

func TestAccInstance_DS_device(t *testing.T) {
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_DS_device(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Stopped"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.%", "1"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.shared.type", "disk"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.shared.properties.path", "/tmp/shared"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.shared.properties.source", "/tmp"),
				),
			},
		},
	})
}

func TestAccInstance_DS_accessInterface(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t) // Due to standalone network creation.
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create stopped instance. No address should be available.
				Config: testAccInstance_DS_accessInterface(networkName, instanceName, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Stopped"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "config.user.access_interface", "eth0"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.%", "1"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.eth0.type", "nic"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.eth0.properties.nictype", "bridged"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.eth0.properties.parent", networkName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.eth0.properties.hwaddr", "00:16:3e:39:7f:36"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.eth0.properties.ipv4.address", "10.150.19.200"),
					resource.TestCheckNoResourceAttr("data.lxd_instance.inst", "mac_address"),
					resource.TestCheckNoResourceAttr("data.lxd_instance.inst", "ipv4_address"),
					resource.TestCheckNoResourceAttr("data.lxd_instance.inst", "ipv6_address"),
				),
			},
			{
				// Start the instance to ensure the addresses get populated.
				Config: testAccInstance_DS_accessInterface(networkName, instanceName, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Running"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "mac_address", "00:16:3e:39:7f:36"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "ipv4_address", "10.150.19.200"),
					resource.TestCheckResourceAttrSet("data.lxd_instance.inst", "ipv6_address"),
				),
			},
		},
	})
}

func TestAccInstance_DS_project(t *testing.T) {
	instanceName := acctest.GenerateName(2, "-")
	projectName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_DS_project(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Stopped"),
				),
			},
		},
	})
}

func testAccInstance_DS_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "inst" {
  name    = %q
  image   = %q
  running = false
}

data "lxd_instance" "inst" {
  name = lxd_instance.inst.name
}
  `, name, acctest.TestImage)
}

func testAccInstance_DS_ephemeral(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "inst" {
  name             = %q
  image            = %q
  profiles         = ["default"]
  ephemeral        = true
  wait_for_network = false
}

data "lxd_instance" "inst" {
  name = lxd_instance.inst.name
}
  `, name, acctest.TestImage)
}

func testAccInstance_DS_config(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "inst" {
  name    = %q
  image   = %q
  running = false

  limits = {
    "cpu"    = 2
    "memory" = "128MiB"
  }

  config = {
    "boot.autostart" = 1
  }
}

data "lxd_instance" "inst" {
  name = lxd_instance.inst.name
}
  `, name, acctest.TestImage)
}

func testAccInstance_DS_device(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "inst" {
  name    = %q
  image   = %q
  running = false

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared"
    }
  }
}

data "lxd_instance" "inst" {
  name = lxd_instance.inst.name
}
  `, name, acctest.TestImage)
}

func testAccInstance_DS_accessInterface(networkName string, instanceName string, running bool) string {
	return fmt.Sprintf(`
resource "lxd_network" "net" {
  name = %q

  config = {
    "ipv4.address" = "10.150.19.1/24"
  }
}

resource "lxd_instance" "inst" {
  name    = %q
  image   = %q
  running = %v

  config = {
    "user.access_interface" = "eth0"
  }

  device {
    name = "eth0"
    type = "nic"

    properties = {
      nictype        = "bridged"
      parent         = "${lxd_network.net.name}"
      hwaddr         = "00:16:3e:39:7f:36"
      "ipv4.address" = "10.150.19.200"
    }
  }
}

data "lxd_instance" "inst" {
  name = lxd_instance.inst.name
}
  `, networkName, instanceName, acctest.TestImage, running)
}

func testAccInstance_DS_project(projectName string, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "proj" {
  name   = %q
  config = {
    "features.images"   = false
    "features.profiles" = false
  }
}

resource "lxd_instance" "inst" {
  name    = %q
  image   = %q
  running = false
  project = lxd_project.proj.name
}

data "lxd_instance" "inst" {
  name    = lxd_instance.inst.name
  project = lxd_instance.inst.project
}
  `, projectName, instanceName, acctest.TestImage)
}
