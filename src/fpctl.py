#!/usr/bin/env python3


# add to ANSIBLE_CONFIG env variable:
# control_path_dir=/tmp/.ansible/cp/
# host_key_checking=False

import sys
import argparse

def main(argv):
    """Entry point, called by fpctl script."""
    args = argv[1:]
    print("fpctl args: {}".format(", ".join(args)))

    parser = argparse.ArgumentParser(description='FLP prototype control utility')
    subparsers = parser.add_subparsers(dest='subparser_name')

    sp_update = subparsers.add_parser('update',
                                      aliases=['up'],
                                      help='update fpctl deployment information')
    sp_update.set_defaults(func=parser.print_help)

    sp_deploy = subparsers.add_parser('deploy',
                                      aliases=['de'],
                                      help='deploy FLP prototype software and configuration')
    sp_deploy.add_argument('--inventory', '-i', metavar='path_to_inventory_file')

    sp_configure = subparsers.add_parser('configure',
                                         aliases=['co'],
                                         help='deploy FLP prototype configuration')
    sp_configure.add_argument('--inventory', '-i', metavar='path_to_inventory_file')

    sp_run = subparsers.add_parser('run',
                                   help='run a custom command on one or all nodes')
    sp_run.add_argument('--inventory', '-i', metavar='path_to_inventory_file')

    sp_start = subparsers.add_parser('start',
                                     help='start some or all FLP prototype processes')
    sp_start.add_argument('--inventory', '-i', metavar='path_to_inventory_file')

    sp_status = subparsers.add_parser('status',
                                      help='view status of some or all FLP prototype processes')
    sp_status.add_argument('--inventory', '-i', metavar='path_to_inventory_file')

    sp_stop = subparsers.add_parser('stop',
                                    help='stop some or all FLP prototype processes')
    sp_stop.add_argument('--inventory', '-i', metavar='path_to_inventory_file')

    sp_log = subparsers.add_parser('log',
                                   help='view the logs of some or all FLP prototype processes')
    sp_log.add_argument('--inventory', '-i', metavar='path_to_inventory_file')

    parsed_args = parser.parse_args(args)
    print('argparse output: {}'.format(vars(parsed_args)))
    parsed_args.func(parsed_args)
    # note to self: action='store_true' for arguments with no value


if __name__ == '__main__':
    main(sys.argv)
