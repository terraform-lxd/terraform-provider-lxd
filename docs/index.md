# terraform-provider-lxd

Use Terraform to manage LXD resources.

## Description

This provider connects to the LXD daemon over local Unix socket or HTTPS.

It makes use of the [LXD client library](https://github.com/canonical/lxd), which
currently looks in `~/snap/lxd/common/config` (and `~/.config/lxc`) for `client.crt`
and `client.key` files to authenticate against the LXD daemon.

To generate these files and store them in the LXD client config, follow these
[steps](https://documentation.ubuntu.com/lxd/en/latest/howto/server_expose/#server-authenticate).
Alternatively, the LXD Terraform provider can generate them on demand if
`generate_client_certificates` is set to true.

Minimum required LXD version is **`4.0`**.

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

  remote {
    name     = "lxd-server-1"
    address  = "https://10.1.1.8:8443"
    password = "password"
    default  = true
  }

  remote {
    name    = "lxd-server-2"
    address = "https://10.1.2.8"
    token   = "token"
  }
}
```

## Configuration Reference

The following arguments are supported:

* `remote` - *Optional* - Specifies an LXD remote (LXD server) to connect
	to. See the `remote` reference below for details.

* `config_dir` - *Optional* - The directory to look for existing LXD
	configuration. Defaults to `$HOME/snap/lxd/common/config` (and fallbacks to `$HOME/.config/lxc`).

* `generate_client_certificates` - *Optional* - Automatically generate the LXD
	client certificate if it does not exist. Defaults to `false`.

* `accept_remote_certificate` - *Optional* - Automatically accept the LXD
	remote's certificate during initial authentication. If this is not set
	to `true`, you must accept the certificate out of band of Terraform or
  use a trust token instead (recommended, see `token` in `remote`).
  Defaults to `false`.

The `remote` block supports:

* `name` - *Optional* - The name of the remote.

* `protocol` - *Optional* - The protocol of remote server (`lxd` or `simplestreams`).

* `address` - *Optional* - The remote address in format `[<scheme>://]<host>[:<port>]`.
  Scheme can be set to either `unix` or `https`. If scheme is not set, it will default to `unix` if first character is `/`, otherwise to `https`.
  Port can be set only for remote HTTPS servers. Port value defaults to `8443` for `lxd` protocol, and to `443` for `simplestreams` protocol.

* `default` - *Optional* - Whether this should be the default remote.
	This remote will then be used when one is not specified in a resource.
	If you choose to _not_ set default=true on a `remote` and do not specify
	a remote in a resource, this provider will attempt to connect to an LXD
	server running on the same host through the UNIX socket. See `Undefined Remote`
	for more information.
	The default can also be set with the `LXD_REMOTE` Environment variable.

* `password` - *Optional* - The [trust password](https://documentation.ubuntu.com/lxd/en/latest/authentication/#adding-client-certificates-using-a-trust-password)
  used for initial authentication with the LXD remote. This method is **not recommended** and has
  been removed in LXD 6.1. Please, use `token` instead.

* `token` - *Optional* - The one-time trust [token](https://documentation.ubuntu.com/lxd/en/latest/authentication/#adding-client-certificates-using-tokens)
  used for initial authentication with the LXD remote.

## Undefined Remote

If you choose to _not_ define a `remote`, this provider will attempt
to connect to an LXD server running on the same host through the UNIX
socket.

## Environment Variable Remote

It is possible to define a single `remote` through environment variables.
The supported variables are:
* `LXD_REMOTE` - The name of the remote.
* `LXD_ADDR` - The address of the LXD remote.
* `LXD_PASSWORD` - The password of the LXD remote.
* `LXD_TOKEN` - The trust token of the LXD remote.
* `LXD_GENERATE_CLIENT_CERTS` - Automatically generate the LXD client certificate if missing.
* `LXD_ACCEPT_SERVER_CERTIFICATE` - Automatically accept the LXD remote's certificate during initial authentication.

## PKI Support

LXD is capable of authenticating via PKI. In order to do this, you must
generate appropriate certificates on _both_ the remote/server side and client
side. Details on how to generate these certificates is out of scope of this
document.
