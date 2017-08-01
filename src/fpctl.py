#!/usr/bin/env python3

import argparse
import errno
import getpass
import json
import logging
import os
import re
import subprocess
import sys
from operator import itemgetter
from collections import OrderedDict

try:
    from colorama import Fore, Style
    from terminaltables import SingleTable
except ImportError as e:
    print('==> ERROR: cannot import a required Python module. Run fpctl setup to ensure '
          'all dependencies are installed.')
    print('Missing module: {}'.format(e.name))
    sys.exit(1)


FPCTL_CONFIG_DIR = os.path.expanduser('~/.config/fpctl')
FPCTL_ROOT_DIR = os.path.expanduser('~/.local')
with open(os.path.join(FPCTL_CONFIG_DIR, '.installed')) as f:
    root_dir = f.readline().strip()
    if (os.path.isdir(root_dir)):
        FPCTL_ROOT_DIR = root_dir
    else:
        raise FileNotFoundError(errno.ENOENT,
                                os.strerror(errno.ENOENT),
                                root_dir)
FPCTL_DATA_DIR = os.path.expanduser(os.path.join(FPCTL_ROOT_DIR, 'share/fpctl'))
INVENTORY_READOUT_GROUP = 'flp-readout'
INVENTORY_QCTASK_GROUP = 'qc-task'
INVENTORY_QCCHECKER_GROUP = 'qc-checker'
INVENTORY_QCREPOSITORY_GROUP = 'qc-repository'
DEFAULT_INVENTORY_PATH = os.path.join(FPCTL_CONFIG_DIR, 'inventory')

TARGET_GROUPS = ['flp-readout', 'qc-task', 'qc-checker']
TASK_NAMES = ['readout', 'qctask', 'qcchecker']

C_WARN = Style.BRIGHT + Fore.YELLOW + '==> WARNING: ' + Style.RESET_ALL
C_YELL = Style.BRIGHT + Fore.YELLOW + '==> ' + Style.RESET_ALL
C_QUEST = C_YELL
C_ERR = Style.BRIGHT + Fore.RED + '==> ERROR: ' + Style.RESET_ALL
C_RED = Style.BRIGHT + Fore.RED + '==> ' + Style.RESET_ALL
C_MSG = Style.BRIGHT + Fore.GREEN + '==> ' + Style.RESET_ALL
C_ITEM_NO_PADDING = Style.BRIGHT + Fore.BLUE + '-> ' + Style.RESET_ALL
C_ITEM = '  ' + C_ITEM_NO_PADDING
BULLET = '\u25CF '


