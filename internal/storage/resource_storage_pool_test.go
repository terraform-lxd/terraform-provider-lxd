package storage_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccStoragePool_dir(t *testing.T) {
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
				Config: testAccStoragePool(poolName, driverName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "0"),
					// Ensure computed keys are not tracked.
					resource.TestCheckNoResourceAttr("lxd_storage_pool.storage_pool1", "config.source"),
				),
			},
			{
				// Ensure no error is thrown on update.
				Config: testAccStoragePool(poolName, driverName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "0"),
				),
			},
		},
	})
}

func TestAccStoragePool_zfs(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	driverName := "zfs"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool(poolName, driverName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "0"),
					// Ensure computed keys are not tracked.
					resource.TestCheckNoResourceAttr("lxd_storage_pool.storage_pool1", "config.zfs.pool_name"),
					resource.TestCheckNoResourceAttr("lxd_storage_pool.storage_pool1", "config.size"),
					resource.TestCheckNoResourceAttr("lxd_storage_pool.storage_pool1", "config.source"),
				),
			},
			{
				// Ensure no error is thrown on update.
				Config: testAccStoragePool(poolName, driverName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "0"),
				),
			},
		},
	})
}

func TestAccStoragePool_lvm(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	driverName := "lvm"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool(poolName, driverName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "0"),
					// Ensure computed keys are not tracked.
					resource.TestCheckNoResourceAttr("lxd_storage_pool.storage_pool1", "config.lvm.vg_name"),
					resource.TestCheckNoResourceAttr("lxd_storage_pool.storage_pool1", "config.lvm.thinpool_name"),
					resource.TestCheckNoResourceAttr("lxd_storage_pool.storage_pool1", "config.size"),
					resource.TestCheckNoResourceAttr("lxd_storage_pool.storage_pool1", "config.source"),
				),
			},
			{
				// Ensure no error is thrown on update.
				Config: testAccStoragePool(poolName, driverName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "0"),
				),
			},
		},
	})
}

func TestAccStoragePool_btrfs(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	driverName := "btrfs"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool(poolName, driverName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "0"),
					// Ensure computed keys are not tracked.
					resource.TestCheckNoResourceAttr("lxd_storage_pool.storage_pool1", "config.size"),
					resource.TestCheckNoResourceAttr("lxd_storage_pool.storage_pool1", "config.source"),
				),
			},
			{
				// Ensure no error is thrown on update.
				Config: testAccStoragePool(poolName, driverName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "0"),
				),
			},
		},
	})
}

func TestAccStoragePool_config(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool_config(poolName, "zfs"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "1"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.size", "128MiB"),
				),
			},
			{
				Config: testAccStoragePool_config(poolName, "lvm"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", "lvm"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "1"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.size", "128MiB"),
				),
			},
			{
				Config: testAccStoragePool_config(poolName, "btrfs"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", "btrfs"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "1"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.size", "128MiB"),
				),
			},
		},
	})
}

func TestAccStoragePool_configSource(t *testing.T) {
	drivers := []string{"dir", "zfs", "btrfs", "lvm"}

	// Skip test if we have no root permissions. Root permissions are required
	// for creating loopback devices.
	acctest.PreCheckRoot(t)

	for _, poolDriver := range drivers {
		poolName := acctest.GenerateName(2, "-")
		poolSource, cleanup := ensureSource(t, poolDriver)

		t.Run(fmt.Sprintf("%s[%s]", t.Name(), poolDriver), func(t *testing.T) {
			defer cleanup()
			resource.Test(t, resource.TestCase{
				PreCheck: func() {
					acctest.PreCheck(t)
					acctest.PreCheckStandalone(t)
				},
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccStoragePool_configSource(poolName, poolDriver, poolSource),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
							resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", poolDriver),
							resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "source", poolSource),
							resource.TestCheckNoResourceAttr("lxd_storage_pool.storage_pool1", "config.source"),
						),
					},
					{
						// Reapply the same config to ensure there is no drift in terraform state.
						Config: testAccStoragePool_configSource(poolName, poolDriver, poolSource),
					},
				},
			})
		})
	}
}

func TestAccStoragePool_project(t *testing.T) {
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
				Config: testAccStoragePool_project(poolName, driverName, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_project.project1", "config.%", "1"),
					resource.TestCheckResourceAttr("lxd_project.project1", "config.features.storage.volumes", "false"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "project", projectName),
				),
			},
		},
	})
}

