# Building AliECS

> **WARNING**: The building instructions described in this page are **for development purposes only**. Users interested in deploying, running and controlling O²/FLP software or their own software with AliECS should refer to the [O²/FLP Suite instructions](https://alice-flp.docs.cern.ch/Operations/Experts/system-configuration/utils/o2-flp-setup/) instead.


## Overview

AliECS consists of:

* core - Go, runs locally for testing purposes and connects to a Mesos master
* executor - Go, runs throughout the cluster
* coconut - Go, runs locally or wherever you like and connects to core
* process control plugin + library (OCC) - C++, runs throughout the cluster linked by user processes
* peanut - Go, local debugging tool

## Building with aliBuild

Assuming you already have an [aliBuild environment](https://alisw.github.io/alibuild/quick.html), building all AliECS components is as easy as:

```bash
$ aliBuild init --defaults o2-dataflow Control
$ aliBuild build --defaults o2-dataflow Control
```

For development purposes, due to the significant differences in building and deployment between the Go and C++ components, it is recommended to manually initialize two separate build instances instead:
```bash
# Includes OCC library, OCC plugin and OCC library example
$ aliBuild init --defaults o2-dataflow Control-OCCPlugin
$ aliBuild build --defaults o2-dataflow Control-OCCPlugin

# Includes core and executor in default build, can be used to build coconut and peanut locally
$ aliBuild init --defaults o2-dataflow Control-Core
$ aliBuild build --defaults o2-dataflow Control-Core
```

Once the `Control-Core` directory is initialized, the user can also follow the manual build instructions below to customize their build.


## Manual build

### Go environment

```bash
$ sudo yum update
$ sudo yum install golang # or latest alisw-golang and load the module
```

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

We also need to get the `protoc` binary for Protobuf 3.5 or later. 
It is available as an aliDist recipe, or as `alisw-protobuf+v3.12.3-1` (or newer) in the alisw YUM repository. 
Pick your favorite way to install it, and make sure to load the Protobuf module (with `alienv` or `aliswmod`) before continuing.

Installing the `protobuf3-compiler` package from regular CentOS repositories and then creating a `protoc` 
symlink to the `protoc3` binary might also work, but is unsupported.


### Clone and build (Go components only)

```bash
$ git clone git@github.com:AliceO2Group/Control.git
# or via HTTPS: $ git clone https://github.com/AliceO2Group/Control.git
$ cd Control
```

Normally in many Go projects you'd just do `go build`, but with AliECS this would give you a "no buildable Go source files" error. This happens because AliECS has its own Makefile instead of using plain `go build`, so we must run `make`.

Running `make` will take a while as all dependencies are gathered, built and installed.
```bash
$ make all
```

You should find several executables including `o2control-core`, `o2control-executor` and `coconut` in `bin`.

For subsequent builds (after the first one), plain `make` (instead of `make all`) is sufficient. See the [Makefile reference](makefile_reference.md) for more information.

If you wish to also build the process control library and/or plugin, see [the OCC readme](/occ/README.md).

This build of AliECS can be run locally and connected to an existing O²/FLP Suite cluster by passing a `--mesosUrl` parameter. If you do this, remember to `systemctl stop o2-aliecs-core` on the head node, in order to stop the core that came with the O²/FLP Suite and use your own.