class Inventory:
    def __init__(self, inventory_path):
        self.inventory_path = inventory_path
        self.inventory_file_lines = []
        self.hosts_cache_file_path = os.path.join(FPCTL_DATA_DIR, 'hosts_cache.json')
        self.hosts_cache = dict()
        self.__init_cache()

    def __init_cache(self):
        hosts_cache = dict()
        if os.path.isfile(self.hosts_cache_file_path):
            try:
                with open(self.hosts_cache_file_path, 'r') as hosts_cache_file:
                    hosts_cache = json.load(hosts_cache_file)
            except Exception as e:
                print(C_WARN + 'A fpctl hosts cache exists but loading failed, so the ' +
                      'cache will be overwritten. If you see this message more than ' +
                      'once, try reinstalling fpctl.')
        self.hosts_cache = hosts_cache

    def __write_cache(self):
        try:
            with open(self.hosts_cache_file_path, 'w') as hosts_cache_file:
                json.dump(self.hosts_cache, hosts_cache_file)
        except Exception as e:
            print(C_WARN + 'Cannot write fpctl hosts cache. If you see this ' +
                  'message more than once, try reinstalling fpctl.')

    def load(self):
        output = subprocess.check_output(['ansible',
                                          'all',
                                          '-i{}'.format(self.inventory_path),
                                          '--list-hosts'])
        inventory_hosts = output.decode(sys.stdout.encoding).splitlines()
        inventory_hosts = inventory_hosts[1:]  # we throw away the first line which is only a summary
        self.inventory_hosts = [line.strip() for line in inventory_hosts]

        inventory_file_lines = []
        with open(self.inventory_path, 'r') as inventory_file:
            inventory_file_lines = inventory_file.readlines()

        self.inventory_file_lines = inventory_file_lines

    def check_hosts(self, force=False):
        try:
            self.__check_for_ssh_auth()
            self.__check_for_sudo_nopasswd()
        finally:
            if self.hosts_cache:
                self.__write_cache()

    def __check_for_ssh_auth(self, force=False):
        hosts_that_cannot_ssh = []
        has_localhosts = False

        result = []
        for target_hostname in self.inventory_hosts:
            ansible_user = os.environ.get('USER')
            for line in self.inventory_file_lines:
                if line.startswith(target_hostname) and 'ansible_user='in line:
                    splitline = line.split(' ')
                    for word in splitline:
                        if word.startswith('ansible_user='):
                            ansible_user = word.strip()[13:]
                            break  # we found an ansible_user override, so we break and go on

            if not force and \
               target_hostname in self.hosts_cache and \
               'auth_methods' in self.hosts_cache[target_hostname] and \
               self.hosts_cache[target_hostname]['auth_methods'] and \
               'ansible_user' in self.hosts_cache[target_hostname] and \
               self.hosts_cache[target_hostname]['ansible_user'] == ansible_user:
                result.append({'host': target_hostname,
                               'auth': self.hosts_cache[target_hostname]['auth_methods']})
                continue

            # HACK: we check if there's an ansible_user specified for this hostname in the
            #      inventory file. This should be replaced with ansible-python binding.
            if target_hostname == 'localhost':
                has_localhosts = True
                self.hosts_cache['localhost'] = {'auth_methods': ['local'],
                                                 'ansible_user': ansible_user}
                result.append({'host': 'localhost',
                               'auth': ['local']})
                continue

            output = b''
            try:
                output = subprocess.check_output(['ssh',
                                                  '-o BatchMode=yes',
                                                  '-o ConnectTimeout=5',
                                                  '-o StrictHostKeyChecking=no',
                                                  '-o GSSAPIAuthentication=yes',
                                                  '-o PubkeyAuthentication=no',
                                                  '{0}@{1}'.format(ansible_user, target_hostname),
                                                  'echo fpctl GSSAPIAuthentication ok'],
                                                 stderr=subprocess.STDOUT)
                logging.debug('SSH GSSAPI check output:{}'.format(output.decode(sys.stdout.encoding)))
            except subprocess.CalledProcessError as e:
                logging.debug('SSH GSSAPI check error: {}'.format(e.output))

            gssapi_auth_ok = 'fpctl GSSAPIAuthentication ok' in output.decode(sys.stdout.encoding)

            try:
                output = subprocess.check_output(['ssh',
                                                  '-o BatchMode=yes',
                                                  '-o ConnectTimeout=5',
                                                  '-o StrictHostKeyChecking=no',
                                                  '-o GSSAPIAuthentication=no',
                                                  '-o PubkeyAuthentication=yes',
                                                  '{0}@{1}'.format(ansible_user, target_hostname),
                                                  'echo fpctl PubkeyAuthentication ok'],
                                                 stderr=subprocess.STDOUT)
                logging.debug('SSH Pubkey check output:{}'.format(output.decode(sys.stdout.encoding)))
            except subprocess.CalledProcessError as e:
                logging.debug('SSH Pubkey check error: {}'.format(e.output))

            pubkey_auth_ok = 'fpctl PubkeyAuthentication ok' in output.decode(sys.stdout.encoding)

            self.hosts_cache[target_hostname] = {'auth_methods': [],
                                                 'ansible_user': ansible_user}
            if not pubkey_auth_ok and not gssapi_auth_ok:
                hosts_that_cannot_ssh.append(target_hostname)

            if pubkey_auth_ok or gssapi_auth_ok:
                auth_ok = []
                if pubkey_auth_ok:
                    auth_ok.append('public key')
                    self.hosts_cache[target_hostname]['auth_methods'].append('public key')
                if gssapi_auth_ok:
                    auth_ok.append('GSSAPI/Kerberos')
                    self.hosts_cache[target_hostname]['auth_methods'].append('GSSAPI/Kerberos')
                result.append({'host': target_hostname, 'auth': auth_ok})

        if has_localhosts:
            print(C_QUEST + 'At least one of your target systems is localhost. SSH authentication '
                  'checks were skipped for localhost inventory entries. '
                  'Make sure that you have ansible_connection=local '
                  'in your inventory, and that passwordless sudo is enabled.')

        if hosts_that_cannot_ssh:
            ansible_ssh_documentation = 'https://github.com/AliceO2Group/Control#authentication-on-the-target-system'
            print(C_ERR + 'The following hosts do not appear to support passwordless '
                  'authentication (through either GSSAPI/Kerberos or public key):')
            for host in hosts_that_cannot_ssh:
                print(C_ITEM + host)
            print(C_RED + 'Since Ansible requires passwordless authentication on the target '
                  'hosts in order to work, fpctl cannot continue.\n' + C_RED +
                  'Please see {} for instructions on how to '
                  'set up passwordless authentication for Ansible/fpctl.'
                  .format(ansible_ssh_documentation))
            self.__write_cache()
            sys.exit(1)

        print(C_MSG + 'Hosts in inventory:')
        for item in result:
            print(C_ITEM + item['host'] + ' [authentication: ' + ', '.join(item['auth']) + ']')

    def __check_for_sudo_nopasswd(self, force=False):
        for target_hostname in self.inventory_hosts:
            ansible_user = os.environ.get('USER')
            for line in self.inventory_file_lines:
                if line.startswith(target_hostname) and 'ansible_user='in line:
                    splitline = line.split(' ')
                    for word in splitline:
                        if word.startswith('ansible_user='):
                            ansible_user = word.strip()[13:]
                            break  # we found an ansible_user override, so we break and go on

            become_with_ksu = False
            for line in self.inventory_file_lines:
                if line.startswith(target_hostname) and 'ansible_become_method=ksu' in line:
                    become_with_ksu = True
                    break  # if this host is set up with Kerberos+ksu, we skip to the next
            if become_with_ksu:
                continue

            if not force and \
               target_hostname in self.hosts_cache and \
               'sudo_nopasswd' in self.hosts_cache[target_hostname] and \
               self.hosts_cache[target_hostname]['sudo_nopasswd'] and \
               'ansible_user' in self.hosts_cache[target_hostname] and \
               self.hosts_cache[target_hostname]['ansible_user'] == ansible_user:
                continue

            output = b''
            if target_hostname == 'localhost':
                try:
                    output = subprocess.check_output(['/bin/sudo -kn echo "fpctl sudo ok"'],
                                                     shell=True,
                                                     stderr=subprocess.STDOUT)
                    logging.debug('local sudo check output:{}'.format(output.decode(sys.stdout.encoding)))
                except subprocess.CalledProcessError as e:
                    logging.debug('local sudo check error: {}'.format(e.output))
            else:
                try:
                    output = subprocess.check_output(['ssh',
                                                      '-o BatchMode=yes',
                                                      '-o ConnectTimeout=5',
                                                      '-o StrictHostKeyChecking=no',
                                                      '{0}@{1}'.format(ansible_user, target_hostname),
                                                      '/bin/sudo -kn echo "fpctl sudo ok"'],
                                                     stderr=subprocess.STDOUT)
                    logging.debug('SSH sudo check output:{}'.format(output.decode(sys.stdout.encoding)))
                except subprocess.CalledProcessError as e:
                    logging.debug('SSH sudo check error: {}'.format(e.output))

            sudo_ok = 'fpctl sudo ok' in output.decode(sys.stdout.encoding)
            self.hosts_cache[target_hostname]['sudo_nopasswd'] = sudo_ok

            if not sudo_ok:
                if query_yes_no('Passwordless sudo not set on host {0}. fpctl requires '
                                'sudo NOPASSWD configuration in order to work. To '
                                'enable this, you should add a file named "zzz-fpctl" to '
                                'the /etc/sudoers.d directory on host {0}, with the '
                                'content "{1} ALL=(ALL) NOPASSWD: ALL".\n'
                                'You may quit fpctl and do it yourself, or fpctl can do '
                                'this for you now. Do you wish to proceed with enabling '
                                'passwordless sudo?'.format(target_hostname, ansible_user),
                                default="yes"):
                    sudoers_extra_path = '/etc/sudoers.d/zzz-fpctl'

                    file_cmd = '/bin/sudo -Sk su -c "EDITOR=tee visudo -f {}"' \
                               .format(sudoers_extra_path)
                    password = getpass.getpass(prompt='[sudo] password for {0}@{1}: '
                                                      .format(ansible_user, target_hostname))
                    sudoers_line = '{} ALL=(ALL) NOPASSWD: ALL\n'.format(ansible_user)

                    if target_hostname == 'localhost':
                        p = subprocess.Popen(file_cmd,
                                             shell=True,
                                             stdin=subprocess.PIPE,
                                             stderr=subprocess.PIPE,
                                             stdout=subprocess.DEVNULL,
                                             universal_newlines=True)
                    else:
                        p = subprocess.Popen(['ssh',
                                              '-o BatchMode=yes',
                                              '-o ConnectTimeout=5',
                                              '-o StrictHostKeyChecking=no',
                                              '{0}@{1}'.format(ansible_user, target_hostname),
                                              file_cmd],
                                             stdin=subprocess.PIPE,
                                             stderr=subprocess.PIPE,
                                             stdout=subprocess.DEVNULL,
                                             universal_newlines=True)

                    p.communicate('{0}\n{1}'.format(password, sudoers_line))
                    if p.returncode:
                        print(C_ERR + 'Could not set up passwordless sudo on host {}. fpctl will now quit.'
                              .format(target_hostname))
                        self.__write_cache()
                        sys.exit(p.returncode)
                    else:
                        self.hosts_cache[target_hostname]['sudo_nopasswd'] = True
                        print(C_MSG + 'Passwordless sudo OK on host {}.'.format(target_hostname))

                else:
                    print(C_ERR + 'Passwordless sudo not allowed on host {}. fpctl will now quit.'
                          .format(target_hostname))
                    self.__write_cache()
                    sys.exit(0)


