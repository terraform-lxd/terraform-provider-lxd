package lxd

import (
	"strings"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLxdStoragePool_importBasic(t *testing.T) {
	poolName := strings.ToLower(petname.Generate(2, "-"))
	resourceName := "lxd_storage_pool.storage_pool1"
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
