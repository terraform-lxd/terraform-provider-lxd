package lxd

import (
	"fmt"
	"testing"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccNetworkZoneRecord_basic(t *testing.T) {
	var record api.NetworkZoneRecord

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZoneRecord_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkZoneRecordExists(t, "lxd_network_zone_record.record", &record),
					resource.TestCheckResourceAttr(
						"lxd_network_zone_record.record",
						"name",
						"ns",
					),
				),
			},
		},
	})
}

func TestAccNetworkZoneRecord_description(t *testing.T) {
	var record api.NetworkZoneRecord

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZoneRecord_desc(),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkZoneRecordExists(t, "lxd_network_record.record", &record),
					resource.TestCheckResourceAttr(
						"lxd_network_record.record",
						"description",
						"descriptive",
					),
				),
			},
		},
	})
}

func testAccNetworkZoneRecordExists(
	t *testing.T,
	n string,
	record *api.NetworkZoneRecord,
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
		z, _, err := client.GetNetworkZoneRecord(rs.Primary.Attributes["zone"], rs.Primary.ID)
		if err != nil {
			return err
		}

		*record = *z

		return nil
	}
}

func testAccNetworkZoneRecordConfig(
	record *api.NetworkZoneRecord,
	k, v string,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if record.Config == nil {
			return fmt.Errorf("No config")
		}

		for key, value := range record.Config {
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

func testAccNetworkZoneRecord_basic() string {
	return `
resource "lxd_network_zone" "zone" {
  name = "custom.example.org"
  description = "descriptive"

  config = {
    "dns.nameservers" = "ns.custom.example.org"
    "peers.ns.address" = "127.0.0.1"
  }
}

resource "lxd_network_zone_record" "record" {
  name = "ns"
  zone = lxd_network_record.zone.id

  config = {}

  entry {
    type = "CNAME"
    value = "another"
  }
}
`
}

func testAccNetworkZoneRecord_desc() string {
	return `
resource "lxd_network_zone_record" "record" {
  name = "ns"
  description = "descriptive"

  config = {}
}
`
}
