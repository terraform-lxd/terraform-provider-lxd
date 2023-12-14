provider "incus" {
  alias = "unix"

}

resource "incus_instance" "test1" {
  provider = "incus.unix"
  name     = "test1"
  image    = "ubuntu"
  profiles = ["default"]
}

output "test1_ip_address" {
  value = incus_instance.test1.ipv4
}
