provider "lxd" {
  address         = "CHANGEME"
  scheme          = "https"
  remote          = "foo"
  remote_password = "password"
}

resource "lxd_instance" "container1" {
  name     = "foo"
  image    = "ubuntu"
  profiles = ["default"]
}
