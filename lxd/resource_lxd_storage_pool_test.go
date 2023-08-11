package lxd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/canonical/lxd/shared/api"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccStoragePool_basic(t *testing.T) {
	var pool api.StoragePool
	poolName := strings.ToLower(petname.Generate(2, "-"))
	source := t.TempDir()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool_basic(poolName, source),
				Check: resource.ComposeTestCheckFunc(
					testAccStoragePoolExists(t, "lxd_storage_pool.storage_pool1", &pool),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
				),
			},
		},
	})
}

func TestAccStoragePool_target(t *testing.T) {
	var pool api.StoragePool
	poolName := strings.ToLower(petname.Generate(2, "-"))

	// t.TempDir cannot be used here as the temp directory
	// is only created on the node running the test - not any
	// of the other nodes in the cluster.
	source := "/mnt"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckClustering(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool_target(poolName, source),
				Check: resource.ComposeTestCheckFunc(
					testAccStoragePoolExists(t, "lxd_storage_pool.storage_pool1", &pool),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1_node1", "config.source", source),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1_node2", "config.source", source),
				),
			},
		},
	})
}

func TestAccStoragePool_project(t *testing.T) {
	var pool api.StoragePool
	var project api.Project

	source := t.TempDir()
	projectName := strings.ToLower(petname.Name())
	poolName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool_project(poolName, source, projectName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "lxd_project.project1", &project),
					testAccStoragePoolExistsInProject(t, "lxd_storage_pool.storage_pool1", &pool, projectName),
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

func testAccStoragePoolExistsInProject(t *testing.T, n string, pool *api.StoragePool, project string) resource.TestCheckFunc {
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
		client = client.UseProject(project)
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

func testAccStoragePool_basic(name, source string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "storage_pool1" {
  name = "%s"
  driver = "dir"
  config = {
    source = "%s"
  }
}
	`, name, source)
}

func testAccStoragePool_target(name, source string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "storage_pool1_node1" {
  target = "node1"

  name = "%s"
  driver = "dir"
  config = {
    source = "%s"
  }
}

resource "lxd_storage_pool" "storage_pool1_node2" {
  target = "node2"

  name = "%s"
  driver = "dir"
  config = {
    source = "%s"
  }
}

resource "lxd_storage_pool" "storage_pool1" {
  depends_on = [
    "lxd_storage_pool.storage_pool1_node1",
    "lxd_storage_pool.storage_pool1_node2",
  ]

  name = "%s"
  driver = "dir"
}
	`, name, source, name, source, name)
}

func testAccStoragePool_project(name, source, project string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
  config = {
	"features.storage.volumes" = false
	"features.images" = false
	"features.profiles" = false
	"features.storage.buckets" = false
  }
}
resource "lxd_storage_pool" "storage_pool1" {
  name = "%s"
  driver = "dir"
  config = {
    source = "%s"
  }
  project = lxd_project.project1.name
}
	`, project, name, source)
}
