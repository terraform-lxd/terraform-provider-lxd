package incus

import (
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccIncusStoragePool_importBasic(t *testing.T) {
	poolName := petname.Generate(2, "-")
	resourceName := "incus_storage_pool.storage_pool1"
	source := t.TempDir()

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool_basic(poolName, source),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
