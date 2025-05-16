# Environment operation order

This chapter attempts to document the order of important operations done during environment transitions.
Since AliECS is an evolving system, the information presented here might be out-of-date, thus please refer to event handling in [core/environment/environment.go](https://github.com/AliceO2Group/Control/blob/master/core/environment/environment.go) and plugin calls in [ControlWorkflows/workflows/readout-dataflow.yaml](https://github.com/AliceO2Group/ControlWorkflows/blob/master/workflows/readout-dataflow.yaml) for the ultimate source of truth.
Also, please report to the ECS developers any inaccuracies.

## State machine triggers

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

Callback execution is further refined with integer indexes, with the syntax `±index`, e.g. `before_CONFIGURE+2`, `enter_CONFIGURED-666`.
An expression with no index is assumed to be indexed `+0`. These indexes do not correspond to timestamps, they are discrete labels that allow more granularity in callbacks, ensuring a strict ordering of callback opportunities within a given callback moment.
Thus, `before_CONFIGURE+2` will complete execution strictly after `before_CONFIGURE` runs, but strictly before `enter_CONFIGURED-666` is executed.

## START_ACTIVITY (Start Of Run)

This is the order of actions happening at a healthy start of run.

### before_START_ACTIVITY

- `before_START_ACTIVITY` hooks with negative weights are executed:
  - `trg.PrepareForRun()` at `-200`
- `"run_number"` is set.
- `"run_start_time_ms"` is set using the current time. It is considered as the SOR and SOSOR timestamps.
- `before_START_ACTIVITY` hooks with positive weights (incl. 0) are executed:
  - `trg.RunLoad()`, `bookkeeping.StartOfRun()` at `10`
  - `bookkeeping.RetrieveFillInfo()` at `11`
  - `kafka.PublishStartActivityUpdate()` at `50`
  - `dcs.StartOfRun()`, `odc.Start()` (does not need to return now), `ccdb.RunStart()` at `100`

### leave_CONFIGURED

- `leave_CONFIGURED` hooks are executed:
  - `kafka.PublishLeaveStateUpdate()` at `0`

### Transition START_ACTIVITY

- Fill Info previously retrieved by the BKP plugin is read from the variable stack and put into transition message to tasks
- Tasks are transitioned to `RUNNING`
- If everything succeeds up to this point, we report that the run has started

### enter_RUNNING

- `enter_RUNNING` hooks are executed
  - `o2-roc-ctp-emulator` for all ROC CTP emulator endpoints, `kafka.PublishEnterStateUpdate()` at `0`

### after_START_ACTIVITY

- `after_START_ACTIVITY` hooks with negative weights are executed
  - `trg.RunStart()` at `-10`
  - waiting until `odc.Start()` executed at `before_START_ACTIVITY+100` completes at `-10`
- `"run_start_completion_time_ms"` is set using current time. It is considered as the EOSOR timestamp.
- `after_START_ACTIVITY` hooks with positive weights (incl. 0) are executed:
  - `bookkeeping.UpdateRunStart()`, `bookkeeping.UpdateEnv()` at `+100`

## STOP_ACTIVITY (End Of Run)

This is the order of actions happening at a healthy end of run.

### before_STOP_ACTIVITY

- `before_STOP_ACTIVITY` hooks with negative weights are executed
  - `trg.RunStop()` at `-10`
- `"run_end_time_ms"` is set using the current time. It is considered as the EOR and SOEOR timestamps.
- `before_STOP_ACTIVITY` hooks with positive weights (incl. 0) are executed:
  - `odc.Stop()` (does not need to return now) at `0`

### leave_RUNNING

- `leave_RUNNING` hooks are executed
  - `kafka.PublishLeaveStateUpdate()` at `0`

### Transition STOP_ACTIVITY

- Tasks are transitioned to `CONFIGURED`
- If everything succeeds up to this point, we consider that the run has stopped

### enter_CONFIGURED

- `enter_CONFIGURED` hooks are executed
  - `kafka.PublishEnterStateUpdate()` at `0`

### after_STOP_ACTIVITY

- `after_STOP_ACTIVITY` hooks with negative weights are executed:
  - `trg.RunUnload()` at `-100`
  - `dcs.EndOfRun()` at `-50`
  - waiting until `odc.Stop()` executed at `before_STOP_ACTIVITY` completes at `-50`
- `"run_end_completion_time_ms"` is set using current time. It is considered as the EOEOR timestamp.
- `after_STOP_ACTIVITY` hooks with positive weights (incl. 0) are executed:
  - `ccdb.RunStop()` at `0`
  - `bookkeeping.UpdateRunStop()`, `bookkeeping.UpdateEnv()` at `+100`