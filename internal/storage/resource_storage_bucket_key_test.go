package storage_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccStorageBucketKey_basic(t *testing.T) {
	poolName := petname.Generate(2, "-")
	bucketName := petname.Generate(2, "-")
	keyName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
			acctest.PreCheckAPIExtensions(t, "storage_buckets")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucketKey_basic(poolName, bucketName, keyName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "dir"),
					resource.TestCheckResourceAttr("lxd_storage_bucket.bucket1", "name", bucketName),
					resource.TestCheckResourceAttr("lxd_storage_bucket.bucket1", "pool", poolName),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "name", keyName),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "bucket", bucketName),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "pool", poolName),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "role", "read-only"),
					resource.TestCheckResourceAttrSet("lxd_storage_bucket_key.key1", "access_key"),
					resource.TestCheckResourceAttrSet("lxd_storage_bucket_key.key1", "secret_key"),
				),
			},
		},
	})
}

func TestAccStorageBucketKey_role(t *testing.T) {
	bucketName := petname.Generate(2, "-")
	keyName := petname.Generate(2, "-")
	role := "admin"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "storage_buckets")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucketKey_role(bucketName, keyName, role),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_bucket.bucket1", "name", bucketName),
					resource.TestCheckResourceAttr("lxd_storage_bucket.bucket1", "pool", "default"),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "name", keyName),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "bucket", bucketName),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "pool", "default"),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "role", role),
					resource.TestCheckResourceAttrSet("lxd_storage_bucket_key.key1", "access_key"),
					resource.TestCheckResourceAttrSet("lxd_storage_bucket_key.key1", "secret_key"),
				),
			},
		},
	})
}

func TestAccStorageBucketKey_project(t *testing.T) {
	projectName := petname.Name()
	bucketName := petname.Generate(2, "-")
	keyName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "storage_buckets")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucketKey_project(projectName, bucketName, keyName),
				Check: resource.ComposeTestCheckFunc(

					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_storage_bucket.bucket1", "pool", "default"),
					resource.TestCheckResourceAttr("lxd_storage_bucket.bucket1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "name", keyName),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "bucket", bucketName),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "pool", "default"),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_storage_bucket_key.key1", "role", "read-only"),
					resource.TestCheckResourceAttrSet("lxd_storage_bucket_key.key1", "access_key"),
					resource.TestCheckResourceAttrSet("lxd_storage_bucket_key.key1", "secret_key"),
				),
			},
		},
	})
}

func TestAccStorageBucketKey_importBasic(t *testing.T) {
	poolName := petname.Generate(2, "-")
	bucketName := petname.Generate(2, "-")
	keyName := petname.Generate(2, "-")
	resourceName := "lxd_storage_bucket_key.key1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
			acctest.PreCheckAPIExtensions(t, "storage_buckets")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucketKey_basic(poolName, bucketName, keyName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("/%s/%s/%s", poolName, bucketName, keyName),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 state, got %d", len(states))
					}
					state := states[0]
					if state.Attributes["access_key"] == "" {
						return fmt.Errorf("expected access_key to be set")
					}

					if state.Attributes["secret_key"] == "" {
						return fmt.Errorf("expected access_key to be set")
					}

					return nil
				},
			},
		},
	})
}

func TestAccStorageBucketKey_importProject(t *testing.T) {
	projectName := petname.Generate(2, "-")
	bucketName := petname.Generate(2, "-")
	keyName := petname.Generate(2, "-")
	resourceName := "lxd_storage_bucket_key.key1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "storage_buckets")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucketKey_project(projectName, bucketName, keyName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("%s/default/%s/%s", projectName, bucketName, keyName),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    true,
				ImportState:                          true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 state, got %d", len(states))
					}
					state := states[0]
					if state.Attributes["access_key"] == "" {
						return fmt.Errorf("expected access_key to be set")
					}

					if state.Attributes["secret_key"] == "" {
						return fmt.Errorf("expected access_key to be set")
					}

					return nil
				},
			},
		},
	})
}

func testAccStorageBucketKey_basic(poolName string, bucketName string, keyName string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name   = "%s"
  driver = "dir"
}

resource "lxd_storage_bucket" "bucket1" {
  name = "%s"
  pool = lxd_storage_pool.pool1.name
}

resource "lxd_storage_bucket_key" "key1" {
  name   = "%s"
  pool   = lxd_storage_bucket.bucket1.pool
  bucket = lxd_storage_bucket.bucket1.name
}
	`, poolName, bucketName, keyName)
}

func testAccStorageBucketKey_role(bucketName string, keyName string, role string) string {
	return fmt.Sprintf(`
resource "lxd_storage_bucket" "bucket1" {
  name = "%s"
  pool = "default"
}

resource "lxd_storage_bucket_key" "key1" {
  name   = "%s"
  pool   = lxd_storage_bucket.bucket1.pool
  bucket = lxd_storage_bucket.bucket1.name
  role   = "%s"
}
 	`, bucketName, keyName, role)
}

func testAccStorageBucketKey_project(projectName string, bucketName string, keyName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"
  config = {
    "features.storage.volumes" = false
  }
}

resource "lxd_storage_bucket" "bucket1" {
  name    = "%s"
  pool    = "default"
  project = lxd_project.project1.name
}

resource "lxd_storage_bucket_key" "key1" {
  name    = "%s"
  project = lxd_storage_bucket.bucket1.project
  pool    = lxd_storage_bucket.bucket1.pool
  bucket  = lxd_storage_bucket.bucket1.name
}
	`, projectName, bucketName, keyName)
}
