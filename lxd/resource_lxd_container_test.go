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

func TestAccContainer_basic(t *testing.T) {
	var container shared.ContainerInfo
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainer_basic(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
				),
			},
		},
	})
}

func TestAccContainer_config(t *testing.T) {
	var container shared.ContainerInfo
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainer_config(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "config.limits.cpu", "2"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerConfig(&container, "limits.cpu", "2"),
				),
			},
		},
	})
}

func TestAccContainer_update(t *testing.T) {
	var container shared.ContainerInfo
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainer_update_1(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "profiles.0", "default"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
				),
			},
			resource.TestStep{
				Config: testAccContainer_update_2(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "profiles.1", "docker"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
				),
			},
		},
	})
}

func testAccContainerRunning(t *testing.T, n string, container *shared.ContainerInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*LxdProvider).Client
		ct, err := client.ContainerInfo(rs.Primary.ID)
		if err != nil {
			return err
		}

		if ct != nil {
			*container = *ct
			return nil
		}

		return fmt.Errorf("Container not found: %s", rs.Primary.ID)
	}
}

func testAccContainerConfig(container *shared.ContainerInfo, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if container.Config == nil {
			return fmt.Errorf("No config")
		}

		for key, value := range container.Config {
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

func testAccContainer_basic(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
  name = "%s"
  image = "ubuntu"
  profiles = ["default"]
}`, name)
}

func testAccContainer_config(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
  name = "%s"
  image = "ubuntu"
  profiles = ["default"]
  config {
    limits.cpu = 2
  }
}`, name)
}

func testAccContainer_update_1(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default"]
}`, name)
}

func testAccContainer_update_2(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default", "docker"]
}`, name)
}
