# Building O² Control

## Overview

O² Control consists of four main components:
* core - Go, runs locally for testing purposes and connects to a Mesos master
* executor - Go, runs throughout the cluster
* coconut - Go, runs locally or wherever you like and connects to core
* process control plugin + library - C++, runs throughout the cluster linked by user processes

## Go environment

```bash
$ sudo yum update
$ sudo yum install golang # or alisw-golang+1.11.1-1 and load the module
```

We also need to get the `protoc` binary for Protobuf 3.5 or later. 
It is available as an aliDist recipe, or as `alisw-protobuf+v3.5.2-4` (or newer) in the alisw YUM repository. 
Pick your favorite way to install it, and make sure to load the Protobuf module before continuing.

Installing the `protobuf3-compiler` package from regular CentOS repositories and then creating a `protoc` 
symlink to the `protoc3` binary might also work, but is unsupported.

Setting `GOPATH` and `PATH`:
```bash
$ export GOPATH=$HOME/go # or some other path for local Go binaries, packages and sources
$ export PATH=$GOPATH/bin:$PATH
```

Check if all is well:
```bash
$ go version
$ go env
```

## Clone and build the Go bits

Fetching the sources. You'll get a "no buildable Go source files" error and that's ok,
it happens because O² Control has its own Makefile instead of using plain `go build`.
```bash
$ go get -d github.com/AliceO2Group/Control
```

Running make. This will take a while as all dependencies are gathered, built and installed.
```bash
$ cd go/src/github.com/AliceO2Group/Control
$ make all
```

You should find `o2control-core`, `o2control-executor` and `coconut` in `bin`.

If you wish to also build the process control library and/or plugin, see [the OCC readme](../occ/README.md).

Otherwise, continue with [setting up the development cluster](DCOS.md) which includes instructions on
installing these components from RPM.
