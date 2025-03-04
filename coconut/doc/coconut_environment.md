## coconut environment

create, destroy and manage AliECS environments

### Synopsis

The environments command allows you to perform operations on environments.

An environment is an instance of a data-driven workflow of tasks, along with its workflow configuration, task configuration and state.

Tasks are logically grouped into roles. Each environment has a distributed state machine, which aggregates the state of its constituent roles and tasks.

An environment can be created, it can be configured and reconfigured multiple times, and it can be started and stopped multiple times.

```
-> STANDBY -(CONFIGURE)-> CONFIGURED -(START_ACTIVITY)-> RUNNING
    |  ↑                   |  |  ↑                        |
    |   ------(RESET)------   |   ----(STOP_ACTIVITY)-----
    |                         |
    |-------------------------
  (EXIT)
    ↓
   DONE
```

If the current state is `RUNNING`, the environment represents a `RUN` and has a run number. This number is only valid until the next `STOP_ACTIVITY` transition, each subsequent `START_ACTIVITY` transition will yield a new run number.

For more information on the behavior of coconut environments, see the subcommands linked below.

### Options

```
  -h, --help   help for environment
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

* [coconut](coconut.md)	 - O² Control and Configuration Utility
* [coconut environment control](coconut_environment_control.md)	 - control the state machine of an environment
* [coconut environment create](coconut_environment_create.md)	 - create a new environment
* [coconut environment destroy](coconut_environment_destroy.md)	 - destroy an environment
* [coconut environment list](coconut_environment_list.md)	 - list environments
* [coconut environment show](coconut_environment_show.md)	 - show environment information

###### Auto generated by spf13/cobra on 27-Nov-2024
