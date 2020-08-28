# O² Control and Configuration

## Build
Starting from the `occ` directory.

```bash
$ mkdir build && cd build
$ cmake .. -DCMAKE_BUILD_TYPE=Debug -DFAIRMQPATH=<path to FairMQ prefix> -DFAIRLOGGERPATH=<path to FairLogger prefix>
$ make
```

It can also be built via aliBuild (package `Control-OCCPlugin`).

## Run example
From build dir:
```bash
$ occlib/examples/dummy-process/occexample-dummy-process
```
or
```bash
$ occlib/examples/dummy-process/occexample-dummy-process --control-port <some port>
```
or
```bash
$ OCC_CONTROL_PORT=<some port> occlib/examples/dummy-process/occexample-dummy-process
```

The dummy process now waits for control commands.

## The OCC state machine

The figure below describes the OCC state machine, as implemented in the OCC library and exposed
by the OCC API for non-FairMQ devices (`controlmode.DIRECT`).

![The OCC state machine](OCCStateMachine-controlmode.DIRECT.png)

Note: a PAUSED state with events PAUSE/RESUME is foreseen but not used yet.

## Single process control with `peanut`

See [`peanut` Overview](peanut/README.md).

## OCC API debugging with `grpcc`

We can send gRPC-based OCC commands manually with an interactive gRPC client
like [`grpcc`](https://github.com/njpatel/grpcc):
```bash
$ sudo yum install http-parser nodejs npm
$ npm install -g grpcc
```

In a new terminal, we go to the `occ` directory (not the `build` dir) and connect via gRPC:
```bash
$ grpcc -i --proto protos/occ.proto --address 127.0.0.1:47100
```

If all went well, we get an interactive environment like so:
```

Connecting to occ_pb.Occ on 127.0.0.1:47100. Available globals:

  client - the client connection to Occ
    stateStream (StateStreamRequest, callback) returns StateStreamReply
    getState (GetStateRequest, callback) returns GetStateReply
    transition (TransitionRequest, callback) returns TransitionReply

  printReply - function to easily print a unary call reply (alias: pr)
  streamReply - function to easily print stream call replies (alias: sr)
  createMetadata - convert JS objects into grpc metadata instances (alias: cm)
  printMetadata - function to easily print a unary call's metadata (alias: pm)

Occ@127.0.0.1:47100>
```

Let's try to send some commands. State changes will be reported in the standard output of the process.
```
Occ@127.0.0.1:47100> client.getState({}, pr)
{
  "state": "STANDBY"
}
Occ@127.0.0.1:47100> client.transition({srcState:"STANDBY", transitionEvent:"CONFIGURE", arguments:[]}, pr)
{
  "trigger": "EXECUTOR",
  "state": "CONFIGURED",
  "transitionEvent": "CONFIGURE",
  "ok": true
}
Occ@127.0.0.1:47100> client.getState({}, pr)
{
  "state": "CONFIGURED"
}
Occ@127.0.0.1:47100> client.transition({srcState:"CONFIGURED", transitionEvent:"START", arguments:[]}, pr)
{
  "trigger": "EXECUTOR",
  "state": "RUNNING",
  "transitionEvent": "START",
  "ok": true
}
Occ@127.0.0.1:47100> client.transition({srcState:"RUNNING", transitionEvent:"STOP", arguments:[]}, pr)
{
  "trigger": "EXECUTOR",
  "state": "CONFIGURED",
  "transitionEvent": "STOP",
  "ok": true
}
Occ@127.0.0.1:47100> client.transition({srcState:"CONFIGURED", transitionEvent:"EXIT", arguments:[]}, pr)
{
  "trigger": "EXECUTOR",
  "state": "DONE",
  "transitionEvent": "EXIT",
  "ok": true
}
# no further commands possible, EXIT stops the process
```

## Developer reference
1. Build & install the OCC library either manually or via aliBuild (`Control-OCCPlugin`);
2. check out [the dummy process example](occlib/examples/dummy-process) and [its entry point](occlib/examples/dummy-process/main.cxx) and to see how to instantiate OCC;
3. implement interface at [`occlib/RuntimeControlledObject.h`](occlib/RuntimeControlledObject.h),
4. link your non-FairMQ O² process against the target `AliceO2::Occ` as described in [the dummy process README](occlib/examples/dummy-process/README.md#standalone-build).
