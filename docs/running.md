# Running AliECS as a developer

**Looking to run your own software with AliECS? Please refer to [O²/FLP Suite deployment instructions](https://alice-flp-suite.docs.cern.ch/installation/)**

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
