# incus_profile

Manages an Incus profile.

## Example Usage

```hcl
resource "incus_profile" "profile1" {
  name = "profile1"

  config = {
    "limits.cpu" = 2
  }

  device {
    name = "shared"
    type = "disk"

    properties = {
      source = "/tmp"
      path   = "/tmp"
    }
  }

  device {
    type = "disk"
    name = "root"

    properties = {
      pool = "default"
      path = "/"
    }
  }
}

resource "incus_instance" "test1" {
  name      = "test1"
  image     = "ubuntu"
  ephemeral = false
  profiles  = [incus_profile.profile1.name]
}
```

## Argument Reference

* `name` - **Required** - Name of the profile.

* `device` - *Optional* - Device definition. See reference below.

* `config` - *Optional* - Map of key/value pairs of
	[instance config settings](https://linuxcontainers.org/incus/docs/main/reference/instance_options/).

* `project` - *Optional* - Name of the project where the profile will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

The `device` block supports:

* `name` - **Required** - Name of the device.

* `type` - **Required** - Type of the device Must be one of none, disk, nic,
	unix-char, unix-block, usb, gpu, infiniband, proxy, unix-hotplug, tpm, pci.

* `properties`- **Required** - Map of key/value pairs of
	[device properties](https://linuxcontainers.org/incus/docs/main/reference/devices/).

## Attribute Reference

No attributes are exported.

## Importing

Profiles can be imported with the following command:

```shell
$ terraform import incus_profile.my_profile [<remote>:][<project>/]<profile_name>
```

## Importing

Import ID syntax: `[<remote>:][<project>/]<name>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<name>` - **Required** - Profile name.

### Import example

Example using terraform import command:

```shell
$ terraform import incus_profile.myprofile proj/profile1
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "incus_profile" "myprofile" {
  name    = "profile1"
  project = "proj"
}

import {
  to = incus_profile.myprofile
  id = "proj/profile1"
}
```

## Notes

* The order in which profiles are specified is important. Incus applies profiles
	from left to right. Profile options may be overridden by other profiles.
