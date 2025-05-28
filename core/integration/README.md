# Integration plugins

The integration plugins allow AliECS to communicate with other ALICE services.
A plugin can register a set of callback which can be invoked upon defined environment events (state transitions).

## Plugin system overview

All plugins should implement the [`Plugin`](https://github.com/AliceO2Group/Control/blob/master/core/integration/plugin.go) interface.
See the existing plugins for examples.

In order to have the plugin loaded by the AliECS, one has to:

- add `RegisterPlugin` to the `init()` function in [AliECS core main source](https://github.com/AliceO2Group/Control/blob/master/cmd/o2-aliecs-core/main.go)
- add plugin name in the `integrationPlugins` list and set the endpoint in the AliECS configuration file (typically at `/o2/components/aliecs/ANY/any/settings` in the configuration store)

# Integrated service operations

In this chapter we list and describe the integrated service plugins.

## Bookkeeping

The legacy Bookkeeping plugin sends updates to Bookkeeping about the state of data taking runs.
As of May 2025, Bookkeeping has transitioned into consuming input from the Kafka event service and the only call in use is "FillInfo", which allows ECS to retrieve LHC fill information.

## CCDB

CCDB plugin calls PDP-provided executable which creates a General Run Parameters (GRP) object at each run start and stop.

## DCS

DCS plugin communicates with the ALICE Detector Control System (DCS).
In particular, it communicates with a gRPC endpoint exposed by `ecs2dcsgateway` (developed by the DCS team), which then passes the requests to the DCS.

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
operation will be performed for the PFR-compatible detectors.

Despite some detectors not having performed PFR, the environment
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

`dcs.PrepareForRun` call fails if no detectors are PFR-compatible
or PFR fails for all those which were PFR-compatible,
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

### ECS2DCS2ECS mock server

ECS2DCS2ECS is a simple server that mimics the DCS service and is developed by the DCS team.
In particular, it sends heartbeats, reports detectors as PFR or SOR available and replies with requested state sequences.

The state sequences should be added as extra arguments to the DCS calls with key `"STATE_SEQUENCE"`, while the value is a comma-separated list of `MILLIS_DELAY:DETECTOR_STATE` values.

From the AliECS user point of view, state sequences can be defined as the following user-vars (Advanced Configuration panel in COG), which are then passed to the corresponding PFR/SOR/EOR calls for all detectors:

- `dcs_pfr_state_sequence`
- `dcs_sor_state_sequence`
- `dcs_eor_state_sequence`

#### Testing tips

Typical valid sequences are:

```
{
  "dcs_pfr_state_sequence": "1000:RUN_OK",
  "dcs_sor_state_sequence": "1000:SOR_PROGRESSING,3000:RUN_OK",
  "dcs_eor_state_sequence": "1000:EOR_PROGRESSING,3000:RUN_OK"
}
```

`PFR_AVAILABLE` and `SOR_AVAILABLE` are typically send by DCS via the heartbeat stream, so requesting them as an answer to an operation request does not reflect the behaviour of the production system.

To trigger a `TIMEOUT` event, one should not request a `TIMEOUT` state in the sequence, but rather put a too long delay compared to the provided gRPC timeout.
Both in the production and the mock setup, `TIMEOUT` events are generated by `ecs2dcsgateway` when a detector does not respond to a request within the specified timeout.

## DD Scheduler

DD scheduler plugin informs the Data Distribution software about the pool of FLPs taking part in data taking.

## Kafka (legacy)

See [Legacy events: Kafka plugin](/docs/kafka.md#legacy-events-kafka-plugin)

## ODC

ODC plugin communicates with the [Online Device Control (ODC)](https://github.com/FairRootGroup/ODC) instance of the ALICE experiment, which controls the event processing farm used in data taking and offline processing.

## Test plugin

Test plugin serves as an example of a plugin and is used for testing the plugin system.

## Trigger

Trigger plugin communicates with the ALICE trigger system.
