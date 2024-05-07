# Workflow Configuration

## Non-critical tasks

Any task in a workflow can be declared as non-critical. A non-critical task is a task that doesn't trigger a global environment ERROR in case of failure. The state of a non-critical task doesn't affect the environment state in any way.

To declare a task as non-critical, a line has to be added in the task role block within a workflow template file. Specifically, in the task section of such a task role (usually after the `load` statement), the line to add is `critical: false`, like in the following example:

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

## State machine callbacks

The underlying state machine library allows us to add callbacks upon entering and leaving states as well as before and after events (transitions).
This is the order of callback execution upon a state transition:
1. `before_<EVENT>` - called before event named `<EVENT>`
2. `before_event` - called before all events
3. `leave_<OLD_STATE>` - called before leaving `<OLD_STATE>`
4. `leave_state` - called before leaving all states
5. `enter_<NEW_STATE>`, `<NEW_STATE>` - called after entering `<NEW_STATE>`
6. `enter_state` - called after entering all states
7. `after_<EVENT>`, `<EVENT>` - called after event named `<EVENT>`
8. `after_event` - called after all events

Callback execution is further refined with integer indexes, with the syntax `±index`, e.g. `before_CONFIGURE+2`, `enter_CONFIGURED-666`. An expression with no index is assumed to be indexed `+0`. These indexes do not correspond to timestamps, they are discrete labels that allow more granularity in callbacks, ensuring a strict ordering of callback opportunities within a given callback moment. Thus, `before_CONFIGURE+2` will complete execution strictly after `before_CONFIGURE` runs, but strictly before `enter_CONFIGURED-666` is executed.

## Workflow hook calls

The state machine callback moments are exposed to the AliECS workflow template interface and can be used as triggers or synchronization points for integration plugin function calls. The `call` block can be used for this purpose, with similar syntax to the `task` block used for controllable tasks. Its fields are as follows.
* `func` - mandatory, it parses as an [`antonmedv/expr`](https://github.com/antonmedv/expr) expression that corresponds to a call to a function that belongs to an integration plugin object (e.g. `bookkeeping.StartOfRun()`, `dcs.EndOfRun()`, etc.).
* `trigger` - mandatory, the expression at `func` will be executed once the state machine reaches this moment.
* `await` - optional, if absent it defaults to the same as `trigger`, the expression at `func` needs to finish by this moment, and the state machine will block until `func` completes.
* `timeout` - optional, Go `time.Duration` expression, defaults to `30s`, the maximum time `func` will be granted to complete before its context is invalidated.
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

# Task Configuration

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

FairMQ task implementors should expect that these values are written to the FairMQ properties map right before the `RUN` transition via `SetProperty` calls.

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
