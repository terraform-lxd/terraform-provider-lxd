# lxd_instance_file

Represents a file in an LXD instance that is not managed by the current Terraform configuration.

This resource is useful for reading files from a LXD instance and passing to other instances, which
works as a simple and effective synchronization mechanism.

## Example

```hcl
# Explanation of what happens in order:
#
# - c1 is created
# - c1 installs nginx and writes its own LAN ip to /my-lan-ip file
# - c2, which was waiting for "c1_ip" to be read, is started
# - c2 invokes curl to read nginx index.html (running in c1) and converts to a text file (/index.txt)
# - output value is set.

data "lxd_instance_file" "c1_ip" {
  instance_name = lxd_instance.c1.name
  target_file = "/my-lan-ip"
  timeout = 30
}

data "lxd_instance_file" "index_txt" {
  instance_name = lxd_instance.c2.name
  target_file = "/index.txt"
  timeout = 30
}

resource "lxd_instance" "c1" {
  name = "c1"
  image = "ubuntu:20.04"
  config = {
    "user.user-data": <<-EOF
    #cloud-config
    package_update: true
    runcmd:
      - apt-get install --no-install-recommends -y nginx
      # get LAN ip
      - ip route get 8.8.8.8 | sed -n '/src/{s/.*src *\([^ ]*\).*/\1/p;q}' > /my-lan-ip
    EOF
  }
}

resource "lxd_instance" "c2" {
  name = "c2"
  image = "ubuntu:20.04"
  config = {
    "user.user-data": <<-EOF
    #cloud-config
    package_update: true
    runcmd:
      - apt-get install --no-install-recommends -y lynx
      - curl http://${trimspace(data.lxd_instance_file.c1_ip.content)} > index.html
      - lynx --dump index.html > /index.txt
    EOF
  }
}

output "out" {
  value = trimspace(data.lxd_instance_file.index_txt.content)
}
```

## Argument Reference

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `instance_name` - *Required* - Name of the instance.

* `target_file` - *Required* - The absolute path of the file on the instance,
	including the filename.

* `project` - *Optional* - Name of the project where the instance from which the file will be read.

* `timeout` - *Optional* - Timeout in seconds to wait for the file.
