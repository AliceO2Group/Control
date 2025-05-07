# The O² control and configuration utility overview

The O² **co**ntrol and **con**figuration **ut**ility is a command line program for interacting with the AliECS core.

## Configuration file

`coconut` can be used with no config file, and by default it will look for a running AliECS
core at `127.0.0.1:32102`.

On `aliBuild` it is provided by the `coconut` recipe (RPM `alisw-coconut`, `alienv`/`aliswmod enter coconut`). Once built, it is a portable, static executable that can safely be copied to another Linux machine and executed.

You can check the local `coconut` configuration with
```
$ coconut about
```

If no config file is used, the previous command will print out `config: builtin`.

To override this, you may create a file `~/.config/coconut/settings.yaml` and fill it as follows:
```yaml

---
endpoint: "127.0.0.1:32102"                            # host:port of AliECS core
config_endpoint: "consul://some-host-with-consul:8500" # or: "file:///path/to/o2control-core/config.yaml", AliECS configuration endpoint
log:
  level: info                                          # values: panic fatal error warning info debug
verbose: false                                         # set to true to debug coconut
nospinner: false                                       # set to true if calling coconut from a script
nocolor: false                                         # set to true if calling coconut from a script
```

## Using `coconut`

`coconut` provides context-sensitive help at every step, including when trying to execute an incomplete command.
The best way to familiarize yourself with it is to simply run `$ coconut`, see what it says and try out
the offered subcommands.

At any step, you can type `$ coconut help <subcommand>` to get information on what you can do and how, for example:

```
$ coconut help environment list
The environment list command shows a list of currently active environments.
This includes environments in any state.

Usage:
  coconut environment list [flags]

Aliases:
  list, ls, l

Flags:
  -h, --help   help for list

Global Flags:
      --config string            optional configuration file for coconut (default $HOME/.config/coconut/settings.yaml)
      --config_endpoint string   configuration endpoint used by AliECS core as PROTO://HOST:PORT (default "consul://127.0.0.1:8500")
      --endpoint string          AliECS core endpoint as HOST:PORT (default "127.0.0.1:32102")
      --nospinner                disable animations in output
      --nocolor                  disable colors in output
  -v, --verbose                  show verbose output for debug purposes
```

Assuming there's a running AliECS core and `coconut` is correctly configured, the following command should
return some details on the AliECS core:

```
$ coconut info
instance name:      AliECS instance
endpoint:           127.0.0.1:32102
core version:       AliECS 0.16.0 revision 977208f
framework id:       fde7f033-0aaf-4d02-9f4e-9ee5ee5824e3-0000
environments count: 0
active tasks count: 0
global state:       CONNECTED
```

The global state is `CONNECTED`, which is good, because it means the core is up and talking to the
resource management system (Apache Mesos). No environments and roles running yet, fair enough.

### Creating an environment

Assuming AliECS was deployed as O²/FLP Suite, FairMQ examples should be available.
The main subcommand for dealing with environments is `environment`. Most subcommands have
shortened variants, so you might as well type `env` or `e`. Let's see what's running.
```
$ coconut env list
no environments running
```
How do we create one? We can always ask `coconut` for a detailed overview.
```
$ coconut help env create
The environment create command requests from AliECS the
creation of a new environment.

The operation may or may not be successful depending on available resources and configuration.

A valid workflow template (sometimes called simply "workflow" for brevity) must be passed to this command via the mandatory workflow-template flag.

Workflows and tasks are managed with a git based configuration system, so the workflow template may be provided simply by name or with repository and branch/tag/hash constraints.
Examples:
 * `coconut env create -w myworkflow` - loads workflow `myworkflow` from default configuration repository at HEAD of master branch
 * `coconut env create -w github.com/AliceO2Group/MyConfRepo/myworkflow` - loads a workflow from a specific git repository, HEAD of master branch
 * `coconut env create -w myworkflow@rev` - loads a workflow from default repository, on branch, tag or revision `rev`
 * `coconut env create -w github.com/AliceO2Group/MyConfRepo/myworkflow@rev` - loads a workflow from a specific git repository, on branch, tag or revision `rev`

For more information on the AliECS workflow configuration system, see documentation for the `coconut repository` command.

Usage:
  coconut environment create [flags]

Aliases:
  create, new, c, n

Flags:
  -e, --extra-vars key1=val1,key2=val2   values passed using key=value CSV or JSON syntax, interpreted as strings key1=val1,key2=val2 or `{"key1": "value1", "key2": "value2"}`
  -h, --help                             help for create
  -w, --workflow-template string         workflow to be loaded in the new environment

Global Flags:
      --config string            optional configuration file for coconut (default $HOME/.config/coconut/settings.yaml)
      --config_endpoint string   configuration endpoint used by AliECS core as PROTO://HOST:PORT (default "consul://127.0.0.1:8500")
      --endpoint string          AliECS core endpoint as HOST:PORT (default "127.0.0.1:32102")
      --nospinner                disable animations in output
      --nocolor                  disable colors in output
  -v, --verbose                  show verbose output for debug purposes
```

