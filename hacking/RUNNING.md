# Running the O² control system

:warning: :construction: **this section is out of date**

**User? See [Single node O²/FLP software deployment instructions](https://gitlab.cern.ch/AliceO2Group/system-configuration/blob/master/ansible/docs/O2_INSTALL_FLP_STANDALONE.md)**

**Developer? See [the run script](run.sh) instead**

This part assumes you have already set up the Go environment, fetched the sources and built
`o2control-core` and `o2control-executor` in `bin`, deployed the DC/OS Vagrant development
environment and set up the software you wish to run on this cluster (for example with `fpctl`).

## grpcc

In order to talk to `o2control-core` we can use `coconut`, or we can make calls directly with a gRPC client,
such as [`grpcc`](https://github.com/njpatel/grpcc).

Assuming you have installed Node.js and `npm` (on CC7 `$ sudo yum install http-parser nodejs npm`), the
installation with `npm` is straightforward.
```bash
$ sudo npm install -g grpcc
```

## Putting it all together

Assuming the DC/OS Vagrant environment is up, a Mesos master will be running at `m1.dcos` with:
* DC/OS interface at [`http://m1.dcos/`](http://m1.dcos/),
* Mesos interface at [`http://m1.dcos/mesos/`](http://m1.dcos/mesos/),
* Marathon interface at [`http://m1.dcos/marathon/`](http://m1.dcos/marathon/).

The `hacking` directory contains some wrapper scripts that rely on a Mesos master at `m1.dcos` and
make running `o2control-core` easy.

It also contains a dummy configuration file (`config.yaml`) which simulates what should normally be
a Consul instance.

Run `o2control-core`:
```bash
$ hacking/run.sh
```
or:
```bash
$ bin/o2control-core -mesos.url http://m1.dcos:5050/api/v1/scheduler -executor.binary </in-cluster/path/to/o2control-executor> -verbose -config "file://hacking/example-config.yaml"
```

Use `grpcc` to talk to it:
```bash
$ hacking/grpcc.sh
```
or:
```bash
$ grpcc -i --proto core/protos/o2control.proto --address 127.0.0.1:47102
```

See [Using `coconut`](../coconut/README.md) for instructions on the O² Control core command line interface.
