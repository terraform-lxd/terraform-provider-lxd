package storage_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccStoragePool_dir(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	driverName := "dir"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccStoragePool(poolName, driverName),
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
				Config: acctest.Provider() + testAccStoragePool(poolName, driverName),
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
				Config: acctest.Provider() + testAccStoragePool(poolName, driverName),
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
				Config: acctest.Provider() + testAccStoragePool(poolName, driverName),
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
				Config: acctest.Provider() + testAccStoragePool(poolName, driverName),
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
				Config: acctest.Provider() + testAccStoragePool(poolName, driverName),
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
				Config: acctest.Provider() + testAccStoragePool(poolName, driverName),
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
				Config: acctest.Provider() + testAccStoragePool(poolName, driverName),
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
				Config: acctest.Provider() + testAccStoragePool_config(poolName, "zfs"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "1"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.size", "128MiB"),
				),
			},
			{
				Config: acctest.Provider() + testAccStoragePool_config(poolName, "lvm"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", "lvm"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.%", "1"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.size", "128MiB"),
				),
			},
			{
				Config: acctest.Provider() + testAccStoragePool_config(poolName, "btrfs"),
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
						Config: acctest.Provider() + testAccStoragePool_configSource(poolName, poolDriver, poolSource),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
							resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", poolDriver),
							resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "config.source", poolSource),
						),
					},
					{
						// Reapply the same config to ensure there is no drift in terraform state.
						Config: acctest.Provider() + testAccStoragePool_configSource(poolName, poolDriver, poolSource),
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
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccStoragePool_project(poolName, driverName, projectName),
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

func TestAccStoragePool_clusterConfigNoMemberOverrides(t *testing.T) {
	targets := acctest.PreCheckClustering(t, 2)
	poolName := acctest.GenerateName(2, "-")
	driverName := "dir"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccStoragePool(poolName, driverName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", "members.%", fmt.Sprintf("%d", len(targets))),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", fmt.Sprintf("members.%s.config.%%", targets[0]), "0"),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool1", fmt.Sprintf("members.%s.config.%%", targets[1]), "0"),
				),
			},
			{
				Config: acctest.Provider() + testAccStoragePool(poolName, driverName),
			},
		},
	})
}

func TestAccStoragePool_clusterConfigEmptyMemberOverrides(t *testing.T) {
	targets := acctest.PreCheckClustering(t, 2)
	poolName := acctest.GenerateName(2, "-")
	driverName := "dir"

	step1Overrides := map[string]map[string]string{
		targets[0]: {},
	}
	step2Overrides := map[string]map[string]string{}
	for _, target := range targets[1:] {
		step2Overrides[target] = map[string]string{}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccStoragePool_memberOverrides(poolName, driverName, nil, step1Overrides),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", "members.%", fmt.Sprintf("%d", len(targets))),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", "member_overrides.%", "1"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("member_overrides.%s.config.%%", targets[0]), "0"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%%", targets[0]), "0"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%%", targets[1]), "0"),
				),
			},
			{
				Config: acctest.Provider() + testAccStoragePool_memberOverrides(poolName, driverName, nil, step2Overrides),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", "members.%", fmt.Sprintf("%d", len(targets))),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", "member_overrides.%", fmt.Sprintf("%d", len(targets)-1)),
					resource.TestCheckNoResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("member_overrides.%s", targets[0])),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%%", targets[0]), "0"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%%", targets[1]), "0"),
				),
			},
			{
				// Reapply the same config to ensure there is no drift in terraform state.
				Config: acctest.Provider() + testAccStoragePool_memberOverrides(poolName, driverName, nil, step2Overrides),
			},
		},
	})
}