def check_for_correct_task(args):
    if args.task:
        if args.task not in TASK_NAMES:
            print(C_ERR + 'Unknown task "{}".'.format(args.task))
            print(C_RED + 'Available tasks:')
            for task_name in TASK_NAMES:
                print(C_ITEM + task_name)
            sys.exit(1)


def print_summary(inventory_path):
    all_target_groups = TARGET_GROUPS + ['qc-repository']
    all_task_names = TASK_NAMES + [Style.RESET_ALL + Style.DIM + Fore.WHITE + '(none)' + Style.RESET_ALL]
    systemd_units = ['flpprototype-readout',
                     'flpprotocype-qctask',
                     'flpprototype-qcchecker',
                     Style.RESET_ALL + Style.DIM + Fore.WHITE + '(none)' + Style.RESET_ALL]
    target_hosts = dict()
    for group in all_target_groups:
        output = subprocess.check_output(['ansible',
                                          group,
                                          '-i{}'.format(inventory_path),
                                          '--list-hosts'])
        if b'hosts (0)' in output:
            target_hosts[group] = []
            continue

        inventory_hosts = output.decode(sys.stdout.encoding).splitlines()
        inventory_hosts = inventory_hosts[1:]  # we throw away the first line which is only a summary
        inventory_hosts = [line.strip() for line in inventory_hosts]
        target_hosts[group] = inventory_hosts

    headers = list('\n'.join(Style.BRIGHT + Fore.BLUE + line + Style.RESET_ALL for line in item.splitlines()) for item in
                   ['Inventory groups',
                    'Target hosts',
                    'Systemd units\n(on target hosts)',
                    'Tasks\n(accessible through fpctl)'])

    rows = list(zip(('[' + item + ']' for item in all_target_groups),
                    ('\n'.join(item) for item in target_hosts.values()),
                    systemd_units,
                    (Style.BRIGHT + Fore.BLUE + item + Style.RESET_ALL for item in all_task_names)))

    table = SingleTable([headers] +
                        rows)
    table.inner_row_border = True
    table.CHAR_H_INNER_HORIZONTAL = b'\xcd'.decode('ibm437')
    table.CHAR_OUTER_TOP_HORIZONTAL = b'\xcd'.decode('ibm437')
    table.CHAR_OUTER_TOP_LEFT = b'\xd5'.decode('ibm437')
    table.CHAR_OUTER_TOP_RIGHT = b'\xb8'.decode('ibm437')
    table.CHAR_OUTER_TOP_INTERSECT = b'\xd1'.decode('ibm437')
    table.CHAR_H_OUTER_LEFT_INTERSECT = b'\xc6'.decode('ibm437')
    table.CHAR_H_OUTER_RIGHT_INTERSECT = b'\xb5'.decode('ibm437')
    table.CHAR_H_INNER_INTERSECT = b'\xd8'.decode('ibm437')
    print(table.table)
    print(C_MSG + 'Configuration files were deployed in /etc/flpprototype.d on the target systems.')
    print(C_MSG + 'FLP prototype software is installed in /opt/alisw. If you wish to use ' +
          'it manually, you must run "module load flpproto" after you SSH into a target system.')
    print(C_MSG + 'It is now possible to control the tasks listed in the last column through fpctl.')


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
        sys.stdout.write(C_QUEST +
                         question + prompt + '\n' +
                         C_QUEST + '------------------------------------\n' +
                         C_QUEST)
        choice = input().lower()
        if default is not None and choice == '':
            return valid[default]
        elif choice in valid:
            return valid[choice]
        else:
            sys.stdout.write(C_QUEST +
                             "Please respond with 'yes' or 'no' "
                             "(or 'y' or 'n').\n" +
                             C_QUEST)


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
            if query_yes_no('Ansible inventory file not found at {}. fpctl can '
                            'autogenerate a default one for you, with target localhost. '
                            'This means that all FLP prototype software will be '
                            'deployed on your current system. '
                            'Would you like to proceed?'
                            .format(inventory_path), default="yes"):
                with open(inventory_path, 'w') as inventory_file:
                    loc = 'localhost ansible_connection=local'
                    inv = ''
                    for group in [INVENTORY_READOUT_GROUP,
                                  INVENTORY_QCTASK_GROUP,
                                  INVENTORY_QCCHECKER_GROUP,
                                  INVENTORY_QCREPOSITORY_GROUP]:
                        inv += '[{0}]\n{1}\n'.format(group, loc)
                    print("{}".format(inv), file=inventory_file)
            else:
                raise FileNotFoundError(errno.ENOENT,
                                        os.strerror(errno.ENOENT),
                                        inventory_path)
    return inventory_path


