provider "lxd" {
  address = "CHANGEME"
  scheme = "https"
  remote = "foo"
  remote_password = "password"
}

resource "lxd_container" "container1" {
  name = "foo"
  image = "ubuntu"
  profiles = ["default"]
}
