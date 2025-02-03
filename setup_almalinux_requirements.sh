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
# Install Podman
sudo dnf install podman -y
# Install Go
dnf install golang -y
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
sed -i 's/PATH=$PATH:$HOME\/.local\/bin\/:$HOME\/bin/PATH=$PATH:$HOME\/.local\/bin\/:$HOME\/bin:$GOPATH\/bin/' ~/.bashrc