def deploy(args):
    """Handler for deploy command"""
    inventory_path = get_inventory_path(args.inventory)

    inv = Inventory(inventory_path)
    inv.load()
    inv.check_hosts(force=True)

    ansible_cwd = os.path.join(FPCTL_DATA_DIR, 'system-configuration/ansible')

    ansible_systemd_path = os.path.join(FPCTL_DATA_DIR, 'Control/systemd/system')
    ansible_systemd_var = 'flpprototype_systemd={}'.format(ansible_systemd_path)

    ansible_extra_params = []
    if (args.ansible_extra_params):
        ansible_extra_params += args.ansible_extra_params.split(' ')

    ansible_extra_vars = [ansible_systemd_var]
    if (args.ansible_extra_vars):
        ansible_extra_vars += args.ansible_extra_vars.split(' ')

    ansible_cmd = ['ansible-playbook',
                   os.path.join(ansible_cwd, 'site.yml'),
                   '-i{}'.format(inventory_path),
                   '-s',
                   '-e"{}"'.format(' '.join(ansible_extra_vars))]
    if args.verbose:
        ansible_cmd += ['-vvv']
    if ansible_extra_params:
        ansible_cmd += ansible_extra_params

    ansible_env = os.environ.copy()
    ansible_env['ANSIBLE_CONFIG'] = os.path.join(FPCTL_CONFIG_DIR, 'ansible.cfg')

    ansible_proc = subprocess.Popen(' '.join(ansible_cmd),
                                    shell=True,
                                    cwd=ansible_cwd,
                                    env=ansible_env)
    ansible_proc.communicate()
    print(C_MSG + 'Deployment summary:')
    print_summary(inventory_path)
    print(C_MSG + 'All done.')


