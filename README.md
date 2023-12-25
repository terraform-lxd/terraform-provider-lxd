# terraform-provider-lxd

LXD Resource provider for Terraform

## Prerequisites

- [Terraform](http://terraform.io)
- [LXD](https://ubuntu.com/lxd)

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

### Development

#### Setup

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
  # Use local git clone of LXD provider
  dev_overrides {
    "terraform-lxd/lxd" = "/home/<REPLACE_ME>/git/terraform-provider-lxd"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

#### Testing

There are two test suites, unit and acceptance. By default the acceptance tests are not run as they require a functional
LXD environment.

##### Unit tests

```shell
make
```

##### Acceptance tests

```shell
make acc
# or run an individual test
TESTARGS="-run TestAccCachedImage_basicVM" make testacc
# increase test verbosity. options are trace, debug, info, warn, or error (default)
TF_LOG=info make testacc
```

## Documentation

Full documentation can be found in the [`docs`](docs) directory.

## Contributors

Some recognition for great contributors to this project:

- [jgraichen](https://github.com/jgraichen)
- [jtopjian](https://github.com/jtopjian)
- [mjrider](https://github.com/mjrider)
- [sl1pm4t](https://github.com/sl1pm4t)
- [yobert](https://github.com/yobert)
