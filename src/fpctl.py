#!/usr/bin/env python3


# add to ANSIBLE_CONFIG env variable:
# control_path_dir=/tmp/.ansible/cp/
# host_key_checking=False

import argparse
import errno
import logging
import os
import subprocess
import sys

FPCTL_CONFIG_DIR = os.path.expanduser('~/.config/fpctl')
FPCTL_DATA_DIR = os.path.expanduser('~/.local/share/fpctl')
INVENTORY_FLPS_GROUP = 'flps'
DEFAULT_INVENTORY_PATH = os.path.join(FPCTL_CONFIG_DIR, 'inventory')


def query_yes_no(question, default="yes"):
    """Ask a yes/no question via raw_input() and return their answer.

    "question" is a string that is presented to the user.
    "default" is the presumed answer if the user just hits <Enter>.
        It must be "yes" (the default), "no" or None (meaning
        an answer is required of the user).

    The "answer" return value is True for "yes" or False for "no".
    """
    valid = {"yes": True, "y": True, "ye": True,
             "no": False, "n": False}
    if default is None:
        prompt = " [y/n] "
    elif default == "yes":
        prompt = " [Y/n] "
    elif default == "no":
        prompt = " [y/N] "
    else:
        raise ValueError("invalid default answer: '%s'" % default)

    while True:
        sys.stdout.write(question + prompt)
        choice = input().lower()
        if default is not None and choice == '':
            return valid[default]
        elif choice in valid:
            return valid[choice]
        else:
            sys.stdout.write("Please respond with 'yes' or 'no' "
                             "(or 'y' or 'n').\n")


def bail(description, exit_code=1):
    """Report a fatal error and exit immediately"""
    print('ERROR: {}\n'.format(description))
    print('fpctl will now quit ({}).'.format(exit_code))
    sys.exit(exit_code)


def get_inventory_path(inventory_option):
    """Get the path of the inventory file. May interact with user."""
    inventory_path = DEFAULT_INVENTORY_PATH

    if inventory_option:
        inventory_path = os.path.abspath(inventory_option)
        if not os.path.isfile:
            raise FileNotFoundError(errno.ENOENT,
                                    os.strerror(errno.ENOENT),
                                    inventory_path)
        else:
            return inventory_path
    else:
        if not os.path.isfile(inventory_path):
            if query_yes_no('Ansible inventory file not found at {}. fpctl can '\
                            'autogenerate a default one for you, with target localhost. '\
                            'This means that all FLP prototype software will be '\
                            'deployed on your current system. '\
                            'Would you like to proceed?'\
                            .format(inventory_path)):
                with open(inventory_path, 'w') as inventory_file:
                    print(f'[{INVENTORY_FLPS_GROUP}]\nlocalhost',
                          file=inventory_file)
            else:
                raise FileNotFoundError(errno.ENOENT,
                                        os.strerror(errno.ENOENT),
                                        inventory_path)
    return inventory_path


