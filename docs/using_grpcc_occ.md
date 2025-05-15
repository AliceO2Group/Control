# OCC API debugging with grpcc

In order to talk to the AliECS core we can use `coconut`, or we can make calls directly with a gRPC client,
such as [`grpcc`](https://github.com/njpatel/grpcc).

Assuming you have installed Node.js and `npm` (on CC7 `$ sudo yum install http-parser nodejs npm`), the
installation with `npm` is straightforward.

```bash
$ sudo yum install http-parser nodejs npm
$ npm install -g grpcc # it can take a few minutes due to grpc build
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
