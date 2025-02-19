## coconut repository remove

remove a git repository

### Synopsis

The repository remove command removes a git repository from the catalogue of workflow configuration sources.
A repository is referenced by its repo id, as reported by`coconut repo list`

```
coconut repository remove <repo id> [flags]
```

### Examples

```
 * `coconut repo remove 1`
 * `coconut repo del 2`
```

### Options

```
  -h, --help   help for remove
```

### Options inherited from parent commands

```
      --config string            optional configuration file for coconut (default $HOME/.config/coconut/settings.yaml)
      --config_endpoint string   configuration endpoint used by AliECS core as PROTO://HOST:PORT (default "apricot://127.0.0.1:32101")
      --endpoint string          AliECS core endpoint as HOST:PORT (default "127.0.0.1:32102")
      --nocolor                  disable colors in output
      --nospinner                disable animations in output
  -v, --verbose                  show verbose output for debug purposes
```

### SEE ALSO

* [coconut repository](coconut_repository.md)	 - manage git repositories for task and workflow configuration

###### Auto generated by spf13/cobra on 27-Nov-2024
