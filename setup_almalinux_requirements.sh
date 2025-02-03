#!/bin/bash
# Update the system
sudo dnf --refresh update -y
sudo dnf upgrade -y
# Add Docker repository
sudo dnf install yum-utils -y
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
# Install Docker
sudo dnf install docker-ce docker-ce-cli containerd.io docker-compose-plugin -y
# Start and enable Docker
sudo systemctl enable --now docker
sudo systemctl start docker

# Install the SIG RPM key.
sudo rpm --import https://www.centos.org/keys/RPM-GPG-KEY-CentOS-SIG-Kmods
# Install the repository.
cat > /etc/yum.repos.d/centos-kmods-kernel-latest.repo <<'EOF'
[centos-kmods-kernel-latest-repos]
name=CentOS $releasever - Kmods - Kernel Latest - Repositories
metalink=https://mirrors.centos.org/metalink?repo=centos-kmods-sig-kernel-latest-$releasever&arch=$basearch&protocol=https,http
#baseurl=http://mirror.stream.centos.org/SIGs/$releasever/kmods/$basearch/kernel-latest
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-CentOS-SIG-Kmods
gpgcheck=1
repo_gpgcheck=0
metadata_expire=6h
countme=1
enabled=1
EOF

# Update the kernel to the latest from the repository added.
dnf update

# Install the btrfs-progs package to add needed utilities for interacting with BTRFS file systems.
dnf install btrfs-progs
# Install Podman dependencies
sudo dnf -y install conmon containers-common crun iptables netavark nftables slirp4netns btrfs-progs
# Install Podman
sudo dnf install podman -y
# Install Go
dnf install golang -y
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
sed -i 's/PATH=$PATH:$HOME\/.local\/bin\/:$HOME\/bin/PATH=$PATH:$HOME\/.local\/bin\/:$HOME\/bin:$GOPATH\/bin/' ~/.bashrc
