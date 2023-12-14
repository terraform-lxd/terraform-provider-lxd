package incus

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/lxc/incus/shared/api"
)

func TestAccVolumeCopyBasic(t *testing.T) {
	var volume api.StorageVolume
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")
	source := t.TempDir()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccVolumeCopyBasic(poolName, source, volumeName),
				Check: resource.ComposeTestCheckFunc(
					testAccVolumeExists(t, "incus_volume.volume1", &volume),
					resource.TestCheckResourceAttr("incus_volume.volume1", "name", volumeName),
					testAccVolumeExists(t, "incus_volume_copy.volume1_copy", &volume),
					resource.TestCheckResourceAttr("incus_volume_copy.volume1_copy", "name", fmt.Sprintf("%s-copy", volumeName))),
			},
			{
				Config: testAccVolumeCopyBasic(poolName, source, volumeName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func testAccVolumeCopyBasic(poolName, source, volumeName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
	name = "%[1]s"
	driver = "dir"
	config = {
	source = "%[2]s"
	}
}

resource "incus_volume" "volume1" {
	name = "%[3]s"
	pool = "${incus_storage_pool.pool1.name}"
}

resource "incus_volume_copy" "volume1_copy" {
	name = "%[3]s-copy"
	pool = "${incus_storage_pool.pool1.name}"
	source_pool = "${incus_storage_pool.pool1.name}"
	source_name = "${incus_volume.volume1.name}"
}
`,
		poolName, source, volumeName)
}
