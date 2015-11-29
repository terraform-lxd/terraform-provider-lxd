provider "lxd" {
  alias = "unix"

}

resource "lxd_container" "test1" {
  provider  = "lxd.unix"
  name      = "test1"
  image     = "ubuntu"
  profiles  = ["default"]
}

output "test1_ip_address" {
  value = "${lxd_container.test1.ipv4}"
}