def configure(args):
    """Handler for configure command"""

    inventory_path = get_inventory_path(args.inventory)

    inv = Inventory(inventory_path)
    inv.load()
    inv.check_hosts()

    ansible_cwd = os.path.join(FPCTL_DATA_DIR, 'system-configuration/ansible')

    ansible_systemd_path = os.path.join(FPCTL_DATA_DIR, 'Control/systemd/system')
    ansible_systemd_var = 'flpprototype_systemd={}'.format(ansible_systemd_path)

    ansible_extra_params = []
    if (args.ansible_extra_params):
        ansible_extra_params += args.ansible_extra_params.split(' ')

    ansible_extra_vars = [ansible_systemd_var]
    if (args.ansible_extra_vars):
        ansible_extra_vars += args.ansible_extra_vars.split(' ')

    ansible_cmd = ['ansible-playbook',
                   os.path.join(ansible_cwd, 'site.yml'),
                   '-i{}'.format(inventory_path),
                   '-s',
                   '-tconfiguration'
                   '-e"{}"'.format(' '.join(ansible_extra_vars))]
    if args.verbose:
        ansible_cmd += ['-vvv']
    if ansible_extra_params:
        ansible_cmd += ansible_extra_params

    ansible_env = os.environ.copy()
    ansible_env['ANSIBLE_CONFIG'] = os.path.join(FPCTL_CONFIG_DIR, 'ansible.cfg')

    ansible_proc = subprocess.Popen(' '.join(ansible_cmd),
                                    shell=True,
                                    cwd=ansible_cwd,
                                    env=ansible_env)
    ansible_proc.communicate()
    print(C_MSG + 'Configuration summary:')
    print_summary(inventory_path)
    print(C_MSG + 'All done.')


def run(args):
    """Handler for run command"""

    inventory_path = get_inventory_path(args.inventory)

    inv = Inventory(inventory_path)
    inv.load()
    inv.check_hosts()

    host = args.host
    custom_command = args.command

    ansible_extra_params = []
    if (args.ansible_extra_params):
        ansible_extra_params += args.ansible_extra_params.split(' ')

    ansible_extra_vars = []
    if (args.ansible_extra_vars):
        ansible_extra_vars += args.ansible_extra_vars.split(' ')

    ansible_cmd = ['ansible',
                   host,
                   '-i"{}"'.format(inventory_path)]
    if args.verbose:
        ansible_cmd += ['-vvv']
    if ansible_extra_params:
        ansible_cmd += ansible_extra_params
    if ansible_extra_vars:
        ansible_cmd += ['-e"{}"'.format(' '.join(ansible_extra_vars))]

    ansible_cmd += ['-a"{}"'.format(custom_command)]
    ansible_env = os.environ.copy()
    ansible_env['ANSIBLE_CONFIG'] = os.path.join(FPCTL_CONFIG_DIR, 'ansible.cfg')

    ansible_proc = subprocess.Popen(' '.join(ansible_cmd),
                                    shell=True,
                                    env=ansible_env)
    ansible_proc.communicate()
    print(C_MSG + 'All done.')


