resource "tls_private_key" "ca" {
  algorithm = "RSA"

  # Private SSL key
  provisioner "local-exec" {
    command = "echo \"${tls_private_key.ca.private_key_pem}\" > ca_key.pem"
  }
}

resource "tls_self_signed_cert" "ca" {
  key_algorithm   = "RSA"
  private_key_pem = tls_private_key.ca.private_key_pem

  subject {
    common_name  = "example"
    organization = "example.com"
  }

  allowed_uses = [
    "key_encipherment",
    "cert_signing",
    "server_auth",
    "client_auth"
  ]

  validity_period_hours = 24000
  early_renewal_hours   = 720
  is_ca_certificate     = true

  # Certs
  provisioner "local-exec" {
    command = "echo \"${tls_self_signed_cert.ca.cert_pem}\" > ca_cert.pem"
  }

  provisioner "local-exec" {
    command = "echo \"${tls_self_signed_cert.ca.cert_pem}\" > ~/.config/lxc/client.ca"
  }
}

resource "tls_private_key" "local" {
  algorithm = "RSA"

  # Private SSL key
  provisioner "local-exec" {
    command = "echo \"${tls_private_key.local.private_key_pem}\" > client_key.pem"
  }

  provisioner "local-exec" {
    command = "echo \"${tls_private_key.local.private_key_pem}\" > ~/.config/lxc/client.key"
  }

  # OpenSSH key
  provisioner "local-exec" {
    command = "echo '${tls_private_key.local.public_key_openssh}' > id_rsa.pub"
  }

  provisioner "local-exec" {
    command = "echo '${tls_private_key.local.private_key_pem}' > id_rsa"
  }
}

resource "tls_cert_request" "local" {
  key_algorithm   = "RSA"
  private_key_pem = tls_private_key.local.private_key_pem

  dns_names = ["client.example.com"]
  subject {
    common_name = "client"
  }

}

resource "tls_locally_signed_cert" "local" {
  cert_request_pem = tls_cert_request.local.cert_request_pem

  ca_key_algorithm   = "RSA"
  ca_private_key_pem = tls_private_key.ca.private_key_pem
  ca_cert_pem        = tls_self_signed_cert.ca.cert_pem

  validity_period_hours = 24000

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "server_auth",
    "client_auth",
  ]

  provisioner "local-exec" {
    command = "echo \"${tls_locally_signed_cert.local.cert_pem}\" > client_cert.pem"
  }

  provisioner "local-exec" {
    command = "echo \"${tls_locally_signed_cert.local.cert_pem}\" > ~/.config/lxc/client.crt"
  }
}

resource "openstack_compute_keypair_v2" "lxd" {
  name       = "lxd"
  public_key = tls_private_key.local.public_key_openssh
}

resource "openstack_compute_instance_v2" "lxd" {
  name            = "lxd"
  image_name      = "Ubuntu 16.04"
  flavor_name     = "m1.medium"
  key_pair        = openstack_compute_keypair_v2.lxd.name
  security_groups = ["AllowAll"]
  user_data       = "#cloud-config\ndisable_root: false"
}

resource "null_resource" "lxd" {
  connection {
    user        = "root"
    private_key = tls_private_key.local.public_key_openssh
    host        = openstack_compute_instance_v2.lxd.access_ip_v6
  }

  provisioner "remote-exec" {
    inline = [
      "apt-add-repository -y ppa:ubuntu-lxc/stable",
      "apt-get update -qq",
      "apt-get install -y lxd",
      "lxc config set core.https_address [::]",
      "lxc config set core.trust_password password",
      "lxc storage create default dir",
      "lxc profile device add default root disk path=/ pool=default",
      "echo '${tls_self_signed_cert.ca.cert_pem}' | sudo tee /var/lib/lxd/server.ca",
      "echo '${tls_locally_signed_cert.lxd.cert_pem}' | sudo tee /var/lib/lxd/server.crt",
      "echo '${tls_self_signed_cert.ca.cert_pem}' | sudo tee -a /var/lib/lxd/server.crt",
      "echo '${tls_private_key.lxd.private_key_pem}' | sudo tee /var/lib/lxd/server.key",
      "systemctl restart lxd",
      "lxc image copy ubuntu:16.04 local: --alias ubuntu",
    ]
  }
}

resource "tls_private_key" "lxd" {
  algorithm = "RSA"
}

resource "tls_cert_request" "lxd" {
  key_algorithm   = "RSA"
  private_key_pem = tls_private_key.lxd.private_key_pem
  ip_addresses    = ["${replace(openstack_compute_instance_v2.lxd.access_ip_v6, "/[][]/", "")}"]

  subject {
    common_name = openstack_compute_instance_v2.lxd.name
  }
}

resource "tls_locally_signed_cert" "lxd" {
  cert_request_pem = tls_cert_request.lxd.cert_request_pem

  ca_key_algorithm   = "RSA"
  ca_private_key_pem = tls_private_key.ca.private_key_pem
  ca_cert_pem        = tls_self_signed_cert.ca.cert_pem

  validity_period_hours = 24000
  early_renewal_hours   = 720

  allowed_uses = [
    "key_encipherment",
    "server_auth",
    "client_auth"
  ]
}

output "hostname" {
  value = openstack_compute_instance_v2.lxd.access_ip_v6
}
