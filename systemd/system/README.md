# Systemd templates for FLP prototype

These are Jinja2 templates, to be deployed through Ansible which then outputs proper Systemd units.

To use this, you need to clone this repo, as well as the system-configuration repo which contains the Ansible configuration.

Assuming the current directory is the one with Ansible's `site.yml` and assuming this repository (Control) is cloned at `~/Control`:

`$ ansible-playbook -i inventory/flpproto-control-testing -s site.yml -e "flpprototype_systemd=~/Control/systemd/system"`

This will install readout with all its dependencies on the machines (clean CC7) from the relevant inventory file, deploy the dummy configuration file and run the readout process through the Systemd unit.

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