def start(args):
    """Handler for start command"""

    inventory_path = get_inventory_path(args.inventory)

    check_for_correct_task(args)
    inv = Inventory(inventory_path)
    inv.load()
    inv.check_hosts()

    ansible_cwd = os.path.join(FPCTL_DATA_DIR, 'system-configuration/ansible')

    ansible_extra_params = []
    if (args.ansible_extra_params):
        ansible_extra_params += args.ansible_extra_params.split(' ')

    ansible_extra_vars = []
    if (args.ansible_extra_vars):
        ansible_extra_vars += args.ansible_extra_vars.split(' ')

    ansible_cmd = ['ansible-playbook',
                   os.path.join(ansible_cwd, 'control.yml'),
                   '-i{}'.format(inventory_path),
                   '-s',
                   '-t{}control-start'
                   .format('{}-'.format(args.task) if args.task else '')]
    if args.verbose:
        ansible_cmd += ['-vvv']
    if ansible_extra_params:
        ansible_cmd += ansible_extra_params
    if ansible_extra_vars:
        ansible_cmd += ['-e"{}"'.format(' '.join(ansible_extra_vars))]

    ansible_env = os.environ.copy()
    ansible_env['ANSIBLE_CONFIG'] = os.path.join(FPCTL_CONFIG_DIR, 'ansible.cfg')

    ansible_proc = subprocess.Popen(' '.join(ansible_cmd),
                                    shell=True,
                                    cwd=ansible_cwd,
                                    env=ansible_env)
    ansible_proc.communicate()
    print(C_MSG + 'All done.')


def status(args):
    """Handler for status command"""
    inventory_path = get_inventory_path(args.inventory)

    check_for_correct_task(args)
    inv = Inventory(inventory_path)
    inv.load()
    inv.check_hosts()

    ansible_cwd = os.path.join(FPCTL_DATA_DIR, 'system-configuration/ansible')

    ansible_extra_params = []
    if (args.ansible_extra_params):
        ansible_extra_params += args.ansible_extra_params.split(' ')

    ansible_extra_vars = []
    if (args.ansible_extra_vars):
        ansible_extra_vars += args.ansible_extra_vars.split(' ')

    print(C_MSG + 'Checking status...')

    ansible_cmd = ['ansible-playbook',
                   os.path.join(ansible_cwd, 'control.yml'),
                   '-i{}'.format(inventory_path),
                   '-s',
                   '-t{}control-status'
                   .format('{}-'.format(args.task) if args.task else '')]
    if args.verbose:
        ansible_cmd += ['-vvv']
    if ansible_extra_params:
        ansible_cmd += ansible_extra_params
    if ansible_extra_vars:
        ansible_cmd += ['-e"{}"'.format(' '.join(ansible_extra_vars))]

    ansible_env = os.environ.copy()
    ansible_env['ANSIBLE_CONFIG'] = os.path.join(FPCTL_CONFIG_DIR, 'ansible.cfg')

    ansible_proc = subprocess.Popen(' '.join(ansible_cmd),
                                    shell=True,
                                    cwd=ansible_cwd,
                                    env=ansible_env,
                                    stdout=subprocess.PIPE)

    output_lines = []
    if args.verbose:
        while True:
            nextline = ansible_proc.stdout.readline()
            u_nextline = nextline.decode(sys.stdout.encoding)
            if 'PLAY RECAP' in u_nextline.strip():
                break
            sys.stdout.write(u_nextline)
            output_lines.append(u_nextline.rstrip())
            sys.stdout.flush()

    out, err = ansible_proc.communicate()

    if not args.verbose:
        output_lines = out.decode(sys.stdout.encoding).splitlines()

    # The output from the a control playbook contains a specially formatted debug
    # module instance. We need to extract it and parse it as JSON.
    in_json = False
    json_entries = []
    for line in output_lines:
        if 'TASK' in line and 'control-status-output-json' in line:
            in_json = True
            continue
        if in_json:
            if line.startswith('task path') or line.startswith('META:'):
                continue
            if line.startswith('ok: ['):
                json_entries.append('{\n')
            elif not line.strip():
                in_json = False
            else:
                json_entries[-1] += (line + '\n')
        else:
            continue

    # print(C_MSG + 'Raw output:\n' + '\n'.join(output_lines))
    json_objects = []
    for entry in json_entries:
        # print(C_ITEM + 'ITEM:  ' + entry)
        obj = json.loads(entry)
        # print(C_ITEM + 'OBJECT:' + str(obj))
        json_objects.append(obj['msg'])

    # By service
    tables = dict()
    rows = []
    available_task_names = TASK_NAMES
    available_target_groups = TARGET_GROUPS
    if args.task:
        # this is ok because check_for_correct_task runs early on
        available_task_names = [args.task]
        available_target_groups = [TARGET_GROUPS[TASK_NAMES.index(args.task)]]

    for i in range(len(available_task_names)):
        servicename = available_task_names[i]
        target_group = available_target_groups[i]
        if servicename not in tables:
            tables[servicename] = list()

        for obj in json_objects:
            if obj['service'] == servicename:
                units = dict()
                for line in obj['systemctl_status_output']:
                    unitname = line.split(':')[0]
                    unitstatus = line.split(':')[1]
                    unitname = re.sub('\.service$', '', unitname)
                    units[unitname] = unitstatus

                for line in obj['systemctl_list_unit_files_output']:
                    unitname = re.sub('\.service$', '', line)
                    if '@' in unitname:
                        continue
                    if unitname not in units:
                        units[unitname] = 'inactive'

                units = OrderedDict(sorted(units.items()))

                unitnames = []
                unitstatuses = []
                for i, (unitname, unitstatus) in enumerate(units.items()):
                    c_bullet = BULLET
                    if unitstatus == 'active':
                        c_bullet = Style.BRIGHT + Fore.GREEN + BULLET + Style.RESET_ALL
                    elif unitstatus == 'reloading' or unitstatus == 'activating' or unitstatus == 'deactivating':
                        c_bullet = Style.BRIGHT + Fore.YELLOW + BULLET + Style.RESET_ALL
                    elif unitstatus == 'inactive':
                        c_bullet = Style.BRIGHT + Fore.WHITE + BULLET + Style.RESET_ALL
                    elif unitstatus == 'failed' or unitstatus == 'error':
                        c_bullet = Style.BRIGHT + Fore.RED + BULLET + Style.RESET_ALL

                    unitnames.append('   ' + unitname)
                    unitstatuses.append(c_bullet + unitstatus)

                tables[servicename].append(['   ' + obj['host'],
                                            '\n'.join(unitnames),
                                            '\n'.join(unitstatuses)])

        tables[servicename] = sorted(tables[servicename], key=itemgetter(0))
        for row in tables[servicename]:
            if not row[1]:
                row[1] = Style.DIM + Fore.WHITE + '   (no units found)' + Style.RESET_ALL
            if not row[2]:
                row[2] = Style.DIM + Fore.WHITE + BULLET + '(none)' + Style.RESET_ALL

        headers = [Style.BRIGHT + Fore.BLUE + 'Inventory group' + Style.RESET_ALL + '\n   Target hosts',
                   Style.BRIGHT + Fore.BLUE + 'Task' + Style.RESET_ALL + '\n   Systemd units',
                   ' \nStatus']

        rows += [[Style.BRIGHT + Fore.BLUE + '[' + target_group + ']' + Style.RESET_ALL,
                  Style.BRIGHT + Fore.BLUE + servicename + Style.RESET_ALL]]
        rows += tables[servicename]

    table = SingleTable([headers] +
                        rows)
    table.inner_row_border = True
    table.inner_column_border = False
    table.CHAR_H_INNER_HORIZONTAL = b'\xcd'.decode('ibm437')
    table.CHAR_OUTER_TOP_HORIZONTAL = b'\xcd'.decode('ibm437')
    table.CHAR_OUTER_TOP_LEFT = b'\xd5'.decode('ibm437')
    table.CHAR_OUTER_TOP_RIGHT = b'\xb8'.decode('ibm437')
    table.CHAR_OUTER_TOP_INTERSECT = b'\xd1'.decode('ibm437')
    table.CHAR_H_OUTER_LEFT_INTERSECT = b'\xc6'.decode('ibm437')
    table.CHAR_H_OUTER_RIGHT_INTERSECT = b'\xb5'.decode('ibm437')
    table.CHAR_H_INNER_INTERSECT = b'\xd8'.decode('ibm437')
    print(table.table)


