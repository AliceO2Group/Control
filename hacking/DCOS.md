# Setting up the DCOS Vagrant development environment

Unfortunately there is no suitable setup with just Mesos, so we're going to have to roll out the official DCOS development environment which is based on Vagrant and VirtualBox. Do not try this with less than 32GB of RAM.

## DCOS Vagrant quick start

1. Install Vagrant and VirtualBox.
2. Install vagrant-hostmanager plugin.
```bash
$ vagrant plugin install vagrant-hostmanager
```
3. Clone, configure, deploy. This will take a while.
```bash
$ git clone https://github.com/dcos/dcos-vagrant
$ cd dcos-vagrant
$ cd <path/to/octl/hacking>/VagrantConfig.yaml .
$ vagrant up
```

## Setting up O² software

For this we need [`fpctl`](https://github.com/AliceO2Group/Control). Install it as instructed with `fpctl setup`, then copy into the `fpctl` configuration directory the inventory file for DCOS Vagrant.
```bash
$ cp <path/to/octl/hacking>/inventory ~/.config/fpctl/
```

Then we run `fpctl` to install O² software and configuration. This will also set some Mesos agent attributes which are necessary for matching an O² role to the correct machine. Ansible will spit out some errors, which are generally safe to ignore in this case.
```bash
$ fpctl deploy -e "ignore_errors=yes"
```
