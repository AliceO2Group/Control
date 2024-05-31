# Environment operation order

This chapter attempts to document the order of important operations done during environment transitions.
Since AliECS is an evolving system, the information presented here might be out-of-date, thus please refer to event handling in [core/environment/environment.go](https://github.com/AliceO2Group/Control/blob/master/core/environment/environment.go) and plugin calls in [ControlWorkflows/workflows/readout-dataflow.yaml](https://github.com/AliceO2Group/ControlWorkflows/blob/master/workflows/readout-dataflow.yaml) for the ultimate source of truth.
Also, please report to the ECS developers any inaccuracies.

[State Machine Callbacks](configuration.md#State-machine-callbacks) documents the order of callbacks that can be associated with state machine transitions.

## START_ACTIVITY (Start Of Run)

This is the order of actions happening at a healthy start of run.

### before_START_ACTIVITY

- `"run_number"` is set.
- `"run_start_time_ms"` is set using the current time. It is considered as the SOR and SOSOR timestamps.
- `before_START_ACTIVITY` hooks are executed:
  - `trg.PrepareForRun()` at `-200`
  - `trg.RunLoad()`, `bookkeeping.StartOfRun()` at `-100`
  - `bookkeeping.RetrieveFillInfo()` at `-99`
  - `kafka.PublishStartActivityUpdate()` at `-50`
  - `dcs.StartOfRun()`, `odc.Start()` (does not need to return now), `ccdb.RunStart()` at `0`

### leave_CONFIGURED

- `leave_CONFIGURED` hooks are executed
  - `kafka.PublishLeaveStateUpdate()` at `0`

### Transition START_ACTIVITY

- Fill Info previously retrieved by the BKP plugin is read from the variable stack and put into transition message to tasks
- Tasks are transitioned to `RUNNING`
- If everything succeeds up to this point, we report that the run has started

### enter_RUNNING

- `enter_RUNNING` hooks are executed
  - `o2-roc-ctp-emulator` for all ROC CTP emulator endpoints, `kafka.PublishEnterStateUpdate()` at `0`

### after_START_ACTIVITY
- `"run_start_completion_time_ms"` is set using current time. It is considered as the EOSOR timestamp.
- `after_START_ACTIVITY` hooks are executed:
  - `trg.RunStart()` at `0`
  - waiting until `odc.Start()` executed at `before_START_ACTIVITY` completes at `0`
  - `bookkeeping.UpdateRunStart()`, `bookkeeping.UpdateEnv()` at `+100`

## STOP_ACTIVITY (End Of Run)

This is the order of actions happening at a healthy end of run.

### before_STOP_ACTIVITY

- `"run_end_time_ms"` is set using the current time. It is considered as the EOR and SOEOR timestamps.
- `before_STOP_ACTIVITY` hooks are executed:
  - `trg.RunStop()`, `odc.Stop()` (does not need to return now) at `0`

### leave_RUNNING

- `leave_RUNNING` hooks are executed
  - `kafka.PublishLeaveStateUpdate()` at `0`

### Transition STOP_ACTIVITY

- Tasks are transitioned to `STOP`
- If everything succeeds up to this point, we consider that the run has stopped

### enter_CONFIGURED

- `enter_CONFIGURED` hooks are executed
  - `kafka.PublishEnterStateUpdate()` at `0`

### after_STOP_ACTIVITY
- `"run_end_completion_time_ms"` is set using current time. It is considered as the EOEOR timestamp.
- `after_STOP_ACTIVITY` hooks are executed:
  - `trg.RunUnload()` at `-100`
  - `ccdb.RunStop()`, `dcs.EndOfRun()` at `0`
  - waiting until `odc.Stop()()` executed at `before_STOP_ACTIVITY` completes at `0`
  - `bookkeeping.UpdateRunStop()`, `bookkeeping.UpdateEnv()` at `+100`