def stop(args):
    """Handler for stop command"""
    inventory_path = get_inventory_path(args.inventory)

    check_for_correct_task(args)
    inv = Inventory(inventory_path)
    inv.load()
    inv.check_hosts()

    ansible_cwd = os.path.join(FPCTL_DATA_DIR, 'system-configuration/ansible')

    ansible_extra_params = []
    if (args.ansible_extra_params):
        ansible_extra_params += args.ansible_extra_params.split(' ')

    ansible_extra_vars = []
    if (args.ansible_extra_vars):
        ansible_extra_vars += args.ansible_extra_vars.split(' ')

    ansible_cmd = ['ansible-playbook',
                   os.path.join(ansible_cwd, 'control.yml'),
                   '-i{}'.format(inventory_path),
                   '-s',
                   '-t{}control-stop'
                   .format('{}-'.format(args.task) if args.task else '')]
    if args.verbose:
        ansible_cmd += ['-vvv']
    if ansible_extra_params:
        ansible_cmd += ansible_extra_params
    if ansible_extra_vars:
        ansible_cmd += ['-e"{}"'.format(' '.join(ansible_extra_vars))]

    ansible_env = os.environ.copy()
    ansible_env['ANSIBLE_CONFIG'] = os.path.join(FPCTL_CONFIG_DIR, 'ansible.cfg')

    ansible_proc = subprocess.Popen(' '.join(ansible_cmd),
                                    shell=True,
                                    cwd=ansible_cwd,
                                    env=ansible_env)
    ansible_proc.communicate()
    print(C_MSG + 'All done.')


def log(args):
    """Handler for log command"""
    print(C_ERR + "Not implemented yet :(\nCalled {}".format(vars(args)))


