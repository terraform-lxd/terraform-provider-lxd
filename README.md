# terraform-provider-lxd

LXD Resource provider for Terraform

## Prerequisites

* [Terraform](http://terraform.io)
* [LXD](https://ubuntu.com/lxd)

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

### Using the terraform-provider-lxd from source

1. Follow these [instructions](https://golang.org/doc/install) to setup a Golang development environment.
2. Checkout the repository `git clone ...`
3. Compile from sources to a development binary:

```shell
cd terraform-provider-lxd
go build -v
```

4. Configure Terraform (`~/.terraformrc`) to use the development binary provider:

```shell
$ cat ~/.terraformrc
provider_installation {
  dev_overrides {
    "terraform-lxd/lxd" = "/home/<REPLACE_ME>/git/terraform-provider-lxd"
  }
}
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
