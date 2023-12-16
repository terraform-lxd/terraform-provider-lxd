package storage_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccStorageVolumeCopy_basic(t *testing.T) {
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolumeCopy_basic(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "lvm"),
					resource.TestCheckResourceAttr("incus_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_volume_copy.volume1_copy", "name", fmt.Sprintf("%s-copy", volumeName)),
					resource.TestCheckResourceAttr("incus_volume_copy.volume1_copy", "pool", "default"),
					resource.TestCheckResourceAttr("incus_volume_copy.volume1_copy", "source_name", volumeName),
					resource.TestCheckResourceAttr("incus_volume_copy.volume1_copy", "source_pool", poolName),
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
resource "incus_storage_pool" "pool1" {
  name   = "%[1]s"
  driver = "lvm"
}

resource "incus_volume" "volume1" {
  name = "%[2]s"
  pool = incus_storage_pool.pool1.name
}

resource "incus_volume_copy" "volume1_copy" {
  name        = "%[2]s-copy"
  pool        = "default"
  source_pool = incus_storage_pool.pool1.name
  source_name = incus_volume.volume1.name
}
`,
		poolName, volumeName)
}
