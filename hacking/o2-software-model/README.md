# O2 software toy model

This directory contains a toy model which pretends to be O2 software in terms of how it uses memory and runs on a K8s cluster.
The manifest file declares an init container which creates a shared memory segment and two identical `memory-user` containers which write to anonymous and/or shared memory with specified rates.
One can set memory requests and limits to the containers and observe the behaviour.

The manifest also includes a `mem-guard` container, which monitors anonymous memory usage of `memory-user`s and kills them when they go above a set threshold.

See the manifest and environment variables inside for configuring memory consumption rates and thresholds.

See OCTRL-1079 for more context and info which was understood using this model.
## Running

1. Add `allocate.py` and `mem_guard.py` scripts to a configmap. This way we can reuse the official python image as-is.

```
kubectl create configmap allocate-script \
  --from-file=allocate.py=./allocate.py \
  --from-file=mem_guard.py=./mem_guard.py \
  -o yaml --dry-run=client | kubectl apply -f -
```

2. Apply the manifest

```
kubectl apply -f mem-allocate-test.yaml
```
