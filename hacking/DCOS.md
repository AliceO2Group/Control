# Setting up the DC/OS Vagrant development environment

Unfortunately there is no suitable setup with just Mesos, so we're going to have to roll out the
official DC/OS development environment which is based on Vagrant and VirtualBox. Do not try this
with less than 32GB of RAM.

## DC/OS Vagrant quick start

1. Install Vagrant and VirtualBox.
2. Install vagrant-hostmanager plugin.
```bash
$ vagrant plugin install vagrant-hostmanager
```
3. Clone, configure, deploy. This will take a while.
```bash
$ git clone https://github.com/dcos/dcos-vagrant
$ cd dcos-vagrant
$ cp <path/to/o²control/hacking>/VagrantConfig.yaml .
$ vagrant up
```
See [the DC/OS Vagrant README](https://github.com/dcos/dcos-vagrant/blob/master/README.md) for more information.

## Setting up O² software

This bit really depends on what we want to run. As a minimum, the executor and occ plugin must be installed
on the DC/OS cluster, and the easiest way to do this is from aliBuild-generated RPMs `alisw-Control` including
`alisw-Control-OCCPlugin`.

[`fpctl`](https://github.com/AliceO2Group/Control/tree/master/fpctl) can help with this. Install it
as [instructed](https://github.com/AliceO2Group/Control/blob/master/fpctl/README.md) with
`fpctl setup`, then copy into the `fpctl` configuration directory the inventory file for DC/OS Vagrant.
```bash
$ cp <path/to/o²control/hacking>/inventory ~/.config/fpctl/
```

Then we run `fpctl` to install O² software and configuration. This will also set some Mesos agent
attributes which can be useful for matching an O² role to the correct machine. Ansible will spit
out some errors, which are generally safe to ignore in this case.
```bash
$ fpctl deploy -e "ignore_errors=yes"
```

If `fpctl` prompts for it, the password for the `vagrant` user on the DC/OS VMs is `vagrant`.

Continue with [running O² Control](RUNNING.md).
