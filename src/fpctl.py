#!/usr/bin/env python3


# add to ANSIBLE_CONFIG env variable:
# control_path_dir=/tmp/.ansible/cp/
# host_key_checking=False

import sys

def main(argv):
    """Entry point, called by fpctl script."""
    args = argv[1:]


    print("fpctl args: {}".format( ", ".join(args)))
    sys.exit(1)

if __name__ == '__main__':
    main(sys.argv)
