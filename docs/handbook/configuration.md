# Workflow Configuration

## The AliECS workflow template language
The AliECS workflow template language is a YAML-based language that allows
the user to define the structure of a data taking and processing activity.
The language is used to define the tasks that are part of the activity,
the data flow relationships between them, the behaviour of integrated services
with respect to the data taking state machine, and the conditions that
trigger these integrated service actions.
The language is designed to be human-readable and easy to understand,
while still being powerful enough to express complex workflows.

To instantiate a data taking activity, or environment, two kinds of files
are needed:
* workflow templates
* task templates

Both kinds of files can be places in a git repository (by convention,
they must be places in their own directories named `workflows` and
`tasks` respectively), and AliECS can pull them directly from there.
This allows for version control and collaboration on the workflow
and task definitions.

See [the ControlWorkflows repository](https://github.com/AliceO2Group/ControlWorkflows/)
for examples of workflow and task templates and their structure.
Also see [its README](https://github.com/AliceO2Group/ControlWorkflows/blob/master/README.md)
for information on specific variables and their meaning, as well as for
the DPL subworkflow loading system.

## Workflow template structure

A workflow template is a YAML file that contains a tree structure whose nodes
are called roles. This structure can be deeply nested, and each role can have
a set of variables that define its behaviour and that of its child roles.

The root of the tree is the root role, which is the top level role in the
workflow template.

All roles have a mandatory `name` attribute. The root role also has a
`description`.

There are five kinds of roles in a workflow template:

- Task roles: These roles represent tasks that are part of the workflow.
They must contain a `task` attribute.
- Call roles: These roles represent calls to integrated services.
They must contain a `call` attribute.
- Aggregator roles: These roles represent aggregations of other roles.
They must contain a `roles` attribute.
- Iterator roles: These roles expand into multiple instances based on
an iterator expression.
They must contain a `for` attribute, as well as a `roles` attribute.
Additionally, their `name` must be parametrized with the iterator variable
specified in the `for` block.
- Include roles: These roles include another workflow template as subtree.
They must contain an `include` attribute.

Task, call and include roles may only appear as leaves in the tree,
while aggregator and iterator roles may not be leaves, and instead act as
containers of child roles.

All roles may have an `enabled` attribute, which is a boolean that
determines whether the role is enabled or not. If a role is not enabled,
it is excluded from the workflow along with its children.

All roles may also have `defaults` and `vars` attributes. Both `defaults`
and `vars` are key-value maps. The `defaults` map is used to set default
values, and values in `vars` override any `defaults` with the same key.
Values set in `defaults` also act as defaults for child roles, and
values set in `vars` also act as vars for child roles.
User-provided parameters further override anything set in `defaults` or
`vars`.

All roles may have one or more `constraints` expressions, which restrict
the deployment of the role (or it child roles) to nodes that satisfy the
constraints. The constraints are specified as a list of expressions that
evaluate to true or false. The expressions are evaluated against the Mesos
attributes set on the nodes in the cluster.

### Task roles

Task roles represent tasks that are part of the workflow. They must contain
a `task` attribute, which contains a key that maps to a task template (i.e.
a distinct YAML file that defines how to run that specific task).

There are two kinds of task roles: data flow task roles and hook task roles.

Data flow task roles represent tasks that are part of the data flow of the
workflow. They usually contain attributes such as `bind` and `connect` that
define the data flow relationships between tasks. Besides `load`, which
references a task template, and `critical`, which determines whether the
task is critical to the data taking activity, they do not contain other
attributes under the `task` key.

```yaml
- name: "stfb"
  enabled: "{{stfb_standalone == 'false'}}"
  vars:
    dd_discovery_stfb_id: stfb-{{ it }}-{{ uid.New() }}
  connect:
    - name: readout
      type: pull
      target: "{{ Up(2).Path }}.readout:readout"
      rateLogging: "{{ fmq_rate_logging }}"
  bind:
    - name: dpl-chan
      type: push
      rateLogging: "{{ fmq_rate_logging }}"
      transport: shmem
      addressing: ipc
      sndBufSize: "4"
      global: "readout-proxy-{{ it }}"
  task:
    load: stfbuilder
```

Hook task roles represent tasks that are not part of the data flow, but
instead are called at specific points in the environment state machine.
They have a well-defined moment when they must start and finish (with
respect to the environment state machine), and they are generally not
long-running tasks. Like data flow task roles, they may be `critical`.
They do not contain `bind` or `connect` attributes, and their `task`
attribute, besides `load`, contains additional attributes that define
the timing of the task. `trigger` is the moment when the task must start,
and `timeout` is the maximum time the task is allowed to run. Optionally,
`await` may be specified in addition to `trigger`, in which case the task
must finish by `await`. If `await` is not specified, it defaults to the
value of `trigger`, i.e. the task must start and finish within the same
state machine moment.

For more information on the values of `trigger` and `await`, see below.

```yaml
- name: fairmq-shmcleanup
  enabled: "{{fmq_initial_shm_cleanup_enabled == 'true'}}"
  vars:
    shell_command: "source /etc/profile.d/o2.sh && O2_PARTITION={{environment_id}} O2_ROLE={{it}} o2-aliecs-shmcleaner"
    user: root
  task:
    load: "shell-command"
    trigger: before_DEPLOY
    timeout: "{{ fmq_initial_shm_cleanup_timeout }}"
    critical: false
```

#### Non-critical tasks

Any task in a workflow can be declared as non-critical. A non-critical
task is a task that doesn't trigger a global environment ERROR in case
of failure. The state of a non-critical task doesn't affect the
environment state in any way.

To declare a task as non-critical, a line has to be added in the task
role block within a workflow template file. Specifically, in the task
section of such a task role (usually after the `load` statement), the
line to add is `critical: false`, like in the following example:

```yaml
roles:
  - name: "non-critical-task"
    vars:
      non-critical-task-var: 'var-value'
    task:
      load: mytask
      critical: false
```

In the absence of an explicit `critical` trait for a given task role, the assumed default value is `critical: true`.

### Call roles

Call roles represent calls to integrated services. They must contain a `call`
attribute, which contains a key that maps to an integration plugin function,
i.e. an API call that is made to an integrated service.

The `call` map must contain a `func` key, which references the function to be
called. The functions available depend on which integration plugins are
loaded into the AliECS instance.
Like hook task roles, call roles have a well-defined moment when they must start
and finish (with respect to the environment state machine), and they are generally
not long-running operations.

```yaml
- name: "reset"
  call:
    func: odc.Reset()
    trigger: before_RESET
    await: after_RESET
    timeout: "{{ odc_reset_timeout }}"
    critical: true
```

See [readout-dataflow](https://github.com/AliceO2Group/ControlWorkflows/blob/master/workflows/readout-dataflow.yaml)
for examples of call roles that reference a variety of integration plugins.

#### Workflow hook call structure

The state machine callback moments are exposed to the AliECS workflow template interface and can be used as triggers or synchronization points for integration plugin function calls. The `call` block can be used for this purpose, with similar syntax to the `task` block used for controllable tasks. Its fields are as follows.
* `func` - mandatory, it parses as an [`antonmedv/expr`](https://github.com/antonmedv/expr) expression that corresponds to a call to a function that belongs to an integration plugin object (e.g. `bookkeeping.StartOfRun()`, `dcs.EndOfRun()`, etc.).
* `trigger` - mandatory, the expression at `func` will be executed once the state machine reaches this moment. For possible values, see [State machine triggers](/docs/handbook/operation_order.md#state-machine-triggers)
* `await` - optional, if absent it defaults to the same as `trigger`, the expression at `func` needs to finish by this moment, and the state machine will block until `func` completes.
* `timeout` - optional, Go `time.Duration` expression, defaults to `30s`, the maximum time that `func` should take. The value is provided to the plugin via `varStack["__call_timeout"]` and the plugin should implement a timeout mechanism. The ECS will not abort the call upon reaching the timeout value!
* `critical` - optional, it defaults to `true`, if `true` then a failure or timeout for `func` will send the environment state machine to `ERROR`.

Consider the following example:
```
# Trigger and await are different: any number of other operations may happen concurrently in between. Regardless of when in time the call actually finishes, its result isn't collected until the environment state machine reaches `after_RESET+0`. The state machine will only block if it reaches `after_RESET+0` and the call isn't done yet (completed or timed out).
      - name: reset
        call:
          func: odc.Reset()
          trigger: before_RESET
          await: after_RESET
          timeout: "{{ odc_reset_timeout }}"
          critical: true
# Trigger and await are the same (the await expression could be omitted here): the call must begin and end within the `after_RESET+100` step. If the workflow template defines no other calls straddling `after_RESET+100` then this call is fully serialized with respect to the state machine and its execution blocks everything else.
      - name: part-term
        call:
          func: odc.PartitionTerminate()
          trigger: after_RESET+100
          await: after_RESET+100
          timeout: "{{ odc_partitionterminate_timeout }}"
          critical: true
```

### Aggregator roles

Aggregator roles represent aggregations of other roles. They must contain a
`roles` attribute, which is a list of child roles.

For the purposes of the state machine, they represent their children, and
any `defaults` or `vars` set on an aggregator role are passed down to its
children (which may in turn override them).

```yaml
- name: "readout"
  vars:
    readout_var: 'this value will be overridden by the 1st child role'
  roles:
    - name: "readout"
      vars:
        readout_var: 'var-value'
      task:
        load: readout
    - name: "stfb"
      vars:
        stfb_var: 'var-value'
      task:
        load: stfbuilder
```

### Iterator roles

Iterator roles expand into multiple instances based on an iterator expression.
They must contain a `for` attribute, which is an expression that evaluates to
a list of values. The `name` attribute must be parametrized with the iterator
variable specified in the `for` block.

```yaml
- name: host-{{ it }}
  for:
    range: "{{ hosts }}"
    var: it
  constraints:
    - attribute: machine_id
      value: "{{ it }}"
  roles:
    - name: "readout"
      task:
        load: readout
```

### Include roles

Include roles include another workflow template as subtree. They must contain
an `include` attribute, which is the path to the workflow template file to
include.

```yaml
- name: dpl
  enabled: "{{ qcdd_enabled == 'true' }}"
  include: qc-daq
```

### Template expressions

The AliECS workflow template language supports expressions in the form of
`{{ expression }}`. These expressions are evaluated by the AliECS core
when the workflow is instantiated, and the result is used in place of the
expression.

See [`antonmedv/expr`](https://github.com/antonmedv/expr) for the full
documentation on the expression syntax.

AliECS extends the syntax with
additional functions and variables that are available in the context of
the workflow template evaluation.

#### Configuration access functions

* `config.Get(path string) string` - Returns the template-processed configuration payload at the given Apricot path.
* `config.Resolve(component string, runType string, roleName string, entryKey string) string` - Returns the resolved path to a configuration entry for the given component, run type, role name, and entry key.
* `config.ResolvePath(path string) string` - Returns the resolved path to a configuration entry for the given path.

#### Inventory access functions

* `inventory.DetectorForHost(hostname string) string` - Returns the detector name for the specified host.
* `inventory.DetectorsForHosts(hosts string) string` - Returns a JSON-format list of detector names for the specified list of hosts (also expected to be JSON-format).
* `inventory.CRUCardsForHost(hostname string) string` - Returns a JSON-format list of CRUs for the specified host.
* `inventory.EndpointsForCRUCard(hostname string, cardSerial string) string` - Returns a JSON-format list of endpoints for the specified CRU card.

#### Runtime KV map access functions

* `runtime.Get(component string, key string) string` - Returns from Apricot the value of the key in the runtime KV map of the specified component.
* `runtime.Set(component string, key string, value string) string` - Sets in Apricot the value of the key into the runtime KV map of the specified component.

#### DPL subworkflow just-in-time generator functions

* `dpl.Generate`
* `dpl.GenerateFromUri`
* `dpl.GenerateFromUriOrFallbackToTemplate`

#### String functions

* `strings.Atoi`, `strings.Itoa`, `strings.TrimQuotes`, `strings.TrimSpace`, `strings.ToUpper`, `strings.ToLower` - See the [Go strings package](https://golang.org/pkg/strings/) for more information.
* `strings.IsTruthy(in string) bool` - Used in condition evaluation. Returns `true` if the string is one of `"true"`, `"yes"`, `"y"`, `"1"`, `"on"`, `"ok"`, otherwise `false`.
* `strings.IsFalsy(in string) bool` - Used in condition evaluation. Returns `true` if the string is empty, or one of `"false"`, `"no"`, `"n"`, `"0"`, `"off"`, `"none"`, otherwise `false`.

#### JSON manipulation functions

* `json.Unmarshal(in string) object` (with alias `json.Deserialize`) - Unmarshals a JSON string into an object.
* `json.Marshal(in object) string` (with alias `json.Serialize`) - Marshals an object into a JSON string.

#### UID generation function

* `uid.New() string` - Returns a new unique identifier string, same format as AliECS environment IDs.

#### General utility functions

* `util.PrefixedOverride(varname string, prefix string) string` - Looks in the current variables stack for a variable with key `varname`, as well as for a variable with key `prefix_varname`. If the latter exists, it returns its value, otherwise it returns the value of `varname` as fallback. If neither exist, it returns `""`. Note that this function may return either the empty string or other falsy values such as `"none"`, so `strings.IsFalsy` should be used to check the output if used in a condition.
* `util.Dump(in string, filepath string) string` - Dumps the input string to a file at the specified path. Returns the string itself.
* `util.SuffixInRange(input string, prefix string, idMinStr string, idMaxStr string) string`

# Task Configuration

## Task template structure

A task template is a YAML file that describes the configuration of a task,
down to the command line arguments and environment variables that are passed
to the task on startup.

These parameters and variables can be static, or they can be dynamic, pulled
from the GUI, the AliECS `vars` and `defaults` defined in the workflow template, 
or from the O² Configuration defaults (in order of importance, from less to more
"defaulty").

A task template must contain a `name` attribute, which is the name of the task,
that is then referenced by a task role in a workflow template.

Task templates can define non-data flow tasks, in which case they only specify
the command to run (for the most part), or they can be data flow tasks, in which
case they also specify the available inbound connections with a `bind` statement.

Data flow tasks can also specify additional parameters in a `properties` map,
which are set during the `CONFIGURE` transition (via the FairMQ plugin interface
or via the OCC library, depending on the task control mechanism).

```yaml
name: readout
defaults:
  readout_cfg_uri: "consul-ini://{{ consul_endpoint }}/o2/components/readout/ANY/any/readout-standalone-{{ task_hostname }}"
  user: flp
  log_task_stdout: none
  log_task_stderr: none
  _module_cmdline: >-
    source /etc/profile.d/modules.sh && MODULEPATH={{ modulepath }} module load Readout Control-OCCPlugin &&
    o2-readout-exe
  _plain_cmdline: "{{ o2_install_path }}/bin/o2-readout-exe"
control:
  mode: direct
wants:
  cpu: 0.15
  memory: 128
bind:
  - name: readout
    type: push
    rateLogging: "{{ fmq_rate_logging }}"
    addressing: ipc
    transport: shmem
properties: {}
command: 
  stdout: "{{ log_task_stdout }}"
  stderr: "{{ log_task_stderr }}"
  shell: true
  env:
    - O2_DETECTOR={{ detector }}
    - O2_PARTITION={{ environment_id }}
  user: "{{ user }}"
  arguments:
    - "{{ readout_cfg_uri }}"
  value: "{{ len(modulepath)>0 ? _module_cmdline : _plain_cmdline }}"
```

## Variables pushed to controlled tasks

FairMQ and non-FairMQ tasks may receive configuration values from a variety of sources, both from their own user code (for example by querying Apricot with or without the O² Configuration library) as well as via AliECS.

Variables whose availability to tasks is handled in some way by AliECS include

 * variables pushed via the JIT mechanism to DPL devices
 * variables delivered to tasks explicitly via task templates.

The latter can be
 * sourced from Apricot with a query from the task template iself (e.g. `config.Get`), or
 * sourced from the variables available to the current AliECS environment, as defined in the workflow template (e.g. readout-dataflow.yaml)

Depending on the specification in the task template (`command.env`, `command.arguments` or `properties`), the push to the given task can happen
 * as system environment variables on task startup,
 * as command line parameters on task startup, or
 * as (FairMQ) key-values during `CONFIGURE`.

In addition to the above, which varies depending on the configuration of the environment itself as well as on the configuration of the system as a whole, some special values are pushed by AliECS itself during `START_ACTIVITY`:

 * `runNumber`
 * `fill_info_fill_number`
 * `fill_info_filling_scheme`
 * `fill_info_beam_type`
 * `fill_info_stable_beam_start_ms`
 * `fill_info_stable_beam_end_ms`
 * `run_type`
 * `run_start_time_ms`
 * `lhc_period`
 * `fillInfoFillNumber`
 * `fillInfoFillingScheme`
 * `fillInfoBeamType`
 * `fillInfoStableBeamStartMs`
 * `fillInfoStableBeamEndMs`
 * `runType`
 * `runStartTimeMs`
 * `lhcPeriod`
 * `pdp_beam_type`
 * `pdp_override_run_start_time`

The following values are pushed by AliECS during `STOP_ACTIVITY`:
 * `run_end_time_ms`

FairMQ task implementors should expect that these values are written to the FairMQ properties map right before the `RUN` and `STOP` transitions via `SetProperty` calls.

## Resource wants and limits

All task templates allow two top-level blocks with identical syntax: `wants` and `limits`. They are used to specify respectively the minimum claimed resources that the task will request from Mesos, and the maximum resource allowance which, if exceeded, will result in the task being killed.

Of these two blocks, `wants` is mandatory, and the absence of a `limits` block assumes unlimited resource usage is allowed for tasks generated from this template.

Resource types currently supported are `cpu` and `memory`. Both are of type `float`, and they represent respectively the number (or fraction) of CPU cores, and the amount of memory (including physical and swap) always expressed in MB (see example of a task template file below).

```
name: readout
wants:
  cpu: 0.15      # 15% of one CPU core
  memory: 128    # 128 MB
limits:
  memory: 8192   # 8 GB, all tasks on this machine will be killed if exceeded; cpu unlimited
defaults:
  (...)
```