func TestAccStoragePool_clusterConfigLifecycle(t *testing.T) {
	targets := acctest.PreCheckClustering(t, 2)
	poolName := acctest.GenerateName(2, "-")
	overrideTarget := targets[0]
	defaultTarget := targets[1]

	// Use the "dir" driver with the "source.recover" key (a node-specific boolean key) so the
	// test does not depend on loop devices or kernel storage modules, which are unavailable
	// when LXD runs nested in a container, as is the case in the cluster test environment.
	driverName := "dir"
	configKey := "source.recover"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "storage_source_recover")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Set config key in global settings.
				Config: acctest.Provider() + testAccStoragePool_memberOverrides(poolName, driverName, map[string]string{configKey: "false"}, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", "config."+configKey, "false"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "false"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "false"),
				),
			},
			{
				// Set config key in global settings and one member override.
				Config: acctest.Provider() + testAccStoragePool_memberOverrides(poolName, driverName, map[string]string{configKey: "false"}, map[string]map[string]string{overrideTarget: {configKey: "true"}}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", "config."+configKey, "false"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("member_overrides.%s.config.%s", overrideTarget, configKey), "true"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "true"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "false"),
				),
			},
			{
				// Change the value of one member override.
				Config: acctest.Provider() + testAccStoragePool_memberOverrides(poolName, driverName, map[string]string{configKey: "false"}, map[string]map[string]string{overrideTarget: {configKey: "false"}}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", "config."+configKey, "false"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("member_overrides.%s.config.%s", overrideTarget, configKey), "false"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "false"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "false"),
				),
			},
			{
				// Change global config while keeping the member override.
				Config: acctest.Provider() + testAccStoragePool_memberOverrides(poolName, driverName, map[string]string{configKey: "true"}, map[string]map[string]string{overrideTarget: {configKey: "false"}}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", "config."+configKey, "true"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("member_overrides.%s.config.%s", overrideTarget, configKey), "false"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "false"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "true"),
				),
			},
			{
				// Set different member override values for both members.
				Config: acctest.Provider() + testAccStoragePool_memberOverrides(poolName, driverName, map[string]string{configKey: "true"}, map[string]map[string]string{overrideTarget: {configKey: "true"}, defaultTarget: {configKey: "false"}}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", "config."+configKey, "true"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("member_overrides.%s.config.%s", overrideTarget, configKey), "true"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("member_overrides.%s.config.%s", defaultTarget, configKey), "false"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "true"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "false"),
				),
			},
			{
				// Remove the config key from member overrides and use the global value.
				Config: acctest.Provider() + testAccStoragePool_memberOverrides(poolName, driverName, map[string]string{configKey: "false"}, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("member_overrides.%s", overrideTarget)),
					resource.TestCheckNoResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("member_overrides.%s", defaultTarget)),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", "config."+configKey, "false"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "false"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "false"),
				),
			},
			{
				// Remove the global config key and ensure it is no longer managed.
				Config: acctest.Provider() + testAccStoragePool_memberOverrides(poolName, driverName, nil, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("lxd_storage_pool.pool", "config."+configKey),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%%", overrideTarget), "0"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool", fmt.Sprintf("members.%s.config.%%", defaultTarget), "0"),
				),
			},
			{
				// Reapply final state to ensure no error and no drift.
				Config: acctest.Provider() + testAccStoragePool_memberOverrides(poolName, driverName, nil, nil),
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
				Config: acctest.Provider() + testAccStoragePool(poolName, driverName),
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
				Config: acctest.Provider() + testAccStoragePool_config(poolName, driverName),
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
				Config: acctest.Provider() + testAccStoragePool_project(poolName, driverName, projectName),
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
  config = {
    source = "%s"
  }
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

func testAccStoragePool_memberOverrides(name string, driver string, config map[string]string, memberOverrides map[string]map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `
resource "lxd_storage_pool" "pool" {
  name   = %q
  driver = %q
`, name, driver)

	if len(config) > 0 {
		configKeys := make([]string, 0, len(config))
		for k := range config {
			configKeys = append(configKeys, k)
		}

		sort.Strings(configKeys)

		b.WriteString(`
  config = {
`)

		for _, key := range configKeys {
			fmt.Fprintf(&b, "    %q = %q\n", key, config[key])
		}

		fmt.Fprintf(&b, `
  }
`)
	}

	if len(memberOverrides) > 0 {
		memberKeys := make([]string, 0, len(memberOverrides))
		for member := range memberOverrides {
			memberKeys = append(memberKeys, member)
		}

		sort.Strings(memberKeys)

		b.WriteString("\n  member_overrides = {\n")
		for _, member := range memberKeys {
			overrideConfig := memberOverrides[member]

			fmt.Fprintf(&b, "    %q = {\n", member)
			b.WriteString("      config = {\n")

			overrideKeys := make([]string, 0, len(overrideConfig))
			for key := range overrideConfig {
				overrideKeys = append(overrideKeys, key)
			}

			sort.Strings(overrideKeys)
			for _, key := range overrideKeys {
				fmt.Fprintf(&b, "        %q = %q\n", key, overrideConfig[key])
			}

			b.WriteString("      }\n")
			b.WriteString("    }\n")
		}

		b.WriteString("  }\n")
	}

	b.WriteString("}\n")
	return b.String()
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
