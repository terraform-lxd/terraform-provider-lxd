#!/bin/bash
set -x

vol=$(curl http://169.254.169.254/latest/meta-data/block-device-mapping/ebs0)
if [[ $vol == *404* ]]; then
  # This is AWS. Re-use the already attached ephemeral.
  sudo umount /mnt
  sudo mkfs.btrfs -f /dev/xvdb
  sudo mount -a
  vol=/mnt/lxd
fi

sudo add-apt-repository -y ppa:ubuntu-lxc/lxd-git-master
sudo apt-get update -qq
sudo apt-get install -y lxd
sudo apt-get install -y build-essential
sudo lxc config set core.https_address [::]
sudo lxc config set core.trust_password the-password
#sudo lxc storage create default dir source=/mnt
sudo lxc storage create default btrfs source=$vol
sudo lxc profile device add default root disk path=/ pool=default
sudo lxc network create lxdbr0 ipv6.address=none ipv4.address=192.168.244.1/24 ipv4.nat=true
sudo lxc network attach-profile lxdbr0 default eth0
sudo lxc image copy images:ubuntu/xenial/amd64 local: --alias ubuntu
sudo usermod -a -G lxd ubuntu
sudo chown -R ubuntu: /home/ubuntu/.config

sudo wget -O /usr/local/bin/gimme https://raw.githubusercontent.com/travis-ci/gimme/master/gimme
sudo chmod +x /usr/local/bin/gimme

cat >> ~/.bashrc <<EOF
eval "\$(/usr/local/bin/gimme 1.8)"
export GOPATH=\$HOME/go
export PATH=\$PATH:\$GOROOT/bin:\$GOPATH/bin

export LXD_ADDR=localhost
export LXD_PORT=8443
export LXD_GENERATE_CLIENT_CERTS=true
export LXD_ACCEPT_SERVER_CERTIFICATE=true
export LXD_SCHEME=https
export LXD_PASSWORD=the-password
EOF

eval "$(/usr/local/bin/gimme 1.8)"
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin

go get github.com/sl1pm4t/terraform-provider-lxd
go get github.com/sl1pm4t/lxdhelpers
go get github.com/gosexy/gettext
go get github.com/dustinkirkland/golang-petname

echo fs.inotify.max_queued_events = 1048576 | sudo tee -a /etc/sysctl.conf
echo fs.inotify.max_user_instances = 1048576 | sudo tee -a  /etc/sysctl.conf
echo fs.inotify.max_user_watches = 1048576 | sudo tee -a /etc/sysctl.conf
echo vm.max_map_count = 262144 | sudo tee -a /etc/sysctl.conf
echo kernel.dmesg_restrict = 0 | sudo tee -a /etc/sysctl.conf

sudo reboot
