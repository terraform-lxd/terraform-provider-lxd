provider "lxd" {
  alias = "unix"

}

resource "lxd_instance" "test1" {
  provider = "lxd.unix"
  name     = "test1"
  image    = "ubuntu"
  profiles = ["default"]
}

output "test1_ip_address" {
  value = lxd_instance.test1.ipv4
}