func TestAccStoragePool_target(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	driverName := "dir"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool_target(poolName, driverName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1_node1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1_node1", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1_node1", "target", "node-1"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1_node2", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1_node2", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1_node2", "target", "node-2"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", driverName),
				),
			},
		},
	})
}

func TestAccStoragePool_importBasic(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	driverName := "zfs"
	resourceName := "lxd_storage_pool.storage_pool1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool(poolName, driverName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        poolName,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    true,
				ImportState:                          true,
			},
		},
	})
}

func TestAccStoragePool_importConfig(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	driverName := "zfs"
	resourceName := "lxd_storage_pool.storage_pool1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool_config(poolName, driverName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        poolName,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    false, // State of "config" will be always empty.
				ImportState:                          true,
			},
		},
	})
}

func TestAccStoragePool_importProject(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	projectName := acctest.GenerateName(2, "-")
	driverName := "zfs"
	resourceName := "lxd_storage_pool.storage_pool1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStoragePool_project(poolName, driverName, projectName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("%s/%s", projectName, poolName),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    true,
				ImportState:                          true,
			},
		},
	})
}

func testAccStoragePool(name string, driver string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "storage_pool1" {
  name   = "%s"
  driver = "%s"
}
	`, name, driver)
}

func testAccStoragePool_config(name string, driver string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "storage_pool1" {
  name   = "%s"
  driver = "%s"
  config = {
    size = "128MiB"
  }
}
	`, name, driver)
}

func testAccStoragePool_configSource(name string, driver string, source string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "storage_pool1" {
  name   = "%s"
  driver = "%s"
  source = "%s"
}
	`, name, driver, source)
}

func testAccStoragePool_project(name string, driver string, project string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
  config = {
    "features.storage.volumes" = false
  }
}

resource "lxd_storage_pool" "storage_pool1" {
  name    = "%s"
  driver  = "%s"
  project = lxd_project.project1.name
}
	`, project, name, driver)
}

func testAccStoragePool_target(name, driver string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "storage_pool1_node1" {
  name   = "%[1]s"
  driver = "%[2]s"
  target = "node-1"
}

resource "lxd_storage_pool" "storage_pool1_node2" {
  name   = "%[1]s"
  driver = "%[2]s"
  target = "node-2"
}

resource "lxd_storage_pool" "storage_pool1" {
  depends_on = [
    lxd_storage_pool.storage_pool1_node1,
    lxd_storage_pool.storage_pool1_node2,
  ]

  name   = "%[1]s"
  driver = "%[2]s"
}
	`, name, driver)
}

// ensureSource ensures temporary storage pool source is created based on the provided
// storage pool driver. For "dir", a temporary directory is created. For "zfs", "lvm",
// and "btrfs" a loopback device pointing to a temporary file is created.
func ensureSource(t *testing.T, driver string) (source string, cleanup func()) {
	switch driver {
	case "dir":
		// For directory storage driver, just create an empty directory.

		source = filepath.Join(os.TempDir(), "tf-storage-pool-dir")
		_ = os.RemoveAll(source)

		err := os.Mkdir(source, os.ModePerm)
		if err != nil {
			t.Fatalf("Failed to create temporary directory: %v", err)
		}

		cleanup = func() {
			_ = os.RemoveAll(source)
		}
	case "zfs", "btrfs", "lvm":
		// For zfs, btrfs, and lvm storage drivers, create a temporary file
		// and attach it as a loopback device.

		disk, err := os.CreateTemp(os.TempDir(), "tf-storage-pool-disk-*")
		if err != nil {
			t.Fatalf("Failed to create temporary disk file: %v", err)
		}
		defer disk.Close()

		err = disk.Truncate(128 * 1024 * 1024) // 128 MiB
		if err != nil {
			t.Fatalf("Failed to truncate temporary disk file: %v", err)
		}

		// Create the loopback device.
		var bufOut, bufErr bytes.Buffer
		cmd := exec.Command("sudo", "losetup", "-fP", "--show", disk.Name())
		cmd.Stdout = &bufOut
		cmd.Stderr = &bufErr
		err = cmd.Run()
		if err != nil {
			t.Fatalf("Failed to create loopback device (%v): %s", err, bufErr.String())
		}

		// Retrieve the loopback device name.
		source = strings.TrimSpace(bufOut.String())

		cleanup = func() {
			// Detach loopback device and remove temporary file.
			_ = exec.Command("sudo", "losetup", "-d", source).Run()
			_ = os.RemoveAll(disk.Name())
		}
	default:
		t.Fatalf("Cannot create source for storage driver %q", driver)
	}

	return source, cleanup
}
