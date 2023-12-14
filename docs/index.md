# terraform-provider-incus

Use Terraform to manage Incus resources.

## Description

This provider connects to the Incus daemon over local Unix socket or HTTPS.

It makes use of the [Incus client library](https://github.com/lxc/incus), which
currently looks in `~/snap/incus/common/config` (and `~/.config/lxc`) for `client.crt`
and `client.key` files to authenticate against the Incus daemon.

To generate these files and store them in the Incus client config, follow these
[steps](https://documentation.ubuntu.com/incus/en/latest/howto/server_expose/#server-authenticate).
Alternatively, the Incus Terraform provider can generate them on demand if
`generate_client_certificates` is set to true.

Minimum required Incus version is **`3.0`**.

## Basic Example

This is all that is needed if the Incus remotes have been defined out of band via
the `lxc` client.

```hcl
provider "incus" {
}
```

## Specifying Multiple Remotes

If you're running `terraform` from a system where lxc is not installed then you
can define all the remotes in the Provider config:

```hcl
provider "incus" {
  generate_client_certificates = true
  accept_remote_certificate    = true

  remote {
    name     = "incus-server-1"
    scheme   = "https"
    address  = "10.1.1.8"
    password = "password"
    default  = true
  }

  remote {
    name     = "incus-server-2"
    scheme   = "https"
    address  = "10.1.2.8"
    password = "password"
  }
}
```

## Configuration Reference

The following arguments are supported:

- `remote` - _Optional_ - Specifies an Incus remote (Incus server) to connect
  to. See the `remote` reference below for details.

- `config_dir` - _Optional_ - The directory to look for existing Incus
  configuration. Defaults to `$HOME/snap/incus/common/config` (and fallbacks to `$HOME/.config/lxc`).

- `generate_client_certificates` - _Optional_ - Automatically generate the Incus
  client certificate if it does not exist. Valid values are `true` and `false`.
  This can also be set with the `INCUS_GENERATE_CLIENT_CERTS` Environment
  variable. Defaults to `false`.

- `accept_remote_certificate` - _Optional_ - Automatically accept the Incus
  remote's certificate. Valid values are `true` and `false`. If this is not set
  to `true`, you must accept the certificate out of band of Terraform. This can
  also be set with the `INCUS_ACCEPT_SERVER_CERTIFICATE` environment variable.
  Defaults to `false`

The `remote` block supports:

- `address` - _Optional_ - The address of the Incus remote.

- `default` - _Optional_ - Whether this should be the default remote.
  This remote will then be used when one is not specified in a resource.
  Valid values are `true` and `false`.
  If you choose to _not_ set default=true on a `remote` and do not specify
  a remote in a resource, this provider will attempt to connect to an Incus
  server running on the same host through the UNIX socket. See `Undefined Remote`
  for more information.
  The default can also be set with the `INCUS_REMOTE` Environment variable.

- `name` - _Optional_ - The name of the Incus remote.

- `password` - _Optional_ - The password to authenticate to the Incus remote.

- `port` - _Optional_ - The port of the Incus remote.

- `scheme` - _Optional_ Whether to connect to the Incus remote via `https` or
  `unix` (UNIX socket). Defaults to `unix`.

## Undefined Remote

If you choose to _not_ define a `remote`, this provider will attempt
to connect to an Incus server running on the same host through the UNIX
socket.

## Environment Variable Remote

It is possible to define a single `remote` through environment variables.
The required variables are:

- `INCUS_REMOTE` - The name of the remote.
- `INCUS_ADDR` - The address of the Incus remote.
- `INCUS_PORT` - The port of the Incus remote.
- `INCUS_PASSWORD` - The password of the Incus remote.
- `INCUS_SCHEME` - The scheme to use (`unix` or `https`).

## PKI Support

Incus is capable of authenticating via PKI. In order to do this, you must
generate appropriate certificates on _both_ the remote/server side and client
side. Details on how to generate these certificates is out of scope of this
document.
