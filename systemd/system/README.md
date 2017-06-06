# Systemd templates for FLP prototype

To be deployed through Ansible, which outputs proper Systemd units:
`ansible-playbook -i inventory/flpproto-control-testing -s site.yml -e "flpprototype_systemd=~/Control/systemd/system"`
