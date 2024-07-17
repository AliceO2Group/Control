# Environment operation order

This chapter attempts to document the order of important operations done during environment transitions.
Since AliECS is an evolving system, the information presented here might be out-of-date, thus please refer to event handling in [core/environment/environment.go](https://github.com/AliceO2Group/Control/blob/master/core/environment/environment.go) and plugin calls in [ControlWorkflows/workflows/readout-dataflow.yaml](https://github.com/AliceO2Group/ControlWorkflows/blob/master/workflows/readout-dataflow.yaml) for the ultimate source of truth.
Also, please report to the ECS developers any inaccuracies.

[State Machine Callbacks](configuration.md#State-machine-callbacks) documents the order of callbacks that can be associated with state machine transitions.

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
- `"run_end_time_ms"` is set using the current time. It is considered as the EOR and SOEOR timestamps.
- `before_STOP_ACTIVITY` hooks with positive weights (incl. 0) are executed:
  - `trg.RunStop()`, `odc.Stop()` (does not need to return now) at `0`

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

# Integrated service operations

## DCS

### DCS operations

The DCS integration plugin exposes to the workflow template (WFT) context the
following operations. Their associated transitions in this table refer
to the [readout-dataflow](https://github.com/AliceO2Group/ControlWorkflows/blob/master/workflows/readout-dataflow.yaml) workflow template.

| **DCS operation**     | **WFT call**        | **Call timing** | **Critical** | **Contingent on detector state** | 
|-----------------------|---------------------|---------------------------|--------------|----------------------------------|
| Prepare For Run (PFR) | `dcs.PrepareForRun` | during `CONFIGURE`        | `false`      | yes                              |
| Start Of Run (SOR)    | `dcs.StartOfRun`    | early in `START_ACTIVITY` | `true`       | yes                              |
| End Of Run (EOR)      | `dcs.EndOfRun`      | late in `STOP_ACTIVITY`   | `true`       | no                               |

The DCS integration plugin subscribes to the [DCS service](https://github.com/AliceO2Group/Control/blob/master/core/integration/dcs/protos/dcs.proto) and continually
receives information on operation-state compatibility for all
detectors.
When a given environment reaches a DCS call, the relevant DCS operation
will be called only if the DCS service reports that all detectors in that
environment are compatible with this operation, except EOR, which is
always called.

### DCS PrepareForRun behaviour

Unlike SOR and EOR, which are mandatory if `dcs_enabled` is set to `true`,
an impossibility to run PFR or a PFR failure will not prevent the
environment from transitioning forward.

#### DCS PFR incompatibility

When `dcs.PrepareForRun` is called, if at least one detector is in a
state that is incompatible with PFR as reported by the DCS service,
a grace period of 10 seconds is given for the detector(s) to become
compatible with PFR, with 1Hz polling frequency. As soon as all
detectors become compatible with PFR, the PFR operation is requested
to the DCS service.

If the grace period ends and at least one detector
included in the environment is still incompatible with PFR, the PFR
operation **will not run for any detector**.

However, the environment
can still transition forward towards the `RUNNING` state, and any DCS
activities that would have taken place in PFR will instead happen
during SOR. Only at that point, if at least one detector is not
compatible with SOR (or if it is but SOR fails), will the environment
declare a failure.

#### DCS PFR failure

When `dcs.PrepareForRun` is called, if all detectors are compatible
with PFR as reported by the DCS service (or become compatible during
the grace period), the PFR operation is immediately requested to the
DCS service.

If this operation fails for one or more detectors, the
`dcs.PrepareForRun` call as a whole is considered to have failed,
but since it is non-critical the environment may still reach the
`CONFIGURED` state and transition forward towards `RUNNING`.

As in the case of an impossibility to run PFR, any DCS activities that
would have taken place in PFR will instead be done during SOR.

### DCS StartOfRun behaviour

The SOR operation is mandatory if `dcs_enabled` is set to `true`
(AliECS GUI "DCS" switched on).

#### DCS SOR incompatibility

When `dcs.StartOfRun` is called, if at least one detector is in a
state that is incompatible with SOR as reported by the DCS service,
or if after a grace period of 10 seconds at least one detector is
still incompatible with SOR, the SOR operation **will not run for any
detector**.

The environment will then declare a **failure**, the
`START_ACTIVITY` transition will be blocked and the environment
will move to `ERROR`.

#### DCS SOR failure

When `dcs.StartOfRun` is called, if all detectors are compatible
with SOR as reported by the DCS service (or become compatible during
the grace period), the SOR operation is immediately requested to the
DCS service.

If this operation fails for one or more detectors, the
`dcs.StartOfRun` call as a whole is considered to have failed.

The environment will then declare a **failure**, the
`START_ACTIVITY` transition will be blocked and the environment
will move to `ERROR`

### DCS EndOfRun behaviour

The EOR operation is mandatory if `dcs_enabled` is set to `true`
(AliECS GUI "DCS" switched on). However, unlike with PFR and SOR, there
is **no check for compatibility** with the EOR operation. The EOR
request will always be sent to the DCS service during `STOP_ACTIVITY`.

#### DCS EOR failure

If this operation fails for one or more detectors, the
`dcs.EndOfRun` call as a whole is considered to have failed.

The environment will then declare a **failure**, the
`STOP_ACTIVITY` transition will be blocked and the environment
will move to `ERROR`.