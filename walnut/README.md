# `walnut` - the O² workflow administration and linting utility

The O² **w**orkflow **a**dministration and **l**i**n**ting **ut**ility is a command line program for workflow configuration
and import validation.

## Configuration file

You can check the local `walnut` configuration with
```bash
$ walnut about
```

## Using `walnut`

At any step, you can type `$ walnut help <subcommand>` to get information on what you can do and how, for example:

```bash
$ walnut help 
walnut is a command line program for interacting with the O² Workflow Administration and Linting Utility.

For more information on the available commands, see the individual documentation for each command.

Usage:
  walnut [command]

Available Commands:
  check       check the file passed against a specified schema.
  convert     Converts a DPL Dump to the required formats.
  help        Help about any command

Flags:
      --config string   optional configuration file for walnut (default $HOME/.config/walnut/settings.yaml)
  -h, --help            help for walnut
  -t, --toggle          Help message for toggle

Use "walnut [command] --help" for more information about a command.
```

### Validating a file

To check if a file conforms to the pre-defined schemas, run the `check` command: 
```bash
$ walnut check
```
Example usage:
```bash
$ walnut check --filename dump.json --format workflow 
```

A successful validation will return no output while any errors will cause the program to exit.

#### SEE ALSO

* [walnut check](./doc/walnut_check.md)


### Converting a DPL Dump to Task/Workflow Templates

The goal of `walnut` is to convert O²/DPL generated dumps to the required number of task templates and workflow templates. The
command used is `convert`:
```bash
$ walnut convert
```

Convert can also receive optional arguments to specify which modules to consider while generating task templates.

Example usage:
```bash
$ walnut convert --filename dump.json --modules TestValue1 TestValue2 
```

A successful validation will return no output while any errors will cause the program to exit.

#### SEE ALSO

* [walnut convert](./doc/walnut_convert.md)