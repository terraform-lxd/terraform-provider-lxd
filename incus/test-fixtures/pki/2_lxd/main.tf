provider "incus" {
  address         = "CHANGEME"
  scheme          = "https"
  remote          = "foo"
  remote_password = "password"
}

resource "incus_instance" "container1" {
  name     = "foo"
  image    = "ubuntu"
  profiles = ["default"]
}
