# operator

Folder with operators regarding Task and Environment deployment.

## Description

In order to deploy Task and Environment workflows to the k8s cluster you need controllers and operators
controlling custom CRDs defining ALICE custom workload. This Folder defines and implements all moving parts together with Makefile
to build, deploy, install CRDs and operators.

## Architecture

The operator is split into two separate binaries with different deployment strategies:

**task-manager** runs as a DaemonSet — one pod per node. Each pod is responsible only for `Task` resources assigned to its node (matched via `spec.nodeName`). This is necessary because the task-manager communicates with OCC gRPC processes running locally on the same node via `hostNetwork`.

**environment-manager** runs as a Deployment with a single replica per cluster. It is responsible for `Environment` resources which are cluster-scoped and not tied to a specific node.

## Getting Started

You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster. Author had the most success with K3s [see](/docs/kubernetes_ecs.md).
**Note:** Your controller will automatically use the current context in your kubeconfig (usually ~/.kube/config) file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster

Following commands show basic use of Makefile. However this isn't exhaustive list.

1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

1. Build and push your images. Default image tags are defined in the Makefile via `TASK_IMG` and `ENVIRONMENT_IMG`. Override them only if you want to use a different registry or tag:

```sh
make docker-build docker-push TASK_IMG=<some-registry>/task-manager:tag ENVIRONMENT_IMG=<some-registry>/environment-manager:tag
```

1. Deploy the controllers to the cluster. Uses the same `TASK_IMG` and `ENVIRONMENT_IMG` defaults, override them if needed:

```sh
make deploy TASK_IMG=<some-registry>/task-manager:tag ENVIRONMENT_IMG=<some-registry>/environment-manager:tag
```

### Uninstall CRDs

To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller

UnDeploy the controller from the cluster:

```sh
make undeploy
```

## Contributing

// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works

This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out

1. Install the CRDs into the cluster:

```sh
make install
```

1. Run a controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run-environment
```

The task-manager requires a `NODE_NAME` environment variable to know which node it is responsible for. In-cluster this is injected automatically via the downward API. When running locally you must set it manually:

```sh
NODE_NAME=<your-node-name> make run-task
```

**NOTE:** You can also install CRDs and run in one step: `make install run-environment` or `make install run-task`

### Modifying the API definitions

If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
