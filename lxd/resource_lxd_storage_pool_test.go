package lxd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"github.com/lxc/lxd/shared/api"
)

func TestAccStoragePool_basic(t *testing.T) {
	var pool api.StoragePool
	poolName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool_basic(poolName),
				Check: resource.ComposeTestCheckFunc(
					testAccStoragePoolExists(t, "lxd_storage_pool.storage_pool1", &pool),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
				),
			},
		},
	})
}

func testAccStoragePoolExists(t *testing.T, n string, pool *api.StoragePool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		poolName := rs.Primary.ID

		client, err := testAccProvider.Meta().(*lxdProvider).GetInstanceServer("")
		if err != nil {
			return err
		}
		v, _, err := client.GetStoragePool(poolName)
		if err != nil {
			return err
		}

		*pool = *v

		return nil
	}
}

func testAccStoragePoolConfig(pool *api.StoragePool, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if pool.Config == nil {
			return fmt.Errorf("No config")
		}

		for key, value := range pool.Config {
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

func testAccStoragePool_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "storage_pool1" {
  name = "%s"
  driver = "dir"
  config = {
    source = "/mnt"
  }
}
	`, name)
}