def main(argv):
    """Entry point, called by fpctl script."""
    args = argv[1:]
    inventory_help = 'path to an Ansible infentory file (default: ~/.config/fpctl/inventory)'
    verbose_help = 'print more output for debugging purposes'
    ansible_extra_params_help = 'additional command line parameters to be passed to Ansible, enclosed in quotes'
    ansible_extra_vars_help = 'additional variables for Ansible as key=value or JSON, enclosed in quotes'

    parser = argparse.ArgumentParser(description=C_MSG + 'FLP prototype control utility',
                                     prog='fpctl',
                                     epilog='run "fpctl OPERATION --help" for information on a specific fpctl operation',
                                     formatter_class=argparse.RawDescriptionHelpFormatter)
    subparsers = parser.add_subparsers(dest='subparser_name')

    sp_deploy = subparsers.add_parser('deploy',
                                      aliases=['de'],
                                      help='deploy FLP prototype software and configuration')
    sp_deploy.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_deploy.add_argument('--verbose', '-v', help=verbose_help, action='store_true')
    sp_deploy.add_argument('--ansible-extra-params', '-p', metavar='ANSIBLE_PARAMS', help=ansible_extra_params_help)
    sp_deploy.add_argument('--ansible-extra-vars', '-e', metavar='ANSIBLE_VARS', help=ansible_extra_vars_help)
    sp_deploy.set_defaults(func=deploy)

    sp_configure = subparsers.add_parser('configure',
                                         aliases=['co'],
                                         help='deploy FLP prototype configuration')
    sp_configure.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_configure.add_argument('--verbose', '-v', help=verbose_help, action='store_true')
    sp_configure.add_argument('--ansible-extra-params', '-p', metavar='ANSIBLE_PARAMS', help=ansible_extra_params_help)
    sp_configure.add_argument('--ansible-extra-vars', '-e', metavar='ANSIBLE_VARS', help=ansible_extra_vars_help)
    sp_configure.set_defaults(func=configure)

    sp_run = subparsers.add_parser('run',
                                   help='run a custom command on one or all nodes')
    sp_run.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_run.add_argument('--verbose', '-v', help=verbose_help, action='store_true')
    sp_run.add_argument('--ansible-extra-params', '-p', metavar='ANSIBLE_PARAMS', help=ansible_extra_params_help)
    sp_run.add_argument('--ansible-extra-vars', '-e', metavar='ANSIBLE_VARS', help=ansible_extra_vars_help)
    sp_run.add_argument('host', metavar='HOST',
                        help='a hostname, an Ansible inventory group, or "all"')
    sp_run.add_argument('command', metavar='COMMAND',
                        help='the command to run on the target node (use quotes '
                             'if it contains whitespace)')
    sp_run.set_defaults(func=run)

    sp_start = subparsers.add_parser('start',
                                     help='start some or all FLP prototype processes')
    sp_start.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_start.add_argument('--verbose', '-v', help=verbose_help, action='store_true')
    sp_start.add_argument('--ansible-extra-params', '-p', metavar='ANSIBLE_PARAMS', help=ansible_extra_params_help)
    sp_start.add_argument('--ansible-extra-vars', '-e', metavar='ANSIBLE_VARS', help=ansible_extra_vars_help)
    sp_start.add_argument('task', metavar='TASK', nargs='?',
                          help='the task to start on the nodes, as configured in the '
                               'inventory file')
    sp_start.set_defaults(func=start)

    sp_status = subparsers.add_parser('status',
                                      help='view status of some or all FLP prototype processes')
    sp_status.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_status.add_argument('--verbose', '-v', help=verbose_help, action='store_true')
    sp_status.add_argument('--ansible-extra-params', '-p', metavar='ANSIBLE_PARAMS', help=ansible_extra_params_help)
    sp_status.add_argument('--ansible-extra-vars', '-e', metavar='ANSIBLE_VARS', help=ansible_extra_vars_help)
    sp_status.add_argument('task', metavar='TASK', nargs='?',
                           help='the task on the nodes for which to query the status, as '
                                'configured in the inventory file')
    sp_status.set_defaults(func=status)

    sp_stop = subparsers.add_parser('stop',
                                    help='stop some or all FLP prototype processes')
    sp_stop.add_argument('--inventory', '-i', metavar='INVENTORY', help=inventory_help)
    sp_stop.add_argument('--verbose', '-v', help=verbose_help, action='store_true')
    sp_stop.add_argument('--ansible-extra-params', '-p', metavar='ANSIBLE_PARAMS', help=ansible_extra_params_help)
    sp_stop.add_argument('--ansible-extra-vars', '-e', metavar='ANSIBLE_VARS', help=ansible_extra_vars_help)
    sp_stop.add_argument('task', metavar='TASK', nargs='?',
                         help='the task to stop on the nodes, as configured in the '
                              'inventory file')
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
        print(C_ERR + 'No operation specified.')
        parser.print_help()
        sys.exit(1)

    parsed_args.func(parsed_args)


if __name__ == '__main__':
    main(sys.argv)
