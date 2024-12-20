package storage_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccStorageVolumeCopy_basic(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	volumeName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolumeCopy_basic(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "lvm"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("lxd_volume_copy.volume1_copy", "name", fmt.Sprintf("%s-copy", volumeName)),
					resource.TestCheckResourceAttr("lxd_volume_copy.volume1_copy", "pool", "default"),
					resource.TestCheckResourceAttr("lxd_volume_copy.volume1_copy", "source_name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume_copy.volume1_copy", "source_pool", poolName),
				),
			},
			{
				Config: testAccStorageVolumeCopy_basic(poolName, volumeName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func testAccStorageVolumeCopy_basic(poolName, volumeName string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name   = "%[1]s"
  driver = "lvm"
}

resource "lxd_volume" "volume1" {
  name = "%[2]s"
  pool = lxd_storage_pool.pool1.name
}

resource "lxd_volume_copy" "volume1_copy" {
  name        = "%[2]s-copy"
  pool        = "default"
  source_pool = lxd_storage_pool.pool1.name
  source_name = lxd_volume.volume1.name
}
`,
		poolName, volumeName)
}
