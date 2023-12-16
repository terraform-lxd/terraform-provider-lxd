package network_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccNetworkZone_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "projects_networks_zones")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZone_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_zone.zone", "name", "custom.example.org"),
					resource.TestCheckResourceAttr("incus_network_zone.zone", "config.%", "2"),
					resource.TestCheckResourceAttr("incus_network_zone.zone", "config.dns.nameservers", "ns.custom.example.org"),
					resource.TestCheckResourceAttr("incus_network_zone.zone", "config.peers.ns.address", "127.0.0.1"),
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
					resource.TestCheckResourceAttr("incus_network_zone.zone", "name", "custom.example.org"),
					resource.TestCheckResourceAttr("incus_network_zone.zone", "description", "descriptive"),
					resource.TestCheckResourceAttr("incus_network_zone.zone", "config.%", "2"),
					resource.TestCheckResourceAttr("incus_network_zone.zone", "config.dns.nameservers", "ns.custom.example.org"),
					resource.TestCheckResourceAttr("incus_network_zone.zone", "config.peers.ns.address", "127.0.0.1"),
				),
			},
			{
				// Ensure no changes on reapply.
				Config: testAccNetworkZone_desc(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_zone.zone", "name", "custom.example.org"),
					resource.TestCheckResourceAttr("incus_network_zone.zone", "description", "descriptive"),
				),
			},
		},
	})
}

func TestAccNetworkZone_project(t *testing.T) {
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "projects_networks_zones")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZone_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_network_zone.zone", "name", "custom.example.org"),
					resource.TestCheckResourceAttr("incus_network_zone.zone", "project", projectName),
				),
			},
		},
	})
}

func TestAccNetworkZone_importBasic(t *testing.T) {
	resourceName := "incus_network_zone.zone"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZone_basic(),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        "custom.example.org",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccNetworkZone_importProject(t *testing.T) {
	resourceName := "incus_network_zone.zone"
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "projects_networks_zones")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZone_project(projectName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("%s/custom.example.org", projectName),
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func testAccNetworkZone_basic() string {
	return `
resource "incus_network_zone" "zone" {
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
resource "incus_network_zone" "zone" {
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
resource "incus_project" "project1" {
  name = "%s"
  config = {
    "features.networks"       = false
    "features.networks.zones" = true
  }
}

resource "incus_network_zone" "zone" {
  name    = "custom.example.org"
  project = incus_project.project1.name

  config = {
    "dns.nameservers"  = "ns.custom.example.org"
    "peers.ns.address" = "127.0.0.1"
  }
}
	`, project)
}
