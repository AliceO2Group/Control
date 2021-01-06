#!/bin/bash

# Requirements:
# LXD installed and running
# created profile "macvlan" to enable host bridge networking
#
# Usage:
# ./newcontainer.sh hostname1 hostname2 hostname3...


echo -n New root password for containers $@:
read -s ROOTPASS
echo

for GUEST in "$@"
do
        echo guest is $GUEST
        lxc launch images:centos/7/amd64 $GUEST -p default -p macvlan

        # Normally we'd use lxc-wait, but it's broken as of Nov 2020
        #lxc-wait -n $GUEST -s RUNNING
        # Workaround:
        lxc exec $GUEST -- bash -c 'while [ "$(systemctl is-system-running 2>/dev/null)" != "running" ] && [ "$(systemctl is-system-running 2>/dev/null)" != "degraded" ]; do :; done'

        # Set root password
        lxc exec $GUEST -- bash -c "echo 'root:$ROOTPASS' | chpasswd"
        lxc exec $GUEST -- bash -c "yum -y install openssh-server sudo nano which numactl-libs firewalld"

        # Set persistent hostname
        lxc exec $GUEST -- bash -c "hostnamectl set-hostname $GUEST"

        # Allow root login
        lxc exec $GUEST -- bash -c "sed -i 's/#PermitRootLogin yes/PermitRootLogin yes/g' /etc/ssh/sshd_config"
        lxc exec $GUEST -- bash -c "systemctl restart sshd"
        lxc exec $GUEST -- bash -c "systemctl enable sshd"
done
