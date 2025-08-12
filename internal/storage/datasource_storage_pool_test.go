package storage_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccStoragePool_DS_dir(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	driverName := "dir"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool_DS(poolName, driverName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "name", poolName),
					resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "driver", driverName),
					resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "status", "Created"),
					resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "description", "Terraform provider storage pool test"),
					resource.TestCheckResourceAttrSet("data.lxd_storage_pool.pool", "config.source"),
				),
			},
		},
	})
}

func TestAccStoragePool_DS_config(t *testing.T) {
	drivers := []string{"zfs", "btrfs", "lvm"}

	for _, poolDriver := range drivers {
		poolName := acctest.GenerateName(2, "-")

		t.Run(fmt.Sprintf("%s[%s]", t.Name(), poolDriver), func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() {
					acctest.PreCheck(t)
					acctest.PreCheckStandalone(t)
				},
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccStoragePool_DS_config(poolName, poolDriver),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "name", poolName),
							resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "driver", poolDriver),
							resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "status", "Created"),
							resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "config.size", "128MiB"),
						),
					},
				},
			})
		})
	}
}

func TestAccStoragePool_DS_project(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	projectName := acctest.GenerateName(2, "-")
	driverName := "dir"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool_DS_project(poolName, driverName, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "name", poolName),
					resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "driver", driverName),
				),
			},
		},
	})
}

func TestAccStoragePool_DS_cluster(t *testing.T) {
	targets := acctest.PreCheckClustering(t, 2)

	poolName := acctest.GenerateName(2, "-")
	driverName := "dir"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool_DS_cluster(poolName, driverName, targets),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "name", poolName),
					resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "driver", driverName),
					resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "status", "Created"),
					resource.TestCheckResourceAttr("data.lxd_storage_pool.pool", "locations.#", strconv.Itoa(len(targets))),
				),
			},
		},
	})
}

func testAccStoragePool_DS(name string, driver string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool" {
  name        = %q
  driver      = %q
  description = "Terraform provider storage pool test"
}

data "lxd_storage_pool" "pool" {
  name = lxd_storage_pool.pool.name
}
	`, name, driver)
}

func testAccStoragePool_DS_config(name string, driver string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool" {
  name   = %q
  driver = %q

  config = {
    size = "128MiB"
  }
}

data "lxd_storage_pool" "pool" {
  name = lxd_storage_pool.pool.name
}
  `, name, driver)
}

func testAccStoragePool_DS_project(name string, driver string, project string) string {
	return fmt.Sprintf(`
resource "lxd_project" "proj" {
  name   = %q
}

resource "lxd_storage_pool" "pool" {
  name    = %q
  driver  = %q
  project = lxd_project.proj.name
}

data "lxd_storage_pool" "pool" {
  name    = lxd_storage_pool.pool.name
  project = lxd_storage_pool.pool.project
}
  `, project, name, driver)
}

func testAccStoragePool_DS_cluster(name string, driver string, targets []string) string {
	var config string
	var deps []string

	for i, target := range targets {
		deps = append(deps, "lxd_storage_pool.pool_node"+strconv.Itoa(i+1))
		config += fmt.Sprintf(`
resource "lxd_storage_pool" "pool_node%d" {
  name   = %q
  driver = %q
  target = %q
}`, i+1, name, driver, target)
	}

	config += fmt.Sprintf(`
resource "lxd_storage_pool" "pool" {
  depends_on = [ %[3]s ]
  name       = %[1]q
  driver     = %[2]q
}

data "lxd_storage_pool" "pool" {
  name   = lxd_storage_pool.pool.name
}
  `, name, driver, strings.Join(deps, ", "))

	return config
}
