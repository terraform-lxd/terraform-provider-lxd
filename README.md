# terraform-provider-lxd

LXD Resource provider for Terraform

[![Build Status](https://travis-ci.org/sl1pm4t/terraform-provider-lxd.svg?branch=master)](https://travis-ci.org/sl1pm4t/terraform-provider-lxd)

## Prerequisites

* [Terraform](http://terraform.io)
* [LXD](https://linuxcontainers.org/lxd)

## Installation

### Using pre-built binary

1. Download the binary from the project [releases page](https://github.com/sl1pm4t/terraform-provider-lxd/releases)
2. Extract provider binary from tar file.
3. Copy to `$PATH` or the `~/.terraform` directory so Terraform can find it.

**Example**
```bash
wget https://github.com/sl1pm4t/terraform-provider-lxd/releases/download/v0.10.0-beta2/terraform-provider-lxd_v0.10.0-beta2_linux_amd64.tar.gz

tar -xzvf terraform-provider-lxd_*.tar.gz

mkdir -p ~/.terraform/
mv terraform-provider-lxd ~/.terraform/
```

### Building from source

1. Follow these [instructions](https://golang.org/doc/install) to setup a Golang development environment.
2. Use `go get` to pull down this repository and compile the binary:
```
go get -v -u github.com/sl1pm4t/terraform-provider-lxd
```

## Usage

This provider connects to the LXD daemon over local Unix socket or HTTPS.

It makes use of the [LXD client library](http://github.com/lxc/lxd), which currently looks in `~/.config/lxc/` for `client.crt` and `client.key` files to authenticate against the LXD daemon.

To generate these files and store them in the LXD client config, follow these [steps](https://linuxcontainers.org/lxd/getting-started-cli/#multiple-hosts).
Alternatively, the LXD Terraform provider can generate them on demand if `generate_client_certificates` is set to true.

### Example Configurations

#### Provider (Use LXC Config)

This is all that is needed if the LXD remotes have been defined out of band via the `lxc` client.

```hcl
provider "lxd" {
}
```

#### Provider (Custom Remotes)

If you're running `terraform` from a system where lxc is not installed then you can define all the remotes in the Provider config:

```hcl
provider "lxd" {
  generate_client_certificates = true
  accept_remote_certificate    = true

  lxd_remote {
    name     = "lxd-server-1"
    scheme   = "https"
    address  = "10.1.1.8"
    password = "password"
  }

  lxd_remote {
    name     = "lxd-server-2"
    scheme   = "https"
    address  = "10.1.2.8"
    password = "password"
  }
}
```


#### Basic Example

This assumes the LXD server has been configured per the LXD documentation, including running `lxd init` to create a default network configuration.

With the LXD daemon installed and initialized, you can launch a basic container from a remote image with:

```hcl
resource "lxd_container" "test1" {
  name      = "test1"
  image     = "ubuntu:x"
  ephemeral = false
}
```

To cache an image locally before launching a container use the `lxd_cached_image` resource.
This ensures the same identical image is available when launching multiple containers.

```hcl
resource "lxd_cached_image" "xenial" {
  source_remote = "ubuntu"
  source_image  = "x"
}

resource "lxd_container" "test1" {
  name      = "test1"
  image     = "${lxd_cached_image.xenial.fingerprint}"
  ephemeral = false
}
```

 > NOTE:
 > See the LXD [documentation](https://linuxcontainers.org/lxd/getting-started-cli/#using-the-built-in-image-remotes) for more info on default image remotes.

#### Container Configuration & Devices

A container can also take a number of configuration and device options. A full reference can be found [here](https://github.com/lxc/lxd/blob/master/doc/configuration.md). For example, to create an autostart container with 2 CPUs and to share the `/tmp` directory with the LXD host:

```hcl
resource "lxd_container" "test1" {
  name      = "test1"
  image     = "ubuntu"
  ephemeral = false

  config {
    boot.autostart = true
  }

  limits {
    cpu = 2
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

Note, the `config` attributes cannot be changed without destroying and re-creating the container, however values in `limits` can be changed on the fly.

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

  * `config_dir`                   - *Optional* - Directory path to client LXD configuration and certs. Defaults to `$HOME/.config/lxc`.
  * `generate_client_certificates` - *Optional* - Generate the LXC client's certificates if they don't exist.
                                     This can also be done out-of-band of Terraform with the lxc command-line client.
  * `accept_remote_certificate`    - *Optional* - Accept the remote LXD server certificate.
                                     This can also be done out-of-band of Terraform with the lxc command-line client.
  * `refresh_interval`             - *Optional* - How often to poll during state changes. Defaults to `10s`.

The `lxd_remote` block supports:

  * `address`  - *Required* - The IP address or hostname of the remote.
  * `name`     - *Required* - The name of the LXD remote, that can be referenced in resource `remote` attributes.
  * `default`  - *Optional* - `true` if this is this the default remote. Default = `false`.
  * `port`     - *Optional* - The port on which the LXD daemon is listening. Default = `8443`.
  * `password` - *Optional* - The trust password configured on the LXD server.
  * `scheme`   - *Optional* - `https` or `unix`. Default = `https`.

The LXD Provider can also be configured via environment variables.

  * `LXD_REMOTE`   - Equivalent to `lxd_remote.name`
  * `LXD_ADDR`     - Equivalent to `lxd_remote.address`
  * `LXD_PORT`     - Equivalent to `lxd_remote.port`
  * `LXD_PASSWORD` - Equivalent to `lxd_remote.password`
  * `LXD_SCHEME`   - Equivalent to `lxd_remote.scheme`

##### The Default remote

> The provider uses the following order of preference to determine the *Default* LXD remote:
>  1. the `lxd_remote` block with attribute `default` set to true
>  2. the remote defined via environment variables (`LXD_REMOTE`, `LXD_ADDR` etc.)
>  3. the remote set in `lxc` config file that is marked as default

### Resources

The following resources are currently available:

  * `lxd_cached_image` - Create and manage a copy of a remote image
  * `lxd_container` - Creates and manages a Container
  * `lxd_network` - Creates and manages a Network
  * `lxd_profile` - Creates and manages a Profile
  * `lxd_snapshot` - Create and manage point in time snapshots of containers and optionally their runtime state
  * `lxd_storage_pool` - Create and manage storage pools
  * `lxd_volume` - Create and manage storage volumes
  * `lxd_volume_container_attach` - Manage attaching `lxd_volume` to `lxd_container`

#### lxd_cached_image

##### Parameters

  * `source_remote` - *Required* - Name of the LXD remote from where image will be pulled.
  * `source_image`  - *Required* - Fingerprint or alias of image to pull.
  * `aliases`       - *Optional* - A list of aliases to assign to the image after pulling.
  * `copy_aliases`  - *Optional* - True to copy the aliases of the image from the remote. Default = false.
  * `remote`        - *Optional* - The remote in which the resource will be created. If it
                                   is not provided, the default provider remote is used.

##### Exported Parameters

  * `architecture`   - The image architecture (e.g. amd64, i386).
  * `created_at`     - The datetime of image creation, in Unix time.
  * `fingerprint`    - The unique hash fingperint of the image.
  * `copied_aliases` - The list of aliases that were copied from the `source_image`.

#### lxd_container

##### Parameters

  * `name`      - *Required* - Name of the container.
  * `image`     - *Required* - Base image from which the container will be created.
  * `profiles`  - *Optional* - Array of LXD config profiles to apply to the new container.
  * `ephemeral` - *Optional* - Boolean indicating if this container is ephemeral. Default = false.
  * `privileged`- *Deprecated* - Boolean indicating if this container will run in privileged mode. Default = false. This argument is deprecated. Use a config setting of `security.privileged=1` instead.
  * `config`    - *Optional* - Map of key/value pairs of [container config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#container-configuration).
  * `limits`    - *Optional* - Map of key/value pairs that define the [container resources limits](https://github.com/lxc/lxd/blob/master/doc/containers.md).
  * `device`    - *Optional* - Device definition. See reference below.
  * `file`      - *Optional* - File to upload to the container. See reference below.
  * `remote`    - *Optional* - The remote in which the resource will be created. If it
                               is not provided, the default provider remote is used.

##### Importing
Importing existing containers is possible with the following ID syntax  
`terraform import lxd_container.container [remote:]name[/images:alpine/3.5]`.  

 * remote             - *Optional* - is the name of the remote in the provider config
 * name               - *Required* - is the container name
 * /images:alpine/3.5 - *Optional* - translates to image = images:alpine/3.5 in the resource configuration

##### Container Network Access

If your container has multiple network interfaces, you can specify which
one Terraform should report the IP address of. To specify the interface, do
the following:

```hcl
resource "lxd_container" "container1" {
  name = "container1"
  image = "images:alpine/3.5/amd64"
  profiles = ["default"]

  config {
    user.access_interface = "eth0"
  }
}
```

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
  * `remote`    - *Optional* - The remote in which the resource will be created. If it
                               is not provided, the default provider remote is used.

##### Exported Attributes

  * `type`      - The type of network. This will be either bridged or physical.
  * `managed`   - Whether or not the network is managed.

#### lxd_profile

##### Parameters

  * `name`      - *Required* - Name of the container.
  * `config`    - *Optional* - Map of key/value pairs of [container config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#container-configuration).
  * `device`    - *Optional* - Device definition. See reference below.
  * `remote`    - *Optional* - The remote in which the resource will be created. If it
                               is not provided, the default provider remote is used.

##### Device Block

  * `name`      - *Required* - Name of the device.
  * `type`      - *Required* - Type of the device Must be one of none, disk, nic, unix-char, unix-block, usb, gpu.
  * `properties`- *Required* - Map of key/value pairs of [device properties](https://github.com/lxc/lxd/blob/master/doc/configuration.md#devices-configuration).

#### lxd_storage_pool

##### Parameters

  * `name`   - *Required* - Name of the storage pool.
  * `driver` - *Required* - Storage Pool driver. Must be one of `dir`, `lvm`, `btrfs`, or `zfs`.
  * `config` - *Required* - Map of key/value pairs of [storage pool config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#storage-pool-configuration). Config settings vary from driver to driver.
  * `remote` - *Optional* - The remote in which the resource will be created. If it
                            is not provided, the default provider remote is used.

#### lxd_volume

##### Parameters

  * `name`   - *Required* - Name of the storage pool.
  * `pool`   - *Required* - The Storage Pool to host the volume.
  * `type`   - *Optional* - The "type" of volume. The default value is `custom`, which is the type to use for storage volumes attached to containers.
  * `config` - *Required* - Map of key/value pairs of [volume config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#storage-volume-configuration). Config settings vary depending on the Storage Pool used.
  * `remote` - *Optional* - The remote in which the resource will be created. If it
                            is not provided, the default provider remote is used.

#### lxd_volume_container_attach

##### Parameters

  * `pool`           - *Required* - Name of the volume's storage pool.
  * `volume_name`    - *Required* - Name of the volume to attach.
  * `container_name` - *Required* - Name of the container to attach the volume to.
  * `path`           - *Required* - Mountpoint of the volume in the container.
  * `device_name`    - *Optional* - The volume device name as seen by the container. By default, this will be the volume name.
  * `remote`         - *Optional* - The remote in which the resource will be created. If it
                                    is not provided, the default provider remote is used.

#### lxd_snapshot

##### Parameters

  * `name` - *Required* - Name of the snapshot.
  * `container_name` - *Required* - The name of the container to snapshot.
  * `stateful` - *Optional* - Set to `true` to create a stateful snapshot, `false` for stateless. Stateful snapshots include runtime state. Default = false
  * `remote`   - *Optional* - The remote in which the resource will be created. If it
                              is not provided, the default provider remote is used.

##### Exported Parameters

  * `creation_date` - The time LXD reported the snapshot was successfully created, in UTC.

## Known Limitations

Many of the base LXD images don't include an SSH server, therefore terraform will be unable to execute any `provisioners`.
Either use the base ubuntu images from the `ubuntu` or `ubuntu-daily` or manually prepare a base image that includes SSH.

## Contributors

Some recognition for great contributors to this project:

  * [jtopjian](https://github.com/jtopjian)
  * [yobert](https://github.com/yobert)
