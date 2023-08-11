package lxd

import (
	"fmt"
	"testing"

	"github.com/canonical/lxd/shared/api"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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
					testAccVolumeExists(t, "lxd_volume.volume1", &volume),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					testAccVolumeExists(t, "lxd_volume_copy.volume1_copy", &volume),
					resource.TestCheckResourceAttr("lxd_volume_copy.volume1_copy", "name", fmt.Sprintf("%s-copy", volumeName))),
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
resource "lxd_storage_pool" "pool1" {
	name = "%[1]s"
	driver = "dir"
	config = {
	source = "%[2]s"
	}
}

resource "lxd_volume" "volume1" {
	name = "%[3]s"
	pool = "${lxd_storage_pool.pool1.name}"
}

resource "lxd_volume_copy" "volume1_copy" {
	name = "%[3]s-copy"
	pool = "${lxd_storage_pool.pool1.name}"
	source_pool = "${lxd_storage_pool.pool1.name}"
	source_name = "${lxd_volume.volume1.name}"
}
`,
		poolName, source, volumeName)
}
