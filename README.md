# terraform-provider-lxd

LXD Resource provider for Terraform

[![Build Status](https://travis-ci.org/terraform-lxd/terraform-provider-lxd.svg?branch=master)](https://travis-ci.org/terraform-lxd/terraform-provider-lxd)

## Prerequisites

* [Terraform](http://terraform.io)
* [LXD](https://linuxcontainers.org/lxd)

## Installation

### Using pre-built binary

1. Download the binary from the project [releases page](https://github.com/terraform-lxd/terraform-provider-lxd/releases/latest)
2. Extract provider binary from tar file.
3. Copy to `$PATH` or the `~/.terraform.d/plugins` directory so Terraform can find it.

**Example**

```bash
# List latest binaries:
curl -s https://api.github.com/repos/terraform-lxd/terraform-provider-lxd/releases/latest | jq '.assets | .[] | .browser_download_url'
  "https://github.com/terraform-lxd/terraform-provider-lxd/releases/download/v1.1.0/terraform-provider-lxd_v1.1.0_darwin_amd64.zip"
  "https://github.com/terraform-lxd/terraform-provider-lxd/releases/download/v1.1.0/terraform-provider-lxd_v1.1.0_linux_amd64.zip"
  "https://github.com/terraform-lxd/terraform-provider-lxd/releases/download/v1.1.0/terraform-provider-lxd_v1.1.0_windows_amd64.zip"

# Retrieve zip
wget https://github.com/terraform-lxd/terraform-provider-lxd/releases/download/v1.1.0/terraform-provider-lxd_v1.1.0_linux_amd64.zip

# Unzip
unzip terraform-provider-lxd_*.zip

# Copy binary to a location where Terraform will find it
mkdir -p ~/.terraform.d/plugins
mv terraform-provider-lxd ~/.terraform.d/plugins
```

### Building from source

1. Follow these [instructions](https://golang.org/doc/install) to setup a Golang development environment.
2. Use `go get` to pull down this repository and compile the binary:

```
go get -v -u github.com/terraform-lxd/terraform-provider-lxd
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