def deploy(args):
    """Handler for deploy command"""
    # TODO: for commands that do remote operations, go through a check for SSH passwordless
    #      authentication and/or sudoers

    inventory_path = get_inventory_path(args.inventory)

    output = subprocess.check_output(['ansible',
                                      INVENTORY_FLPS_GROUP,
                                      '-i{}'.format(inventory_path),
                                      '--list-hosts'])
    inventory_hosts = output.decode(sys.stdout.encoding).splitlines()
    inventory_hosts = inventory_hosts[1:]  # we throw away the first line which is only a summary
    inventory_hosts = [line.strip() for line in inventory_hosts]

    print('Inventory:\n{}'.format('\n'.join(inventory_hosts)))

    with open(inventory_path, 'r') as inventory_file:
        inventory_file_lines = inventory_file.readlines()

    hosts_that_cannot_ssh = []
    for target_hostname in inventory_hosts:
        # HACK: we check if there's an ansible_user specified for this hostname in the
        #      inventory file. This should be replaced with ansible-python binding.
        ansible_user = os.environ.get('USER')
        for line in inventory_file_lines:
            if line.startswith(target_hostname) and 'ansible_user='in line:
                splitline = line.split(' ')
                for word in splitline:
                    if word.startswith('ansible_user='):
                        ansible_user = word.strip()[13:]

        try:
            output = subprocess.check_output(['ssh',
                                              '-o BatchMode=yes',
                                              '-o ConnectTimeout=5',
                                              '-o StrictHostKeyChecking=no',
                                              '-o GSSAPIAuthentication=yes',
                                              '-o PubkeyAuthentication=no',
                                              f'{ansible_user}@{target_hostname}',
                                              'echo fpctl GSSAPIAuthentication ok'],
                                             stderr=subprocess.STDOUT)
            logging.debug(f'SSH GSSAPI check output:{output.decode(sys.stdout.encoding)}')
        except subprocess.CalledProcessError as e:
            logging.debug(f'SSH GSSAPI check error: {e.output}')

        gssapi_auth_ok = 'fpctl GSSAPIAuthentication ok' in output.decode(sys.stdout.encoding)

        try:
            output = subprocess.check_output(['ssh',
                                              '-o BatchMode=yes',
                                              '-o ConnectTimeout=5',
                                              '-o StrictHostKeyChecking=no',
                                              '-o GSSAPIAuthentication=no',
                                              '-o PubkeyAuthentication=yes',
                                              f'{ansible_user}@{target_hostname}',
                                              'echo fpctl PubkeyAuthentication ok'],
                                             stderr=subprocess.STDOUT)
            logging.debug(f'SSH Pubkey check output:{output.decode(sys.stdout.encoding)}')
        except subprocess.CalledProcessError as e:
            logging.debug(f'SSH Pubkey check error: {e.output}')

        pubkey_auth_ok = 'fpctl PubkeyAuthentication ok' in output.decode(sys.stdout.encoding)

        print(f'Host {target_hostname} SSH GSSAPI login {"OK" if gssapi_auth_ok else "unavailable"}.')
        print(f'Host {target_hostname} SSH Pubkey login {"OK" if pubkey_auth_ok else "unavailable"}.')

        if not pubkey_auth_ok and not gssapi_auth_ok:
            hosts_that_cannot_ssh.append(target_hostname)

    if hosts_that_cannot_ssh:
        ansible_ssh_documentation = 'https://github.com/AliceO2Group/Control#authentication-on-the-target-system'
        print(f'The following hosts do not appear to support passwordless '
              f'authentication (through either GSSAPI/Kerberos or public key):\n'
              f'{chr(10).join(hosts_that_cannot_ssh)}'
              f'\nSince Ansible requires passwordless authentication on the target '
              f'hosts in order to work, fpctl cannot continue.\n'
              f'Please see {ansible_ssh_documentation} for instructions on how to '
              f'set up passwordless authentication for Ansible/fpctl.')
        sys.exit(1)


def configure(args):
    """Handler for configure command"""
    print("Not implemented yet :(\nCalled {}".format(vars(args)))


def run(args):
    """Handler for run command"""
    print("Not implemented yet :(\nCalled {}".format(vars(args)))


def start(args):
    """Handler for start command"""
    print("Not implemented yet :(\nCalled {}".format(vars(args)))


def status(args):
    """Handler for status command"""
    print("Not implemented yet :(\nCalled {}".format(vars(args)))


def stop(args):
    """Handler for stop command"""
    print("Not implemented yet :(\nCalled {}".format(vars(args)))


def log(args):
    """Handler for log command"""
    print("Not implemented yet :(\nCalled {}".format(vars(args)))


def main(argv):
    """Entry point, called by fpctl script."""
    args = argv[1:]
    print("fpctl args: {}".format(", ".join(args)))
    inventory_help = 'path to an Ansible infentory file (default: ~/.config/fpctl/inventory)'

    parser = argparse.ArgumentParser(description='FLP prototype control utility')
    subparsers = parser.add_subparsers(dest='subparser_name')

    sp_deploy = subparsers.add_parser('deploy',
                                      aliases=['de'],
                                      help='deploy FLP prototype software and configuration')
    sp_deploy.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_deploy.set_defaults(func=deploy)

    sp_configure = subparsers.add_parser('configure',
                                         aliases=['co'],
                                         help='deploy FLP prototype configuration')
    sp_configure.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_configure.set_defaults(func=configure)

    sp_run = subparsers.add_parser('run',
                                   help='run a custom command on one or all nodes')
    sp_run.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_run.set_defaults(func=run)

    sp_start = subparsers.add_parser('start',
                                     help='start some or all FLP prototype processes')
    sp_start.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_start.set_defaults(func=start)

    sp_status = subparsers.add_parser('status',
                                      help='view status of some or all FLP prototype processes')
    sp_status.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_status.set_defaults(func=status)

    sp_stop = subparsers.add_parser('stop',
                                    help='stop some or all FLP prototype processes')
    sp_stop.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_stop.set_defaults(func=stop)

    sp_log = subparsers.add_parser('log',
                                   help='view the logs of some or all FLP prototype processes')
    sp_log.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_log.set_defaults(func=log)

    # Update and setup are fake entries with the only purpose of generating documentation.
    # They are handled by the fpctl shell script
    subparsers.add_parser('update',
                          aliases=['up'],
                          help='update fpctl deployment information')
    subparsers.add_parser('setup',
                          help='install and configure fpctl')

    parsed_args = parser.parse_args(args)
    logging.debug('argparse output: {}'.format(vars(parsed_args)))

    if not parsed_args.subparser_name:
        print('No operation specified.')
        parser.print_help()
        sys.exit(1)

    parsed_args.func(parsed_args)

    # note to self: action='store_true' for arguments with no value


if __name__ == '__main__':
    main(sys.argv)
