package lxd

import (
	"fmt"
	"testing"

	"github.com/canonical/lxd/shared/api"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccNetworkZone_basic(t *testing.T) {
	var zone api.NetworkZone

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZone_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkZoneExists(t, "lxd_network_zone.zone", &zone),
					testAccNetworkZoneConfig(&zone, "peers.ns.address", "127.0.0.1"),
					resource.TestCheckResourceAttr(
						"lxd_network_zone.zone",
						"name",
						"custom.example.org",
					),
				),
			},
		},
	})
}

func TestAccNetworkZone_description(t *testing.T) {
	var zone api.NetworkZone

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZone_desc(),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkZoneExists(t, "lxd_network_zone.zone", &zone),
					resource.TestCheckResourceAttr(
						"lxd_network_zone.zone",
						"description",
						"descriptive",
					),
				),
			},
		},
	})
}

func TestAccNetworkZone_project(t *testing.T) {
	var zone api.NetworkZone
	var project api.Project
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZone_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "lxd_project.project1", &project),
					testAccNetworkZoneExistsInProject(
						t,
						"lxd_network_zone.zone",
						&zone,
						projectName,
					),
				),
			},
		},
	})
}

func testAccNetworkZoneExists(
	t *testing.T,
	n string,
	zone *api.NetworkZone,
) resource.TestCheckFunc {
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
		z, _, err := client.GetNetworkZone(rs.Primary.ID)
		if err != nil {
			return err
		}

		*zone = *z

		return nil
	}
}

func testAccNetworkZoneExistsInProject(
	t *testing.T,
	n string,
	zone *api.NetworkZone,
	project string,
) resource.TestCheckFunc {
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
		client = client.UseProject(project)
		z, _, err := client.GetNetworkZone(rs.Primary.ID)
		if err != nil {
			return err
		}

		*zone = *z

		return nil
	}
}

func testAccNetworkZoneConfig(zone *api.NetworkZone, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if zone.Config == nil {
			return fmt.Errorf("No config")
		}

		for key, value := range zone.Config {
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

func testAccNetworkZone_basic() string {
	return `
resource "lxd_network_zone" "zone" {
  name = "custom.example.org"

  config = {
    "dns.nameservers" = "ns.custom.example.org"
    "peers.ns.address" = "127.0.0.1"
  }
}
`
}

func testAccNetworkZone_desc() string {
	return `
resource "lxd_network_zone" "zone" {
  name = "custom.example.org"
  description = "descriptive"

  config = {
    "dns.nameservers" = "ns.custom.example.org"
    "peers.ns.address" = "127.0.0.1"
  }
}
`
}

func testAccNetworkZone_project(project string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
  config = {
	"features.storage.volumes" = false
	"features.images" = false
	"features.profiles" = false
	"features.storage.buckets" = false
	"features.networks" = false
	"features.networks.zones" = true
  }
}
resource "lxd_network_zone" "zone" {
  name = "custom.example.org"
  project = lxd_project.project1.name

  config = {
    "dns.nameservers" = "ns.custom.example.org"
    "peers.ns.address" = "127.0.0.1"
  }
}
	`, project)
}
