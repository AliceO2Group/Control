# FlpPrototype control wrapper design

* Initial instructions for installing the script would include wget bash script into a bin
  path (like /usr/local/bin), or into ~/bin and adding this to PATH.
* Initial shell script fpctl: if python3 exists and if repos are set up, call
  ~/.local/fpctl/Control/fpctl.py. Therefore, fpctl is always the entry point.
  Else, if any of the requirements for running the actual Python wrapper are not met, do
  stuff like installing python34, git, ansible, etc., git clone Control and
  system-configuration, creating ~/.config/fpctl, ...
* Instruct people to add their SSH keys to gitlab.cern.ch in order for the script to work.
  On error bail out (this should only happen on systems other than CC7).
* The inventory file is assumed to be ~/.config/fpctl/inventory, this can be overridden
  as -i /path/to/inventory.
* Syntax ideas:
  * fpctl update/up
  * fpctl deploy/de
  * fpctl configure/co
  * fpctl run [machinename] [command]   // wrapper for ansible interactive command
  * fpctl start [taskname]
  * fpctl status [taskname]
  * fpctl stop [taskname]
  * fpctl log [taskname]
  * fpctl help
* Also do documentation for inventory.
