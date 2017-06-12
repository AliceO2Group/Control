# Control

This repository hosts those parts of the control system for the September 2017 TPC test that aren't already kept in system-configuration.

## Getting started

To use this, you need to clone this repo, as well as the system-configuration repo which contains the Ansible configuration.

```
$ yum install ansible
$ git clone git@github.com:AliceO2Group/Control.git
$ git clone ssh://git@gitlab.cern.ch:7999/AliceO2Group/system-configuration.git
```

It is also advisable to edit the inventory file so it points to a fresh system (in the system-configuration repository, `ansible/inventory/flpproto-control-testing`). The target system should accept SSH public key authentication.

Before running Ansible commands on a target system, a way is needed for Ansible to log in and perform tasks which usually require root privileges. As far as the target system is concerned, you should make sure that:
* either the target system allows SSH login as root (configuration file `/etc/ssh/sshd_config`), accepts public key authentication for root, and Ansible is run as root (by appending `-u root` to Ansible commands); OR
* the target system accepts public key authentication for the unprivileged user, and this user is sudo-enabled with NOPASSWD on the target system.

Ideally one would use an unprivileged user, and keep SSH root login disabled (default on CC7). If this is the case, (on CC7) the user on the target system should be in the group `wheel` (`# gpasswd -a username wheel`) and the line `%wheel  ALL=(ALL)       NOPASSWD: ALL` should be present and uncommented in the sudoers configuration file. To check this, run `visudo` as root on the target system.

Assuming the current directory is the one with Ansible's `site.yml` (directory `ansible` in the system-configuration repository) and assuming this repository (Control) is cloned at `~/Control`, this is the single step for deployment, configuration and execution:

`$ ansible-playbook -i inventory/flpproto-control-testing -s site.yml -e "flpprototype_systemd=~/Control/systemd/system"`

This will install readout with all its dependencies on the machines (clean CC7) from the relevant inventory file, deploy the dummy configuration file and run the readout process through the Systemd unit.

Add `-t `*`tag`*` ` where *`tag`* is `installation`, `configuration` or `execution` to only run one of these phases.

## On the target machine

View the logs for the readout service:

`# journalctl -u flpprototype-readout`

Control the service:

`# systemctl start flpprototype-readout`

`# systemctl status flpprototype-readout`

`# systemctl stop flpprototype-readout`

Start a readout service with a specific configuration (by default, configuration files are deployed to `/etc/flpprototype.d`):

`# systemctl start flpprototype-readout@configDummy`

## On the controller machine

Query or control the flpprototype-readout Systemd service state on all readout machines without going through the Ansible role:

`$ ansible -b -i inventory/flpproto-control-testing all -a "systemctl start flpprototype-readout"`

`$ ansible -b -i inventory/flpproto-control-testing all -a "systemctl status flpprototype-readout"`

`$ ansible -b -i inventory/flpproto-control-testing all -a "systemctl stop flpprototype-readout"`

