#!/usr/bin/env python3


# add to ANSIBLE_CONFIG env variable:
# control_path_dir=/tmp/.ansible/cp/
# host_key_checking=False

import argparse
import logging
import os
import subprocess
import sys

FPCTL_CONFIG_DIR = os.path.expanduser('~/.config/fpctl')
FPCTL_DATA_DIR = os.path.expanduser('~/.local/share/fpctl')

def bail(description, exit_code = 1):
    """Report a fatal error and exit immediately"""
    print('ERROR: {}\n'.format(description))
    print('fpctl will now quit ({}).'.format(exit_code))
    sys.exit(exit_code)

def deploy(args):
    """Handler for deploy command"""
    print("Not implemented yet :(\nCalled {}".format(vars(args)))
    #TODO: for commands that do remote operations, go through a check for SSH passwordless
    #      authentication and/or sudoers

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
