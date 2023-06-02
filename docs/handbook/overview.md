# Design Overview

AliECS in a distributed application, using Apache Mesos as toolkit. It integrates a task scheduler component, a purpose-built distributed state machine system, a multi-source stateful process configuration mechanism, and a control plugin and library compatible with any data-driven O2 process.

## AliECS Structure

![](AliECS-diag.png)

| Component   | Description |
|-------------|-------------|
| core        | Main AliECS service and API entry point. Runs on the head node. Interfaces with the configuration system, the DCS, the trigger system and user interfaces. |
| executor    | Runs on every FLP node in the cluster. Started as needed by the AliECS core. An executor handles all the controlled processes of a given node. |
| OCC plugin  | FairMQ plugin that translates the FairMQ plugin interface (and state machine) into the O² Control and Configuration interface in order to interact with the executor. |
| OCC libary  | Equivalent of the OCC plugin, but for tasks that aren't based on FairMQ. Provides a task state machine and an O² Control and Configuration endpoint. |
| configuration | A Consul data store, running on the head node on port 8500. Used for AliECS configuration as well as application-specific component configuration. |
| Mesos master | Main service of the Mesos resource management system, running on the head node on port 5050. |
| Mesos agent  | Agent service of the Mesos RMS, running on every FLP. |
| AliECS GUI  | Instance of the user-facing web interface for AliECS (`cog`), running on the head node. This is the main entry point for regular users. |
| AliECS CLI  | The `coconut` command, provided by the package with the same name. This is the reference client for advanced users and developers. |


## Resource Management

Apache Mesos is a cluster resource management system. It greatly streamlines distributed application development by providing a unified distributed execution environment. Mesos facilitates the management of O²/FLP components, resources and tasks inside the O²/FLP facility, effectively enabling the developer to program against the datacenter (i.e., the O²/FLP facility at LHC Point 2) as if it was a single pool of resources.

For AliECS, Mesos acts as an authoritative source of knowledge on the state of the cluster, as well as providing transport facilities for communication between the AliECS core and the executor.

You can view the state of the cluster as presented by Mesos via the Mesos web interface, served on port `5050` of your head node when deployed via the [O²/FLP Suite setup tool](../../installation/).


## FairMQ

The O² project has chosen FairMQ as the common message passing and data transport framework for its data-driven processes. It has been developed in the context of FairRoot, a simulation, reconstruction and analysis framework for particle physics experiments. FairMQ provides the basic building blocks to implement complex data processing workflows, including a message queue, a configuration mechanism, a state machine, and a plugin system.

Thus, when we discuss the state machine of an AliECS-controlled process, we usually refer to the [FairMQ state machine](https://github.com/FairRootGroup/FairMQ/blob/master/docs/Device.md#13-state-machine).

## State machines

The main state machine of AliECS is the environment state machine, which represents the collective state of all the tasks involved in a given data processing activity.

![](AliECS-envsm.svg)

While FairMQ devices use their own, FairMQ-specific state machine, non-FairMQ tasks based on the [OCC library](https://alice-flp-suite.docs.cern.ch/aliecs/occ/) use the same state machine as the AliECS environment state machine, the only difference being that the `START_ACTIVITY` transition is simply `START`, and the `STOP_ACTIVITY` transition is simply `STOP`.

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
* `timeout` - optional, Go `time.Duration` expression, the maximum time `func` will be granted to complete before its context is invalidated.
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
