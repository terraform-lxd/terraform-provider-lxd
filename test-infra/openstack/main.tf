variable "key_name" {}

variable "private_key" {}

variable "image_id" {}

variable "network_id" {}

variable "pool" {
  default = "public"
}

variable "flavor" {
  default = "m1.large"
}

resource "openstack_networking_secgroup_v2" "lxd_acc_tests" {
  name        = "lxd_acc_tests"
  description = "Rules for lxd acceptance tests"
}

resource "openstack_networking_floatingip_v2" "lxd_acc_tests" {
  pool = "${var.pool}"
}

resource "openstack_networking_secgroup_rule_v2" "lxd_acc_tests_rule_1" {
  security_group_id = "${openstack_networking_secgroup_v2.lxd_acc_tests.id}"
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 1
  port_range_max    = 65535
  remote_ip_prefix  = "0.0.0.0/0"
}

resource "openstack_networking_secgroup_rule_v2" "lxd_acc_tests_rule_2" {
  security_group_id = "${openstack_networking_secgroup_v2.lxd_acc_tests.id}"
  direction         = "ingress"
  ethertype         = "IPv6"
  protocol          = "tcp"
  port_range_min    = 1
  port_range_max    = 65535
  remote_ip_prefix  = "::/0"
}

resource "openstack_networking_secgroup_rule_v2" "lxd_acc_tests_rule_3" {
  security_group_id = "${openstack_networking_secgroup_v2.lxd_acc_tests.id}"
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "udp"
  port_range_min    = 1
  port_range_max    = 65535
  remote_ip_prefix  = "0.0.0.0/0"
}

resource "openstack_networking_secgroup_rule_v2" "lxd_acc_tests_rule_4" {
  security_group_id = "${openstack_networking_secgroup_v2.lxd_acc_tests.id}"
  direction         = "ingress"
  ethertype         = "IPv6"
  protocol          = "udp"
  port_range_min    = 1
  port_range_max    = 65535
  remote_ip_prefix  = "::/0"
}

resource "openstack_networking_secgroup_rule_v2" "lxd_acc_tests_rule_5" {
  security_group_id = "${openstack_networking_secgroup_v2.lxd_acc_tests.id}"
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "icmp"
  remote_ip_prefix  = "0.0.0.0/0"
}

resource "openstack_blockstorage_volume_v2" "lxd_acc_tests" {
  name = "lxd_acc_tests"
  size = 10
}

resource "openstack_networking_secgroup_rule_v2" "lxd_acc_tests_rule_6" {
  security_group_id = "${openstack_networking_secgroup_v2.lxd_acc_tests.id}"
  direction         = "ingress"
  ethertype         = "IPv6"
  protocol          = "icmp"
  remote_ip_prefix  = "::/0"
}

resource "openstack_compute_instance_v2" "lxd_acc_tests" {
  name        = "lxd_acc_tests"
  image_id    = "${var.image_id}"
  flavor_name = "${var.flavor}"
  key_pair    = "${var.key_name}"

  security_groups = ["${openstack_networking_secgroup_v2.lxd_acc_tests.name}"]

  network {
    uuid = "${var.network_id}"
  }
}

resource "openstack_compute_floatingip_associate_v2" "lxd_acc_tests" {
  instance_id = "${openstack_compute_instance_v2.lxd_acc_tests.id}"
  floating_ip = "${openstack_networking_floatingip_v2.lxd_acc_tests.address}"
}

resource "openstack_compute_volume_attach_v2" "lxd_acc_tests" {
  instance_id = "${openstack_compute_instance_v2.lxd_acc_tests.id}"
  volume_id   = "${openstack_blockstorage_volume_v2.lxd_acc_tests.id}"
}

resource "null_resource" "lxd_acc_tests" {
  connection {
    user        = "ubuntu"
    type        = "ssh"
    private_key = "${file(var.private_key)}"
    host        = "${openstack_compute_floatingip_associate_v2.lxd_acc_tests.floating_ip}"
  }

  provisioner "remote-exec" {
    script = "../files/deploy.sh"
  }
}

output "public_ip" {
  value = "${openstack_compute_floatingip_associate_v2.lxd_acc_tests.floating_ip}"
}
