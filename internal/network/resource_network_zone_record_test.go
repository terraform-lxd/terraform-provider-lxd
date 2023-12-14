package network_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/maveonair/terraform-provider-incus/internal/acctest"
)

func TestAccNetworkZoneRecord_basic(t *testing.T) {
	recordName := petname.Generate(2, "-")
	zoneName := petname.Generate(3, ".")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZoneRecord(zoneName, recordName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_zone.zone", "name", zoneName),
					resource.TestCheckResourceAttr("incus_network_zone.zone", "config.dns.nameservers", fmt.Sprintf("ns.%s", zoneName)),
					resource.TestCheckResourceAttr("incus_network_zone_record.record", "name", recordName),
					resource.TestCheckResourceAttr("incus_network_zone_record.record", "zone", zoneName),
					resource.TestCheckResourceAttr("incus_network_zone_record.record", "description", "Network zone record"),
				),
			},
		},
	})
}

func TestAccNetworkZoneRecord_entries(t *testing.T) {
	recordName := petname.Generate(2, "-")
	zoneName := petname.Generate(3, ".")

	entry1 := map[string]string{
		"type":  "CNAME",
		"value": "one",
		"ttl":   "3600",
	}

	entry2 := map[string]string{
		"type":  "CNAME",
		"value": "two",
		"ttl":   "3600",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZoneRecord_entries_1(zoneName, recordName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_zone.zone", "name", zoneName),
					resource.TestCheckResourceAttr("incus_network_zone_record.record", "name", recordName),
					resource.TestCheckResourceAttr("incus_network_zone_record.record", "zone", zoneName),
					resource.TestCheckTypeSetElemNestedAttrs("incus_network_zone_record.record", "entry.*", entry1),
				),
			},
			{
				Config: testAccNetworkZoneRecord_entries_2(zoneName, recordName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_zone.zone", "name", zoneName),
					resource.TestCheckResourceAttr("incus_network_zone_record.record", "name", recordName),
					resource.TestCheckResourceAttr("incus_network_zone_record.record", "zone", zoneName),
					resource.TestCheckTypeSetElemNestedAttrs("incus_network_zone_record.record", "entry.*", entry2),
				),
			},
		},
	})
}

func TestAccNetworkZoneRecord_importBasic(t *testing.T) {
	resourceName := "incus_network_zone_record.record"
	recordName := petname.Generate(2, "-")
	zoneName := petname.Generate(3, ".")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkZoneRecord(zoneName, recordName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("/%s/%s", zoneName, recordName),
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func testAccNetworkZoneRecord(zoneName, recordName string) string {
	return fmt.Sprintf(`
resource "incus_network_zone" "zone" {
  name = "%[1]s"

  config = {
    "dns.nameservers"  = "ns.%[1]s"
    "peers.ns.address" = "127.0.0.1"
  }
}

resource "incus_network_zone_record" "record" {
  name        = "%[2]s"
  zone        = incus_network_zone.zone.name
  description = "Network zone record"
}
`, zoneName, recordName)
}

func testAccNetworkZoneRecord_entries_1(zoneName, recordName string) string {
	return fmt.Sprintf(`
resource "incus_network_zone" "zone" {
  name = "%[1]s"

  config = {
    "dns.nameservers"  = "ns.%[1]s"
    "peers.ns.address" = "127.0.0.1"
  }
}

resource "incus_network_zone_record" "record" {
  name = "%[2]s"
  zone = incus_network_zone.zone.name

  entry {
    type  = "CNAME"
    value = "one"
    ttl   = 3600
  }
}
`, zoneName, recordName)
}

func testAccNetworkZoneRecord_entries_2(zoneName, recordName string) string {
	return fmt.Sprintf(`
resource "incus_network_zone" "zone" {
  name = "%[1]s"

  config = {
    "dns.nameservers"  = "ns.%[1]s"
    "peers.ns.address" = "127.0.0.1"
  }
}

resource "incus_network_zone_record" "record" {
  name = "%[2]s"
  zone = incus_network_zone.zone.name

  entry {
    type  = "CNAME"
    value = "two"
    ttl   = 3600
  }
}
`, zoneName, recordName)
}
