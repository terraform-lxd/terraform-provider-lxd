# terraform-provider-incus

Use Terraform to manage Incus resources.

## Description

This provider connects to the Incus daemon over local Unix socket or HTTPS.

It makes use of the [Incus client library](https://github.com/lxc/incus), which
currently looks in `~/.config/incus` for `client.crt`
and `client.key` files to authenticate against the Incus daemon.

To generate these files and store them in the Incus client config, follow these
[steps](https://linuxcontainers.org/incus/docs/main/howto/server_expose/#authenticate-with-the-incus-server).
Alternatively, the Incus Terraform provider can generate them on demand if
`generate_client_certificates` is set to true.

Minimum required Incus version is **`0.3.0`**.

## Basic Example

This is all that is needed if the Incus remotes have been defined out of band via
the `incus` client.

```hcl
provider "incus" {
}
```

## Specifying Multiple Remotes

If you're running `terraform` from a system where Incus is not installed then you
can define all the remotes in the Provider config:

```hcl
provider "incus" {
  generate_client_certificates = true
  accept_remote_certificate    = true

  remote {
    name    = "incus-server-1"
    scheme  = "https"
    address = "10.1.1.8"
    token   = "token"
    default = true
  }

  remote {
    name    = "incus-server-2"
    scheme  = "https"
    address = "10.1.2.8"
    token   = "token"
  }
}
```

## Configuration Reference

The following arguments are supported:

* `remote` - *Optional* - Specifies an Incus remote (Incus server) to connect
	to. See the `remote` reference below for details.

* `config_dir` - *Optional* - The directory to look for existing Incus
	configuration. Defaults to `$HOME/.config/incus`

* `generate_client_certificates` - *Optional* - Automatically generate the Incus
	client certificate if it does not exist. Valid values are `true` and `false`.
	This can also be set with the `INCUS_GENERATE_CLIENT_CERTS` Environment
	variable. Defaults to `false`.

* `accept_remote_certificate` - *Optional* - Automatically accept the Incus
	remote's certificate. Valid values are `true` and `false`. If this is not set
	to `true`, you must accept the certificate out of band of Terraform. This can
	also be set with the `INCUS_ACCEPT_SERVER_CERTIFICATE` environment variable.
  Defaults to `false`

The `remote` block supports:

* `address` - *Optional* - The address of the Incus remote.

* `default` - *Optional* - Whether this should be the default remote.
	This remote will then be used when one is not specified in a resource.
	Valid values are `true` and `false`.
	If you choose to _not_ set default=true on a `remote` and do not specify
	a remote in a resource, this provider will attempt to connect to an Incus
	server running on the same host through the UNIX socket. See `Undefined Remote`
	for more information.
	The default can also be set with the `INCUS_REMOTE` Environment variable.

* `name` - *Optional* - The name of the Incus remote.

* `token` - *Optional* - The one-time trust [token](https://linuxcontainers.org/incus/docs/main/authentication/#adding-client-certificates-using-tokens) used for initial authentication with the Incus remote.

* `port` - *Optional* - The port of the Incus remote.

* `scheme` - *Optional* Whether to connect to the Incus remote via `https` or
	`unix` (UNIX socket). Defaults to `unix`.

## Undefined Remote

If you choose to _not_ define a `remote`, this provider will attempt
to connect to an Incus server running on the same host through the UNIX
socket.

## Environment Variable Remote

It is possible to define a single `remote` through environment variables.
The required variables are:

* `INCUS_REMOTE` - The name of the remote.
* `INCUS_ADDR` - The address of the Incus remote.
* `INCUS_PORT` - The port of the Incus remote.
* `INCUS_TOKEN` - The trust token of the Incus remote.
* `INCUS_SCHEME` - The scheme to use (`unix` or `https`).

## PKI Support

Incus is capable of [authenticating via PKI](https://linuxcontainers.org/incus/docs/main/authentication/#using-a-pki-system). In order to do this, you must
generate appropriate certificates on _both_ the remote/server side and client
side. Details on how to generate these certificates is out of scope of this
document.
