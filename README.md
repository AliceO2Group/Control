# Control

This repository hosts those parts of the control system for the September 2017 TPC test that aren't already kept in system-configuration.

## Quick start

Assuming a default CC7 setup with Kerberos authentication.

Create the inventory file:
```
echo -e "[flps]\nmy-testing-machine.cern.ch ansible_become_method=ksu collectd_network_host=localhost" > myinventory
```
Replace my-testing-machine.cern.ch with the hostname of your target machine (can also be `localhost`).

Install Ansible, clone the repos and deploy:
```
sudo yum -y install git ansible
git clone git@github.com:AliceO2Group/Control.git
git clone ssh://git@gitlab.cern.ch:7999/AliceO2Group/system-configuration.git
sudo sed -i "/^# control_path_dir =/s/.*/control_path_dir = \/tmp\/.ansible\/cp/" /etc/ansible/ansible.cfg
sudo sed -i "/^#host_key_checking =/s/.*/host_key_checking = False/" /etc/ansible/ansible.cfg
cd system-configuration/ansible
ansible-playbook -i ../../myinventory -s site.yml -e "flpprototype_systemd=../../Control/systemd/system"
```

## Full guide

### Setting up Ansible

To use this, you need to clone this repo, as well as the system-configuration repo which contains the Ansible configuration.

```
$ sudo yum install git ansible
$ git clone git@github.com:AliceO2Group/Control.git
$ git clone ssh://git@gitlab.cern.ch:7999/AliceO2Group/system-configuration.git
```

You should also create an inventory file which points to one or more fresh systems. Here's what an inventory file should look like:
```
[flps]
my-readout-testing-machine.cern.ch
my-other-readout-testing-machine.cern.ch
```

The target system should accept passwordless SSH authentication (Kerberos, public key). This guide assumes that the target system is a clean CC7 instance on CERN OpenStack.

If you are using Kerberos login for Ansible (default if you run CC7 with your CERN user account), you must also add an option in your inventory file to do passwordless privilege escalation with `ksu` instead of `sudo`, as the latter does not support `NOPASSWD` with Kerberos.

```
[flps]
cc7-testing-machine.cern.ch ansible_become_method=ksu collectd_network_host=localhost
```

### Ansible and AFS

If your home directory is *not* on AFS, skip to the next section.

If you are running a default CC7 configuration with your home directory on AFS on your control machine, you must change the `control_path_dir` value in `/etc/ansible/ansible.cfg` to **any path that is not on AFS**. For instance, `/tmp/.ansible/cp` is a good value that's already suggested in the configuration file, so all you have to do is uncomment it.

The reason for this is that Ansible uses SSH multiplexing to avoid creating new TCP connections for each SSH session to a target machine after the first one. This improves performance, but requires a socket file, which Ansible places in `~/.ansible/cp` by default. AFS doesn't like this, and Ansible's SSH fails with an "Operation not permitted" error.

For more information, see https://en.wikibooks.org/wiki/OpenSSH/Cookbook/Multiplexing#Errors_Preventing_Multiplexing.

### Authentication on the target system

If you are running CC7 with your CERN user account and Kerberos authentication, skip to the next section (but be sure to set `ksu` as privilege escalation tool in your inventory).

Before running Ansible commands on a target system, a way is needed for Ansible to log in and perform tasks which usually require root privileges. As far as the target system is concerned, you should make sure that:
* either the target system allows SSH login as root (configuration file `/etc/ssh/sshd_config`), accepts public key authentication for root, and Ansible is run as root (by appending `-u root` to Ansible commands); OR
* the target system accepts public key authentication for the unprivileged user, and this user is `sudo`-enabled with `NOPASSWD` on the target system.

Ideally one would use an unprivileged user, and keep SSH root login disabled (default on CC7). If this is the case, the user on the target system must be in the group `wheel`. The command `# gpasswd -a username wheel` adds a user to the `wheel` group. To allow passwordless `sudo` the line `%wheel  ALL=(ALL)       NOPASSWD: ALL` should be present and uncommented in the sudoers configuration file. To check this, run `# visudo` as root on the target system.

### Running ansible-playbook

Assuming the current directory is the one with Ansible's `site.yml` (directory `ansible` in the system-configuration repository) and assuming this repository (Control) is cloned at `~/Control`, this is the single step for deployment, configuration and execution (adjust the paths as needed):

```
$ ansible-playbook -i path/to/inventory/file -s site.yml -e "flpprototype_systemd=~/Control/systemd/system"
```

This will install `alisw-flpproto` with all its dependencies on the machines from the relevant inventory file and deploy the dummy configuration files. It will also deploy some Systemd units for readout and QC.

Add `-t `*`tag`*` ` where *`tag`* is `installation`, `configuration` or `execution` to only run one of these phases.

## Things to do on the target machine

View the logs for a service:

`$ sudo journalctl -u flpprototype-readout`

`$ sudo journalctl -u flpprototype-qctask`

`$ sudo journalctl -u flpprototype-qcchecker`

Control the service:

`$ sudo systemctl start flpprototype-readout`

`$ sudo systemctl status flpprototype-readout`

`$ sudo systemctl stop flpprototype-readout`

## Parametrized services

Systemd templates allow the user to pass arguments when starting a unit.

Start a readout service with a specific configuration (by default, configuration files are deployed to `/etc/flpprototype.d`):

`$ sudo systemctl start flpprototype-readout@configDummy`

The QC task service is similar, but it requires two parameters (the task name and the configuration file name):

`$ sudo systemctl start flpprototype-qctask@myTask_1@example-default`

## Things to do on the controller machine

Query or control the flpprototype-readout Systemd service state on all machines without going through the Ansible role:

`$ ansible -b -i myinventoryfile all -a "systemctl start flpprototype-readout"`

`$ ansible -b -i myinventoryfile all -a "systemctl status flpprototype-readout"`

`$ ansible -b -i myinventoryfile all -a "systemctl stop flpprototype-readout"`

Example with QC task, parametrized:

`$ ansible -b -i myinventoryfile all -a "systemctl status flpprototype-qctask@myTask_1@example-default"`


