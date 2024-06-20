package storage_test

import (
	"fmt"
	"testing"

	"github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccStorageBucket_basic(t *testing.T) {
	poolName := petname.Generate(2, "-")
	bucketName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucket_basic(poolName, bucketName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "dir"),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "name", bucketName),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "pool", poolName),
				),
			},
		},
	})
}

func TestAccStorageBucket_target(t *testing.T) {
	bucketName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucket_target(bucketName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "name", bucketName),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "pool", "default"),
				),
			},
		},
	})
}

func TestAccStorageBucket_project(t *testing.T) {
	projectName := petname.Name()
	bucketName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucket_project(projectName, bucketName),
				Check: resource.ComposeTestCheckFunc(

					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "pool", "default"),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "project", projectName),
				),
			},
		},
	})
}

func TestAccStorageBucket_importBasic(t *testing.T) {
	poolName := petname.Generate(2, "-")
	bucketName := petname.Generate(2, "-")
	resourceName := "incus_storage_bucket.bucket1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucket_basic(poolName, bucketName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("/%s/%s", poolName, bucketName),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportState:                          true,
				ImportStateVerify:                    true,
			},
		},
	})
}

func TestAccStorageBucket_importProject(t *testing.T) {
	projectName := petname.Generate(2, "-")
	bucketName := petname.Generate(2, "-")
	resourceName := "incus_storage_bucket.bucket1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucket_project(projectName, bucketName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("%s/default/%s", projectName, bucketName),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    true,
				ImportState:                          true,
			},
		},
	})
}

func testAccStorageBucket_basic(poolName string, bucketName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%s"
  driver = "dir"
}

resource "incus_storage_bucket" "bucket1" {
  name = "%s"
  pool = incus_storage_pool.pool1.name
}
	`, poolName, bucketName)
}

func testAccStorageBucket_target(bucketName string) string {
	return fmt.Sprintf(`
resource "incus_storage_bucket" "bucket1" {
	name    = "%s"
	pool    = "default"
	target = "node-2"
}
 	`, bucketName)
}

func testAccStorageBucket_project(projectName string, bucketName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
	name = "%s"
	config = {
		"features.storage.volumes" = false
	}
}

resource "incus_storage_bucket" "bucket1" {
	name    = "%s"
	pool    = "default"
	project = incus_project.project1.name
}
	`, projectName, bucketName)
}
