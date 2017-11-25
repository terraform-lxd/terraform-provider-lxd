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

## Documentation

Full documentation can be found in the [`docs`](docs) directory.

## Known Limitations

Many of the base LXD images don't include an SSH server, therefore terraform
will be unable to execute any `provisioners`. Either use the base ubuntu images
from the `ubuntu` or `ubuntu-daily` or manually prepare a base image that
includes SSH.

## Contributors

Some recognition for great contributors to this project:

  * [jtopjian](https://github.com/jtopjian)
  * [mjrider](https://github.com/mjrider)
  * [yobert](https://github.com/yobert)
