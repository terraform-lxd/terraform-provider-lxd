provider "lxd" {
  alias   = "https"
  scheme  = "https"
  address = "192.168.1.8"
}

resource "lxd_container" "test2" {
  provider = "lxd.https"
  name     = "test2"
  image    = "ubuntu"
  profiles = ["default"]
}

output "test2_ip_address" {
  value = "${lxd_container.test2.ipv4}"
}
