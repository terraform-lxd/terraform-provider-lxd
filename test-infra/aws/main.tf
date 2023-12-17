variable "private_key" {}

variable "key_name" {
  default = "default"
}

variable "ami" {
  default = "ami-835b4efa"
}

provider "aws" {
  region = "us-west-2"
}

resource "aws_security_group" "incus_acc_tests" {
  name        = "incus_acc_tests"
  description = "Incus Test Infra Allow all inbound traffic"

  ingress {
    from_port   = 1
    to_port     = 65535
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 1
    to_port     = 65535
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = -1
    to_port     = -1
    protocol    = "icmp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_spot_instance_request" "incus_acc_tests" {
  ami                  = var.ami
  spot_price           = "0.0221"
  instance_type        = "c3.large"
  wait_for_fulfillment = true
  spot_type            = "one-time"

  key_name = var.key_name

  security_groups = ["${aws_security_group.incus_acc_tests.name}"]

  root_block_device {
    volume_size           = 20
    delete_on_termination = true
  }

  tags {
    Name = "Incus Acceptance Test Infra"
  }
}

resource "null_resource" "incus_acc_tests" {
  connection {
    type        = "ssh"
    user        = "ubuntu"
    host        = aws_spot_instance_request.incus_acc_tests.public_ip
    private_key = file(var.private_key)
  }

  provisioner "remote-exec" {
    script = "../files/deploy.sh"
  }
}

output "public_ip" {
  value = aws_spot_instance_request.incus_acc_tests.public_ip
}
