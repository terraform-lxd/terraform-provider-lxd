# terraform-provider-lxd

LXD Resource provider for Terraform

[![Build Status](https://travis-ci.org/sl1pm4t/terraform-provider-lxd.svg?branch=master)](https://travis-ci.org/sl1pm4t/terraform-provider-lxd)

## Prerequisites

* [Terraform](http://terraform.io)
* [LXD](https://linuxcontainers.org/lxd)

## Usage

This provider connects to the LXD daemon over local Unix socket or HTTPS.

It makes use of the [LXD client library](http://github.com/lxc/lxd), which currently looks in `~/.config/lxc/` for `client.crt` and `client.key` files that must exist to be able to communicate with the LXD daemon.

To generate these files and store them in the LXD daemon, follow these [steps](https://linuxcontainers.org/lxd/getting-started-cli/#multiple-hosts).

### Example Configurations

#### Provider (HTTPS)

```hcl
provider "lxd" {
  scheme                       = "https"
  address                      = "10.1.1.8"
  remote                       = "lxd-server"
  remote_password              = "password"
  generate_client_certificates = true
  accept_server_certificate    = true
}
```

#### Basic Example

This assumes the LXD server has been configured per the LXD documentation, including running `lxd init` to create a default network configuration.

This example also assumes an image called `ubuntu` has been cached locally on the LXD server. This can be done by running:

```shell
$ lxc image copy images:ubuntu/xenial/amd64 local: --alias=ubuntu
```

With those pieces in place, you can launch a basic container with:

```hcl
resource "lxd_container" "test1" {
  name      = "test1"
  image     = "ubuntu"
  ephemeral = false
}
```

You can also launch a container directly from a remote image, not locally cached, by referencing the remote image name using the format `[remote:]<image_alias|image_hash>`

```hcl
resource "lxd_container" "test1" {
  name      = "test1"
  image     = "images:ubuntu/xenial/amd64"
  ephemeral = false
}
```

 > NOTE:
 > Currently only the following remotes are supported:
 > * remote named defined in LXD provider (same as omitting `<remote>:` prefix)
 > * `images`
 > * `ubuntu`
 > * `ubuntu-daily`
 > See the LXD [documentation](https://linuxcontainers.org/lxd/getting-started-cli/#using-the-built-in-image-remotes) for more info on default image remotes.

#### Container Configuration & Devices

A container can also take a number of configuration and device options. A full reference can be found [here](https://github.com/lxc/lxd/blob/master/doc/configuration.md). For example, to create a container with 2 CPUs and to share the `/tmp` directory with the LXD host:

```hcl
resource "lxd_container" "test1" {
  name      = "test1"
  image     = "ubuntu"
  ephemeral = false

  config {
    limits.cpu = 2
  }

  device {
    name = "shared"
    type = "disk"

    properties {
      source = "/tmp"
      path   = "/tmp"
    }
  }
}
```

#### Profiles

Profiles can be used to share common configurations between containers. Profiles accept the same configuration and device options that containers can use.

The order which profiles are specified is important. LXD applies profiles from "left to right", so profile options may be overridden by other profiles.

```hcl
resource "lxd_profile" "profile1" {
  name = "profile1"

  config {
    limits.cpu = 2
  }

  device {
    name = "shared"
    type = "disk"

    properties {
      source = "/tmp"
      path   = "/tmp"
    }
  }
}

resource "lxd_container" "test1" {
  name      = "test1"
  image     = "ubuntu"
  ephemeral = false
  profiles  = ["default", "${lxd_profile.profile1.name}"]
}
```

#### Networks

If you're using LXD 2.3 or later, you can create networks with the `lxd_network` resource. See [this](https://www.stgraber.org/2016/10/27/network-management-with-lxd-2-3/) blog post for details about LXD networking and the [configuration reference](https://github.com/lxc/lxd/blob/master/doc/configuration.md) for all network details.

This example creates a standard NAT network similar to what `lxd init` creates. Containers will access this network via their `eth0` interface:

```hcl
resource "lxd_network" "new_default" {
  name = "new_default"

  config {
    ipv4.address = "10.150.19.1/24"
    ipv4.nat     = "true"
    ipv6.address = "fd42:474b:622d:259d::1/64"
    ipv6.nat     = "true"
  }
}

resource "lxd_profile" "profile1" {
  name = "profile1"

  device {
    name = "eth0"
    type = "nic"

    properties {
      nictype = "bridged"
      parent  = "${lxd_network.new_default.name}"
    }
  }
}

resource "lxd_container" "test1" {
  name      = "test1"
  image     = "ubuntu"
  ephemeral = false
  profiles  = ["${lxd_profile.profile1.name}"]
}
```

This example creates a second internal network that containers will access via `eth1`. Containers will use the `default` profile to gain access to the default network on `eth0`.

```hcl
resource "lxd_network" "internal" {
  name = "internal"

  config {
    ipv4.address = "192.168.255.1/24"
  }
}

resource "lxd_profile" "profile1" {
  name = "profile1"

  device {
    name = "eth1"
    type = "nic"

    properties {
      nictype = "bridged"
      parent  = "${lxd_network.internal.name}"
    }
  }
}

resource "lxd_container" "test1" {
  name      = "test1"
  image     = "ubuntu"
  ephemeral = false
  profiles  = ["default", "${lxd_profile.profile1.name}"]

	provisioner "local-exec" {
		command = "lxc exec local:${self.name} dhclient eth1"
	}
}
```

Finally, LXD networks can be used to create tunnels to other LXD servers. In order to create a tunnel, designate one LXD server as the tunnel "server". This server will offer DHCP leases to the tunnel "client". For example:

```hcl
resource "lxd_network" "vxtun" {
  name = "vxtun"

  config {
    tunnel.vxtun.protocol = "vxlan"
    tunnel.vxtun.id       = 9999
    tunnel.vxtun.local    = "10.1.1.1"
    tunnel.vxtun.remote   = "10.255.1.1"
    ipv4.address          = "192.168.255.1/24"
    ipv6.address          = "none"
  }
}
```

For the tunnel client:

```hcl
resource "lxd_network" "vxtun" {
  name = "vxtun"

  config {
    tunnel.vxtun.protocol = "vxlan"
    tunnel.vxtun.id       = 9999
    tunnel.vxtun.local    = "10.255.1.1"
    tunnel.vxtun.remote   = "10.1.1.1"
    ipv4.address          = "none"
    ipv6.address          = "none"
  }
}
```

Note how the `local` and `remote` addresses are swapped between the two. Also note how the client does not provide an IP address range.

With these resources in place, attach them to a profile in the exact same way described in the other examples.

_note_: `local` and `remote` accept both IPv4 and IPv6 addresses.

#### Storage Pools

To create and manage Storage Pools, use the `lxd_storage_pool` resource:

```hcl
resource "lxd_storage_pool" "pool1" {
  name = "mypool"
  driver = "dir"
  config {
    source = "/var/lib/lxd/storage-pools/mypool"
  }
}
```

#### Volumes

Volumes are storage devices allocated from a Storage Pool. The `lxd_volume` resource can
create and manage storage volumes:

```hcl
resource "lxd_storage_pool" "pool1" {
  name = "mypool"
  driver = "dir"
  config {
    source = "/var/lib/lxd/storage-pools/mypool"
  }
}

resource "lxd_volume" "volume1" {
  name = "myvolume"
  pool = "${lxd_storage_pool.pool1.name}"
}
```

_note_: Technically, an LXD volume is simply a container or profile device of type "disk".

#### Attaching Volumes

Volumes can be attached to containers by using the `lxd_volume_container_attach` resource:

```hcl
resource "lxd_storage_pool" "pool1" {
  name = "mypool"
  driver = "dir"
  config {
    source = "/var/lib/lxd/storage-pools/mypool"
  }
}

resource "lxd_volume" "volume1" {
  name = "myvolume"
  pool = "${lxd_storage_pool.pool1.name}"
}

resource "lxd_container" "container1" {
  name = "%s"
  image = "ubuntu"
  profiles = ["default"]
}

resource "lxd_volume_container_attach" "attach1" {
  pool = "${lxd_storage_pool.pool1.name}"
  volume_name = "${lxd_volume.volume1.name}"
  container_name = "${lxd_container.container1.name}"
  path = "/tmp"
}
```

#### Snapshots

The `lxd_snapshot` resource can be used to create a point in time snapshot of a container.

```hcl
resource "lxd_snapshot" "snap1" {
  container_name = "${lxd_container.container1.name}"
  name = "snap1"
}
```

## Reference

### Provider

#### lxd

##### Parameters

  * `address`  - *Optional* - Unix socket file path or IP / FQDN where LXD daemon can be reached. Defaults to `/var/lib/lxd/unix.socket`
  * `scheme`   - *Optional* - `https` or `unix`. Defaults to `unix`.
  * `port`     - *Optional* - `https` scheme only - The port on which the LXD daemon is listening. Defaults to 8443.
  * `remote`   - *Optional* - Name of the remote LXD as it exists in the local lxc config. Defaults to `local`.
  * `remote_password` - *Optional* - Password of the remote LXD server.
  * `config_dir` - *Optional* - Directory path to client LXD configuration and certs. Defaults to `$HOME/.config/lxc`.
  * `generate_client_certificates` - *Optional* - Generate the LXC client's certificates if they don't exist. This can also be done out-of-band of Terraform with the lxc command-line client.
  * `accept_remote_certificate` - *Optional* - Accept the remote LXD server certificate. This can also be done out-of-band of Terraform with the lxc command-line client.

### Resources

The following resources are currently available:

  * `lxd_container` - Creates and manages a Container
  * `lxd_network` - Creates and manages a Network
  * `lxd_profile` - Creates and manages a Profile

#### lxd_container

##### Parameters

  * `name`      - *Required* - Name of the container.
  * `image`     - *Required* - Base image from which the container will be created.
  * `profiles`  - *Optional* - Array of LXD config profiles to apply to the new container.
  * `ephemeral` - *Optional* - Boolean indicating if this container is ephemeral. Default = false.
  * `privileged`- *Optional* - Boolean indicating if this container will run in privileged mode. Default = false.
  * `config`    - *Optional* - Map of key/value pairs of [container config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#container-configuration).
  * `device`    - *Optional* - Device definition. See reference below.
  * `file`      - *Optional* - File to upload to the container. See reference below.

##### Device Block

  * `name`      - *Required* - Name of the device.
  * `type`      - *Required* - Type of the device Must be one of none, disk, nic, unix-char, unix-block, usb, gpu.
  * `properties`- *Required* - Map of key/value pairs of [device properties](https://github.com/lxc/lxd/blob/master/doc/configuration.md#devices-configuration).

##### File Block

  * `content` - *Required* - The _contents_ of the file. Use the `file()` function to read in the content of a file from disk.
  * `target_file` - *Required* - The absolute path of the file on the container, including the filename.
  * `uid` - *Optional* - The UID of the file. Must be an unquoted integer.
  * `gid` - *Optional* - The GID of the file. Must be an unquoted integer.
  * `mode` - *Optional* - The octal permissions of the file, must be quoted.
  * `create_directories` - *Optional* - Whether to create the directories leading to the target if they do not exist.

#### lxd_network

##### Parameters

  * `name`      - *Required* - Name of the network. This is usually the device the network will appear as to containers.
  * `config`    - *Optional* - Map of key/value pairs of [network config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#network-configuration).

##### Exported Attributes

  * `type`      - The type of network. This will be either bridged or physical.
  * `managed`   - Whether or not the network is managed.

#### lxd_profile

##### Parameters

  * `name`      - *Required* - Name of the container.
  * `config`    - *Optional* - Map of key/value pairs of [container config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#container-configuration).
  * `device`    - *Optional* - Device definition. See reference below.

##### Device Block

  * `name`      - *Required* - Name of the device.
  * `type`      - *Required* - Type of the device Must be one of none, disk, nic, unix-char, unix-block, usb, gpu.
  * `properties`- *Required* - Map of key/value pairs of [device properties](https://github.com/lxc/lxd/blob/master/doc/configuration.md#devices-configuration).

#### lxd_storage_pool

##### Parameters

  * `name`   - *Required* - Name of the storage pool.
  * `driver` - *Required* - Storage Pool driver. Must be one of `dir`, `lvm`, `btrfs`, or `zfs`.
  * `config` - *Required* - Map of key/value pairs of [storage pool config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#storage-pool-configuration). Config settings vary from driver to driver.

#### lxd_volume

##### Parameters

  * `name`   - *Required* - Name of the storage pool.
  * `pool`   - *Required* - The Storage Pool to host the volume.
  * `type`   - *Optional* - The "type" of volume. The default value is `custom`, which is the type to use for storage volumes attached to containers.
  * `config` - *Required* - Map of key/value pairs of [volume config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#storage-volume-configuration). Config settings vary depending on the Storage Pool used.

#### lxd_volume_container_attach

##### Parameters

  * `pool`           - *Required* - Name of the volume's storage pool.
  * `volume_name`    - *Required* - Name of the volume to attach.
  * `container_name` - *Required* - Name of the container to attach the volume to.
  * `path`           - *Required* - Mountpoint of the volume in the container.
  * `device_name`    - *Optional* - The volume device name as seen by the container. By default, this will be the volume name.

#### lxd_snapshot

##### Parameters

  * `name` - *Required* - Name of the snapshot.
  * `container_name` - *Required* - The name of the container to snapshot.
  * `stateful` - *Optional* - Set to `true` to create a stateful snapshot, `false` for stateless. Stateful snapshots include runtime state. Default = false

##### Exported Parameters

  * `creation_date` - The time LXD reported the snapshot was successfully created, in UTC.

## Known Limitations

All the base LXD images do not include an SSH server, therefore terraform will be unable to execute any `provisioners`.
A basic base image must be prepared in advance, that includes the SSH server.

## To Do

- [x] Support for using client cert / key from other paths
- [ ] Ability to update container config
- [ ] Ability to exec commands via LXD WebSocket channel
- [x] Ability to upload files
- [x] Volumes support
- [ ] Add LXD `image` resource
- [ ] Remote image datasource

## Contributors

Some recognition for great contributors to this project:

  * [jtopjian](https://github.com/jtopjian)
  * [yobert](https://github.com/yobert)
