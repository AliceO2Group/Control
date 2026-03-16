# ECS in Kubernetes

For now we can run OCC tasks in Kubernetes (namely `readout`, `stfsender`, `stfbuilder-senderoutput`) using
the task controller inside the `control-operator` folder and `kubectltask` created as a version of `controllabletask`
for the Mesos executor wrapping `kubectl` tool. However as the `kubectl` requires manifests we used `control-operator/ecs-manifests`
to store these test manifests. There are more test files apart from these manifests.

There are 3 subfolders inside `ecs-manifests`, namely `control-workflows`, `kubernetes-manifests` and `occ-configure-arguments`.

### `control-workflows`

In order to run given task from ECS you need to provide yaml template normally contained inside the `ControlWorkflows`
repository that is processed by ECS core and sent to Mesos framework that runs the given task on a given agent.
Inside the folder you can find files with suffixes `docker`, `kube`, `orig` appended to the name of the task they
are representing. These files are to be put into the `ControlWorkflows` so ECS can find those and run task
in proper way. Eg. if one is to run readout in Kubernetes copy (or symlink) `readout-kube.yaml` into the `ControlWorkflows`
directory under the name `readout.yaml` (same for the other tasks)

### `kubernetes-manifests`

Kubectltask requires Kubernetes manifests to pass to `kubectl`, these manifests are located in directory `kubernetes-manifests`.
There are two types of manifests `task.yaml` and `task-test.yaml`. The first one is the actual manifest to be used by
executors and kubectltask where environment variables substitution is used in a form of `${VAR}`. `task-test.yaml`
has the same form as templated manifest, but with actual test values to test container/binary inside the kubernetes.
Apply the manifest by invoking:

```bash
kubectl apply -f task-test.yaml
```

### `occ-configure-arguments`

yaml files containing data used by `peanut` to properly transition `readout`, `stfsender` and `stfbuilder-senderoutput`
to `CONFIGURED`. Use these files by either loading them into `peanut` with LoadConfiguration in TUI mode or
by using `--config` flag in CLI mode.