Note that if your `coconut` instance is configured correctly to point to the core's configuration (either Consul
or file), you can use the `coconut template list` command to view the available workflow templates. As usual, see
`coconut help template list` for a detailed explanation of the query syntax.

```
$ coconut template list
Available templates in loaded configuration:
github.com/AliceO2Group/ControlWorkflows/
└── [revision] flp-suite-v0.9.0
    ├── odc-shim
    ├── readout-qc
    ├── readout-stfb-qc
    ├── readout-stfb-stfs-odc
    ├── readout-stfb-stfs
    ├── readout-stfb
    └── readout
```

Let's create an environment by loading the workflow template for the FairMQ 1-n-1 example.
This will take a few seconds.
```
$ coconut env create -w fairmq-ex-1-n-1@master
new environment created
environment id:     8132d249-e1b4-11e8-9f09-a08cfdc880fc
state:              CONFIGURED
```

Boom. All environments transition to `CONFIGURED` immediately after creation.
This corresponds to the `READY` state for a FairMQ process, so a lot has already happened behind the scenes.

```
$ coconut env list
                   ID                  |         CREATED         |   STATE     
+--------------------------------------+-------------------------+------------+
  8132d249-e1b4-11e8-9f09-a08cfdc880fc | 2018-11-06 12:10:01 CET | CONFIGURED  
```
Take note of the environment ID, as it's the primary key for other environment operations.

We can also check what tasks are currently running.

```
$ coconut role list
                               NAME                              |    HOSTNAME    | LOCKED  
+----------------------------------------------------------------+----------------+--------+
  fairmq-ex-1-n-1-processor#813a7311-e1b4-11e8-9f09-a08cfdc880fc | 192.168.65.131 | true    
  fairmq-ex-1-n-1-processor#813a8b57-e1b4-11e8-9f09-a08cfdc880fc | 192.168.65.131 | true    
  fairmq-ex-1-n-1-sink#8138df27-e1b4-11e8-9f09-a08cfdc880fc      | 192.168.65.131 | true    
  fairmq-ex-1-n-1-sampler#813a2cb6-e1b4-11e8-9f09-a08cfdc880fc   | 192.168.65.131 | true    
  fairmq-ex-1-n-1-processor#813a4945-e1b4-11e8-9f09-a08cfdc880fc | 192.168.65.131 | true    
  fairmq-ex-1-n-1-processor#813a5e7a-e1b4-11e8-9f09-a08cfdc880fc | 192.168.65.131 | true    
```

### Controlling an environment

Let's start the data flow. If all goes well, `START_ACTIVITY` (or `start`) takes us to `RUNNING`.

```
$ coconut env control 8132d249-e1b4-11e8-9f09-a08cfdc880fc --event START_ACTIVITY
transition complete
environment id:     8132d249-e1b4-11e8-9f09-a08cfdc880fc
state:              RUNNING
```

We can also query the state of the environment.

```
$ coconut env show 8132d249-e1b4-11e8-9f09-a08cfdc880fc
environment id:     8132d249-e1b4-11e8-9f09-a08cfdc880fc
created:            2018-11-06 12:10:01 CET
state:              RUNNING
roles:              fairmq-ex-1-n-1-processor#813a8b57-e1b4-11e8-9f09-a08cfdc880fc, fairmq-ex-1-n-1-processor#813a7311-e1b4-11e8-9f09-a08cfdc880fc, fairmq-ex-1-n-1-processor#813a5e7a-e1b4-11e8-9f09-a08cfdc880fc, fairmq-ex-1-n-1-processor#813a4945-e1b4-11e8-9f09-a08cfdc880fc, fairmq-ex-1-n-1-sampler#813a2cb6-e1b4-11e8-9f09-a08cfdc880fc, fairmq-ex-1-n-1-sink#8138df27-e1b4-11e8-9f09-a08cfdc880fc
```

And then we go back.
```
$ coconut e t 8132d249-e1b4-11e8-9f09-a08cfdc880fc -e STOP_ACTIVITY  
transition complete
environment id:     8132d249-e1b4-11e8-9f09-a08cfdc880fc
state:              CONFIGURED
```
