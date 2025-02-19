# lxd_storage_pool

Provides information about an existing LXD storage pool.

## Example Usage

```hcl
data "lxd_storage_pool" "pool" {
  name = "my-pool"
}
```

## Argument Reference

* `name` - **Required** - Name of the storage pool.

* `project` - *Optional* - Name of the project where storage pool is located.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote is used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `driver` - Storage pool driver.

* `description` - Description of the storage pool.

* `config` - Map of key/value pairs of
	[storage pool config settings](https://documentation.ubuntu.com/lxd/en/latest/reference/storage_drivers/).
	Config settings vary from driver to driver.

* `locations` - List of cluster members where storage pool is located.

* `status` - The status of the storage pool.

