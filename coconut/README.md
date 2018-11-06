# `coconut` - the O² control and configuration utility

The O² **co**ntrol and **con**figuration **ut**ility is a command line program for interacting with the O²
Control core.

## Configuration file

`coconut` can be used with no config file, and by default it will look for a running O² Control
core at `127.0.0.1:47102`.

You can check the local `coconut` configuration with
```bash
$ coconut about
```

If no config file is used, the previous command will print out `config: builtin`.

To override this, create a file `~/.config/coconut/settings.yaml` and fill it in along these lines:
```yaml

---
endpoint: "127.0.0.1:47102"
config_endpoint: "consul://some-host-with-consul:8500" # or: "file:///path/to/o2control-core/config.yaml"
log:
  #values: panic fatal error warning info debug
  level: info
```

## Using `coconut`

`coconut` provides context-sensitive help at every step, including when trying to execute an incomplete command.
The best way to familiarize yourself with it is to simply run `$ coconut`, see what it says and try out
the offered subcommands.

At any step, you can type `$ coconut help <subcommand>` to get information on what you can do and how, for example:

```bash
$ coconut help environment list
The environment list command shows a list of currently active environments.
This includes O² environments in any state.

Usage:
  coconut environment list [flags]

Aliases:
  list, ls, l

Flags:
  -h, --help   help for list

Global Flags:
      --config string            configuration file (default $HOME/.config/coconut/settings.yaml)
      --config_endpoint string   O² Configuration endpoint as PROTO://HOST:PORT (default "consul://127.0.0.1:8500")
      --endpoint string          O² Control endpoint as HOST:PORT (default "127.0.0.1:47102")
  -v, --verbose                  show verbose output for debug purposes

```

Assuming there's a running O² Control core and `coconut` is correctly configured, the following command should
return some details on the O² Control core:

```bash
$ coconut info
O² Control core running on 127.0.0.1:47102
framework id:       1f303909-7beb-4bd2-800d-d71470e211d4-0078
environments count: 0
roles count:        0
global state:       CONNECTED
```

The global state is `CONNECTED`, which is good, because it means the core is up and talking to the
resource management system (Apache Mesos). No environments and roles running yet, fair enough.

### Creating an environment

If you started the core with the provided `config.yaml`, it should come preloaded with some FairMQ examples.
The main subcommand for dealing with environments is (unsurprisingly) `environment`. Most subcommands have
shortened variants, so you might as well type `env` or `e`. Let's see what's running.
```bash
$ coconut env list
no environments running
```
How do we create one? We can always ask `coconut`.
```bash
$ coconut help env create
The environment create command requests from O² Control the
creation of a new O² environment.

Usage:
  coconut environment create [flags]

Aliases:
  create, new, c, n

Flags:
  -h, --help              help for create
  -w, --workflow string   workflow to be loaded in the new environment
# ...
```

Note that if your `coconut` instance is configured correctly to point to the core's configuration (either Consul
or file), you can use the low level `dump` subcommand to list the available workflow templates.

```bash
$ coconut config dump /o2/control/workflows
```

Let's create an environment by loading the workflow template for the FairMQ 1-n-1 example.
This will take a few seconds.
```bash
$ coconut env create -w fairmq-ex-1-n-1
new environment created
environment id:     8132d249-e1b4-11e8-9f09-a08cfdc880fc
state:              CONFIGURED
```

Boom. All environments transition to `CONFIGURED` immediately after creation.
This corresponds to the `READY` state for a FairMQ process, so a lot has already happened behind the scenes.

```bash
$ coconut env list
                   ID                  |         CREATED         |   STATE     
+--------------------------------------+-------------------------+------------+
  8132d249-e1b4-11e8-9f09-a08cfdc880fc | 2018-11-06 12:10:01 CET | CONFIGURED  
```
Take note of the environment ID, as it's the primary key for other environment operations.

We can also check what tasks are currently running.

```bash
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

Let's start the data flow. If all goes well, `START_ACTIVITY` takes us to `RUNNING`.

```bash
$ coconut env control 8132d249-e1b4-11e8-9f09-a08cfdc880fc --event START_ACTIVITY
transition complete
environment id:     8132d249-e1b4-11e8-9f09-a08cfdc880fc
state:              RUNNING
```

We can also query the state of the environment.

```bash
$ coconut env show 8132d249-e1b4-11e8-9f09-a08cfdc880fc
environment id:     8132d249-e1b4-11e8-9f09-a08cfdc880fc
created:            2018-11-06 12:10:01 CET
state:              RUNNING
roles:              fairmq-ex-1-n-1-processor#813a8b57-e1b4-11e8-9f09-a08cfdc880fc, fairmq-ex-1-n-1-processor#813a7311-e1b4-11e8-9f09-a08cfdc880fc, fairmq-ex-1-n-1-processor#813a5e7a-e1b4-11e8-9f09-a08cfdc880fc, fairmq-ex-1-n-1-processor#813a4945-e1b4-11e8-9f09-a08cfdc880fc, fairmq-ex-1-n-1-sampler#813a2cb6-e1b4-11e8-9f09-a08cfdc880fc, fairmq-ex-1-n-1-sink#8138df27-e1b4-11e8-9f09-a08cfdc880fc
```

And then we go back.
```bash
$ coconut e t 8132d249-e1b4-11e8-9f09-a08cfdc880fc -e STOP_ACTIVITY  
transition complete
environment id:     8132d249-e1b4-11e8-9f09-a08cfdc880fc
state:              CONFIGURED
```

As of 11/2018 InfoLogger integration is work in progress, so the best way to check what's up with a specific
task is with the Mesos GUI.
On the DC/OS Vagrant test cluster, this is accessible at [http://m1.dcos/mesos/]().
Pick the correct task by ID, Name, State, etc. and click on *Sandbox* in the rightmost column, and then open
`stderr`.

Environment teardown is also work in progress, so in the short term `pkill` will have to do.
