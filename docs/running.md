# Running AliECS as a developer


> **WARNING**: The running instructions described in this page are **for development purposes only**. Users interested in deploying, running and controlling O²/FLP software or their own software with AliECS should refer to the [O²/FLP Suite instructions](https://alice-flp-suite.docs.cern.ch/installation/) instead.


## Running the AliECS core

This part assumes you have already set up the Go environment, fetched the sources and built all AliECS Go components.

The recommended way to set up a Mesos cluster is by performing a complete deployment of the O²/FLP Suite with `o2-flp-setup`. The AliECS core on the head node should be stopped (`systemctl stop o2-aliecs-core`) and your own AliECS core should be made to point to the head node.

The following example flags assume a remote head node `centosvmtest`, the use of the default `settings.yaml` file, very verbose output, verbose workflow dumps on every workflow deployment, and the executor having been copied (`scp`) to `/opt/o2control-executor` on all controlled nodes:

```bash
--coreConfigurationUri
"file://$HOME/workspace/Control/hacking/settings.yaml"
--globalConfigurationUri
"consul://centosvmtest:8500"
--mesosUrl
http://centosvmtest:5050/api/v1/scheduler
--verbose
--veryVerbose
--executor
/opt/o2control-executor
--dumpWorkflows
```

See [Using `coconut`](./coconut/README.md) for instructions on the O² Control core command line interface.

# Running AliECS in production

The AliECS core runs as a systemd service in the O²/FLP cluster at Point 2.

## Health checks

There is a checker script that polls AliECS for its status (`coconut env list`).

1) The checker script (checkAliECScore available in GL) now makes 3 attempts with 10 seconds timeout.
2) All failed attempts are recorded in the aliecs local file /tmp/checkAliECScore.out
3) The ILG message is issued at the third consecutive failure.
