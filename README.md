# Control

This repository hosts those parts of the control system for the September 2017 TPC test that aren't already kept in system-configuration.

## Automatic setup with `fpctl`

`fpctl` is an FLP prototype setup and control utility. Its goal is to make it easy
to deploy, configure and control an FLP prototype testing stack on one or more
target machines.

`fpctl` requires CERN CentOS 7, with or without Kerberos authentication. If your source or target systems are not set up with CERN Kerberos authentication, you must enable passwordless login via public key authentication (see [Authentication on the target system](https://github.com/AliceO2Group/Control/blob/master/README.md#authentication-on-the-target-system)).

Set the `fpctl` install path and get `fpctl`:
```
export FPCTL_ROOT_DIR=~/.local
mkdir -p $FPCTL_ROOT_DIR/bin
curl -o $FPCTL_ROOT_DIR/bin/fpctl https://raw.githubusercontent.com/AliceO2Group/Control/master/src/fpctl
chmod +x $FPCTL_ROOT_DIR/bin/fpctl
export PATH="$FPCTL_ROOT_DIR/bin:$PATH"
```

Add `fpctl` binary directory to PATH (optional but useful):
```
echo -e '\nexport PATH='"$FPCTL_ROOT_DIR"'/bin:$PATH\n' >> ~/.bashrc
```

Setup `fpctl` (if you don't run this it will run anyway the first time you use `fpctl` for some other operation):
```
fpctl setup
```
You should now have some repositories in `~/.local/share/fpctl` (or some other directory, depending on your `FPCTL_ROOT_DIR`). The setup routine also takes care of installing things like git and Ansible on your system.

Currently, in order to install `fpctl` you need access to certain git repositories on CERN GitLab. To gain access, you need to join [the alice-o2-detector-teams e-group](https://e-groups.cern.ch/e-groups/EgroupsSearch.do?searchValue=alice-o2-detector-teams).

If you want to deploy FLP prototype on a remote system, you should create an Ansible inventory file in `~/.config/fpctl/inventory`. Otherwise, if you simply wish to deploy all the FLP prototype software on your local machine, you can proceed immediately:
```
fpctl deploy
```
If you haven't provided an inventory file, `fpctl` will offer to create one for you. `fpctl` may also offer to configure passwordless sudo, depending on your configuration. The deployment operation may take a while. You will see some Ansible output as everything is installed and configured.

Start/stop all the tasks (`fpctl` and Ansible take care of doing this in the correct order):
```
fpctl start
fpctl stop
```

You may also choose to only control one of the tasks:
```
fpctl start readout
fpctl start qctask
fpctl start qcchecker
```
An instruction like these previous three will start a given task on all the inventory items assigned to it. For instance, if you have two hosts under `[flp-readout]` in your inventory, `fpctl start readout` will start two processes, one for each machine. Similarly, `fpctl stop readout` will stop them all.

Occasionally you might want to run `fpctl update` to pull any changes to the deployment data and to the `fpctl` tool itself.

To deploy any configuration changes to your FLP prototype systems without reinstalling everything, run:
```
fpctl configure
```
It is a good idea to then restart all the FLP prototype processes to apply the new configuration.

For more information, check `fpctl --help`.

## fpctl Configuration and Inventory

`fpctl` uses standard Ansible inventory files. The default inventory file path is `~/.config/fpctl/inventory`. An alternative inventory file can be passed to `fpctl` with the option `-i` or `--inventory`. A `fpctl`/Ansible inventory file must provide one or more hosts for each of the following four machine groups:

- `flp-readout`
- `qc-task`
- `qc-checker`
- `qc-repository`

A host can belong to more than one group, and in fact, the default inventory that `fpctl deploy` can generate automatically looks like this:

```
[flp-readout]
localhost ansible_connection=local
[qc-task]
localhost ansible_connection=local
[qc-checker]
localhost ansible_connection=local
[qc-repository]
localhost ansible_connection=local
```

The whitespace-separated key-value pairs after the hostname are per-host variables. These can override Ansible settings, as well as any variable internally used by `fpctl`'s Ansible playbooks. For example:

- `ansible_become_method=ksu` - use `ksu` instead of `sudo` for acquiring privileges on the target system (useful on Kerberos setups);
- `small_hugepages_count`/`large_hugepages_count` - number of hugepages allocated for the readout process;
- `flpprototype_qc_mysql_root_password` - the MariaDB root password of an existing MariaDB instance for the `qc-repository` (only necessary if a password exists),
- `ansible_connection=local` - run everything locally instead of using SSH (only applies to `localhost`)

Here's an example of an inventory file with some remote machines, including two readout machines, one QC checker machine, and one machine with a QC task and a QC repository:

```
[flp-readout]
my-readout-1.cern.ch ansible_become_method=ksu small_hugepages_count=256
my-readout-2.cern.ch ansible_become_method=ksu small_hugepages_count=256
[qc-task]
my-qctask.cern.ch ansible_become_method=ksu
[qc-checker]
my-qcchecker.cern.ch
[qc-repository]
my-qctask.cern.ch ansible_become_method=ksu
```

For more information on inventory files, see [the Ansible Inventory documentation](http://docs.ansible.com/ansible/intro_inventory.html).

## Quick start manual setup with Ansible

Assuming a default CC7 setup with Kerberos authentication. If your source or target systems are **not** set up with CERN Kerberos authentication, you must enable passwordless login via public key authentication (see [Authentication on the target system](#authentication-on-the-target-system)).

Create the inventory file:
```
echo -e "[flp-readout]\nlocalhost ansible_connection=local\n[qc-task]\nlocalhost ansible_connection=local\n[qc-checker]\nlocalhost ansible_connection=local\n[qc-repository]\nlocalhost ansible_connection=local\n" > myinventory
```
Replace `localhost` with the hostname of your target machine (and remove the `ansible_connection=local` variable which only applies to `localhost`).

Install Ansible, clone the repos and deploy:
```
sudo yum -y install git ansible
git clone https://github.com/AliceO2Group/Control.git
git clone https://gitlab.cern.ch/AliceO2Group/system-configuration.git
sudo sed -i "/^# control_path_dir =/s/.*/control_path_dir = \/tmp\/.ansible\/cp/" /etc/ansible/ansible.cfg
sudo sed -i "/^#host_key_checking =/s/.*/host_key_checking = False/" /etc/ansible/ansible.cfg
cd system-configuration/ansible
ansible-playbook -i ../../myinventory -s site.yml -e "flpprototype_systemd=../../Control/systemd/system"
```
The `git clone` statements above assume git over HTTPS. If you use SSH public key authentication with GitHub and/or GitLab, you may do the following instead:
```
git clone git@github.com:AliceO2Group/Control.git
git clone ssh://git@gitlab.cern.ch:7999/AliceO2Group/system-configuration.git
```

## Full guide for Ansible deployment

### Setting up Ansible

To use this, you need to clone this repo, as well as the system-configuration repo which contains the Ansible configuration.

```
$ sudo yum install git ansible
$ git clone git@github.com:AliceO2Group/Control.git
$ git clone ssh://git@gitlab.cern.ch:7999/AliceO2Group/system-configuration.git
```

You should also create an inventory file which points to one or more fresh systems. Here's what an inventory file should look like:
```
[flp-readout]
my-readout-testing-machine.cern.ch
my-other-readout-testing-machine.cern.ch
[qc-task]
...
[qc-checker]
...
[qc-repository]
...
```

The target system should accept passwordless SSH authentication (Kerberos, public key). This guide assumes that the target system is a clean CC7 instance on CERN OpenStack.

If you are using Kerberos login for Ansible (default if you run CC7 with your CERN user account), you must also add an option in your inventory file to do passwordless privilege escalation with `ksu` instead of `sudo`, as the latter does not support `NOPASSWD` with Kerberos.

```
[flp-readout]
cc7-testing-machine.cern.ch ansible_become_method=ksu
[qc-task]
cc7-testing-machine.cern.ch ansible_become_method=ksu
[qc-checker]
cc7-testing-machine.cern.ch ansible_become_method=ksu
[qc-repository]
cc7-testing-machine.cern.ch ansible_become_method=ksu
```

### Ansible and AFS

If your home directory is *not* on AFS, skip to the next section.

If you are running a default CC7 configuration with your home directory on AFS on your control machine, you must change the `control_path_dir` value in `/etc/ansible/ansible.cfg` to **any path that is not on AFS**. For instance, `/tmp/.ansible/cp` is a good value that's already suggested in the configuration file, so all you have to do is uncomment it.

The reason for this is that Ansible uses SSH multiplexing to avoid creating new TCP connections for each SSH session to a target machine after the first one. This improves performance, but requires a socket file, which Ansible places in `~/.ansible/cp` by default. AFS doesn't like this, and Ansible's SSH fails with an "Operation not permitted" error.

For more information, see https://en.wikibooks.org/wiki/OpenSSH/Cookbook/Multiplexing#Errors_Preventing_Multiplexing.

### Authentication on the target system

If you are running CC7 with your CERN user account and Kerberos authentication on both your system and the target system, skip to the next section (but be sure to set `ksu` as privilege escalation tool in your inventory).

Before running Ansible commands on a target system, a way is needed for Ansible to log in and perform tasks which usually require root privileges. As far as the target system is concerned, you should make sure that:
* either the target system allows SSH login as root (configuration file `/etc/ssh/sshd_config`), accepts public key authentication for root, and Ansible is run as root (by appending `-u root` to Ansible commands); OR
* the target system accepts public key authentication for the unprivileged user, and this user is `sudo`-enabled with `NOPASSWD` on the target system.

Ideally one would use an unprivileged user, and keep SSH root login disabled (default on CC7). If this is the case, the user on the target system must be in the group `wheel`. The command `# gpasswd -a username wheel` adds a user to the `wheel` group. To allow passwordless `sudo` the line `%wheel  ALL=(ALL)       NOPASSWD: ALL` should be present and uncommented in the sudoers configuration file. To check this, run `# visudo` as root on the target system.

To enable public key authentication on the target system, the following steps are needed.

1) Make sure you have a public key on the control machine (i.e. your machine), it is usually called `~/.ssh/id_rsa.pub` or `~/.ssh/id_dsa.pub`. If not, you can create one with `ssh-keygen`.
2) Add the contents of your `id_rsa.pub` (or similar) to `~/.ssh/authorized_keys` *on the target system*. To do that, either SSH into it and copy the contents, or run `ssh-copy-id your_username@target_hostname` on the control machine.
3) Since you are now relying on SSH public key authentication, you must make sure that your inventory file does not contain `ansible_become_method=ksu`, as this only works with Kerberos.
4) You must also make sure that the unprivileged user on the target system is a member of the `wheel` group (`# gpasswd -a username wheel`) and that the line `%wheel  ALL=(ALL)       NOPASSWD: ALL` is uncommented in the `sudoers` file (editable with `visudo`).

For more information on SSH public key authentication, see https://help.ubuntu.com/community/SSH/OpenSSH/Keys.

### Running ansible-playbook

Assuming the current directory is the one with Ansible's `site.yml` (directory `ansible` in the system-configuration repository) and assuming this repository (Control) is cloned at `~/Control`, this is the single step for deployment, configuration and execution (adjust the paths as needed):

```
$ ansible-playbook -i path/to/inventory/file -s site.yml -e "flpprototype_systemd=~/Control/systemd/system"
```

This will install `alisw-flpproto` with all its dependencies on the machines from the relevant inventory file and deploy the dummy configuration files. It will also deploy some Systemd units for readout and QC.

Add `-t `*`tag`*` ` where *`tag`* is `installation`, `configuration` or `execution` to only run one of these phases.

### Things to do on the target machine

View the logs for a service:

`$ sudo journalctl -u flpprototype-readout`

`$ sudo journalctl -u flpprototype-qctask`

`$ sudo journalctl -u flpprototype-qcchecker`

Control the service:

`$ sudo systemctl start flpprototype-readout`

`$ sudo systemctl status flpprototype-readout`

`$ sudo systemctl stop flpprototype-readout`

### Parametrized services

Systemd templates allow the user to pass arguments when starting a unit.

Start a readout service with a specific configuration (by default, configuration files are deployed to `/etc/flpprototype.d`):

`$ sudo systemctl start flpprototype-readout@configDummy`

The QC task service is similar, but it requires two parameters (the task name and the configuration file name):

`$ sudo systemctl start flpprototype-qctask@myTask_1@example-default`

### Things to do on the controller machine

Query or control the flpprototype-readout Systemd service state on all machines without going through the Ansible role:

`$ ansible -b -i myinventoryfile all -a "systemctl start flpprototype-readout"`

`$ ansible -b -i myinventoryfile all -a "systemctl status flpprototype-readout"`

`$ ansible -b -i myinventoryfile all -a "systemctl stop flpprototype-readout"`

Example with QC task, parametrized:

`$ ansible -b -i myinventoryfile all -a "systemctl status flpprototype-qctask@myTask_1@example-default"`
