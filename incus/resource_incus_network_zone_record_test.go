package incus

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/lxc/incus/shared/api"
)

func TestAccNetworkZoneRecord_basic(t *testing.T) {
	var record api.NetworkZoneRecord

	recordName := petname.Name()
	zoneName := petname.Generate(3, ".")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZoneRecord(zoneName, recordName),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkZoneRecordExists(t, "incus_network_zone_record.record", &record),
					resource.TestCheckResourceAttr(
						"incus_network_zone_record.record", "name", recordName),
				),
			},
		},
	})
}

func TestAccNetworkZoneRecord_description(t *testing.T) {
	var record api.NetworkZoneRecord

	recordName := petname.Name()
	zoneName := petname.Generate(3, ".")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZoneRecord(zoneName, recordName),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkZoneRecordExists(t, "incus_network_zone_record.record", &record),
					resource.TestCheckResourceAttr("incus_network_zone_record.record",
						"description",
						"descriptive",
					),
				),
			},
		},
	})
}

func TestAccNetworkZoneRecord_entry(t *testing.T) {
	var record api.NetworkZoneRecord

	recordName := petname.Name()
	zoneName := petname.Generate(3, ".")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZoneRecord(zoneName, recordName),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkZoneRecordExists(t, "incus_network_zone_record.record", &record),
					resource.TestCheckTypeSetElemNestedAttrs(
						"incus_network_zone_record.record",
						"entry.*",
						map[string]string{"type": "CNAME", "value": "another", "ttl": "3600"},
					),
				),
			},
		},
	})
}

func TestAccNetworkZoneRecord_zone(t *testing.T) {
	var record api.NetworkZoneRecord

	recordName := petname.Name()
	zoneName := petname.Generate(3, ".")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZoneRecord(zoneName, recordName),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkZoneRecordExists(t, "incus_network_zone_record.record", &record),
					resource.TestCheckResourceAttr("incus_network_zone_record.record",
						"zone",
						zoneName,
					),
				),
			},
		},
	})
}

func testAccNetworkZoneRecordExists(t *testing.T, n string, record *api.NetworkZoneRecord) resource.TestCheckFunc {
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
		z, _, err := client.GetNetworkZoneRecord(rs.Primary.Attributes["zone"], rs.Primary.ID)
		if err != nil {
			return err
		}

		*record = *z

		return nil
	}
}

func testAccNetworkZoneRecord(zoneName, recordName string) string {
	return fmt.Sprintf(`
resource "incus_network_zone" "zone" {
  name = "%[1]s"

  config = {
    "dns.nameservers" = "ns.%[1]s"
    "peers.ns.address" = "127.0.0.1"
  }
}

resource "incus_network_zone_record" "record" {
  name = "%[2]s"
  zone = incus_network_zone.zone.id
  description = "descriptive"

  config = {}

  entry {
    type = "CNAME"
    value = "another"
    ttl = 3600
  }
}
`, zoneName, recordName)
}
