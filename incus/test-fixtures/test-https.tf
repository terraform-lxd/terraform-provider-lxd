provider "incus" {
  alias   = "https"
  scheme  = "https"
  address = "192.168.1.8"
}

resource "incus_instance" "test2" {
  provider = "incus.https"
  name     = "test2"
  image    = "ubuntu"
  profiles = ["default"]
}

output "test2_ip_address" {
  value = incus_instance.test2.ipv4
}
