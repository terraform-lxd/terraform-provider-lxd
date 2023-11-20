package network_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccNetworkZone_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZone_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "name", "custom.example.org"),
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "config.%", "2"),
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "config.dns.nameservers", "ns.custom.example.org"),
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "config.peers.ns.address", "127.0.0.1"),
				),
			},
		},
	})
}

func TestAccNetworkZone_description(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZone_desc(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "name", "custom.example.org"),
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "description", "descriptive"),
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "config.%", "2"),
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "config.dns.nameservers", "ns.custom.example.org"),
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "config.peers.ns.address", "127.0.0.1"),
				),
			},
			{
				// Ensure no changes on reapply.
				Config: testAccNetworkZone_desc(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "name", "custom.example.org"),
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "description", "descriptive"),
				),
			},
		},
	})
}

// TODO:
//   - Precheck "projects_networks_zones" extension
func TestAccNetworkZone_project(t *testing.T) {
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZone_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "name", "custom.example.org"),
					resource.TestCheckResourceAttr("lxd_network_zone.zone", "project", projectName),
				),
			},
		},
	})
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
  name        = "custom.example.org"
  description = "descriptive"
  config = {
    "dns.nameservers"  = "ns.custom.example.org"
    "peers.ns.address" = "127.0.0.1"
  }
}
`
}

func testAccNetworkZone_project(project string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"
  config = {
	"features.networks" = false
	"features.networks.zones" = true
  }
}

resource "lxd_network_zone" "zone" {
  name    = "custom.example.org"
  project = lxd_project.project1.name

  config = {
    "dns.nameservers"  = "ns.custom.example.org"
    "peers.ns.address" = "127.0.0.1"
  }
}
	`, project)
}
