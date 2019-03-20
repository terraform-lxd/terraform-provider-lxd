package lxd

import (
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccLxdStoragePool_importBasic(t *testing.T) {
	poolName := strings.ToLower(petname.Generate(2, "-"))
	resourceName := "lxd_storage_pool.storage_pool1"

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccStoragePoolBasicConfig(poolName),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
