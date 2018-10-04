# terraform-provider-lxd

Use Terraform to manage LXD resources.

## Description

This provider connects to the LXD daemon over local Unix socket or HTTPS.

It makes use of the [LXD client library](http://github.com/lxc/lxd), which
currently looks in `~/.config/lxc/` for `client.crt` and `client.key` files to
authenticate against the LXD daemon.

To generate these files and store them in the LXD client config, follow these
[steps](https://linuxcontainers.org/lxd/getting-started-cli/#multiple-hosts).
Alternatively, the LXD Terraform provider can generate them on demand if
`generate_client_certificates` is set to true.

## Resources

A list of supported resources can be found [here](resources).

## Basic Example

This is all that is needed if the LXD remotes have been defined out of band via
the `lxc` client.

```hcl
provider "lxd" {
}
```

## Specifying Multiple Remotes

If you're running `terraform` from a system where lxc is not installed then you
can define all the remotes in the Provider config:

```hcl
provider "lxd" {
  generate_client_certificates = true
  accept_remote_certificate    = true

  lxd_remote {
    name     = "lxd-server-1"
    scheme   = "https"
    address  = "10.1.1.8"
    password = "password"
    default  = true
  }

  lxd_remote {
    name     = "lxd-server-2"
    scheme   = "https"
    address  = "10.1.2.8"
    password = "password"
  }
}
```

## Configuration Reference

The following arguments are supported:

* `lxd_remote` - *Required* - Specifies an LXD remote (LXD server) to connect
	to. See the `lxd_remote` reference below for details.

* `config_dir` - *Optional* - The directory to look for existing LXD
	configuration. Defaults to `$HOME/.config/lxc`.

* `generate_client_certificates` - *Optional* - Automatically generate the LXD
	client certificate if it does not exist. Valid values are `true` and `false`.
	This can also be set with the `LXD_GENERATE_CLIENT_CERTS` Environment
	variable.

* `accept_remote_certificate` - *Optional* - Automatically accept the LXD
	remote's certificate. Valid values are `true` and `false`. If this is not set
	to `true`, you must accept the certificate out of band of Terraform. This can
	also be set with the `LXD_ACCEPT_SERVER_CERTIFICATE` environment variable.

* `refresh_interval` - *Optional* - How often to poll during state change.
	Defaults to "10s", or 10 seconds. Valid values are a Go-style parsable time
	duration (`10s`, `1m`, `5h`).

The `lxd_remote` block supports:

* `address` - *Optional* - The address of the LXD remote.

* `default` - *Optional* - Whether this should be the default remote. 
	This remote will then be used when one is not specified in a resource.
	Valid values are `true` and `false`.
	If you choose to _not_ set default=true on an `lxd_remote` and do not specify
	a remote in a resource, this provider will attempt to connect to an LXD
	server running on the same host through the UNIX socket. See `Undefined Remote`
	for more information.
	The default can also be set with the `LXD_REMOTE` Environment variable.

* `name` - *Optional* - The name of the LXD remote.

* `password` - *Optional* - The password to authenticate to the LXD remote.

* `port` - *Optional* - The port of the LXD remote.

* `scheme` - *Optional* Whether to connect to the LXD remote via `https` or
	`unix` (UNIX socket). Defaults to `unix`.

## Undefined Remote

If you choose to _not_ define an `lxd_remote`, this provider will attempt
to connect to an LXD server running on the same host through the UNIX
socket.

## Environment Variable Remote

It is possible to define a single `lxd_remote` through environment variables.
The required variables are:

* `LXD_REMOTE` - The name of the remote.
* `LXD_ADDR` - The address of the LXD remote.
* `LXD_PORT` - The port of the LXD remote.
* `LXD_PASSWORD` - The password of the LXD remote.
* `LXD_SCHEME` - The scheme to use (`unix` or `https`).

## PKI Support

LXD is capable of authenticating via PKI. In order to do this, you must
generate appropriate certificates on _both_ the remote/server side and client
side. Details on how to generate these certificates is out of scope of this
document.
