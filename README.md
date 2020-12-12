# terraform-provider-lxd

LXD Resource provider for Terraform

## Prerequisites

* [Terraform](http://terraform.io)
* [LXD](https://linuxcontainers.org/lxd)

## Installation

This provider is published in the [Terraform Registry](https://registry.terraform.io/providers/terraform-lxd/lxd/).

Follow the official instructions for declaring providers in your Terraform configuration
[here](https://www.terraform.io/docs/configuration/provider-requirements.html).

### Quick Example

Add the following to your Terraform configuration:

```hcl
terraform {
  required_providers {
    lxd = {
      source = "terraform-lxd/lxd"
    }
  }
}
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

  * [jgraichen](https://github.com/jgraichen)
  * [jtopjian](https://github.com/jtopjian)
  * [mjrider](https://github.com/mjrider)
  * [sl1pm4t](https://github.com/sl1pm4t)
  * [yobert](https://github.com/yobert)
