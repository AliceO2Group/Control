# ECS with Kubernetes

> ⚠️ **Warning**
> All Kubernetes work done is in a stage of prototype.

## Kubernetes Cluster

While prototyping we used many Kubernetes clusters, namely [`kind`](https://kind.sigs.k8s.io/), [`minikube`](https://minikube.sigs.k8s.io/docs/) and [`k3s`](https://k3s.io/)
in both local and remote cluster deployment. We used Openstack for remote deployment.
Follow the guides at the individual distributions in order to create the desired cluster setup.
k3s is recommended to run this prototype, as it is lightweight
and easily installed distribution which is also [`CNCF`](https://www.cncf.io/training/certification/) certified.

All settings of `k3s` were used as default except one: locked-in-memory size. Use `ulimit -l` to learn
what is the limit for the current user and `LimitMEMLOCK` inside the k3s systemd service config
to set it for correct value. Right now the `flp` user has unlimited size (`LimitMEMLOCK=infinity`).
This config is necessary because even if you are running Pods with the privileged security context
under user flp, Kubernetes still sets limits according to its internal settings and doesn't
respect linux settings.

Another setup we expect at this moment to be present at the target nodes
is ability to run Pods with privileged permissions and also under user `flp`.
This means that the machine has to have `flp` user setup the same way as
if you would do the installation with [`o2-flp-setup`](https://alice-flp.docs.cern.ch/Operations/Experts/system-configuration/utils/o2-flp-setup/).

## Task Controller

Following text assumes that there is a Task Controller from `control-operator` running
at your K8s cluster and Task CRD installed at your cluster.
You can find the details about the usage in the [documentation](/control-operator/README.md).

## Running tasks (`KubectlTask`)

ECS is setup to run tasks through Mesos on all required hosts baremetal with active
task management (see [`ControllableTask`](/executor/executable/controllabletask.go))
and OCC gRPC communication. When running docker task through ECS we could easily
wrap command to be run into the docker container with proper settings
([see](/docs/running_docker.md)). This is however not possible for Kubernetes
workloads as the Pods are "hidden" inside the cluster. So we plan
to deploy our own Task Controller which will connect to and guide
OCC state machine of required tasks. Thus we need to create custom
POC way to communicate with Kubernetes cluster from Mesos executor.

The reason why we don't call Kubernetes cluster directly from ECS core
is that ECS does a lot of heavy lifting while deploying workloads,
monitoring workloads and by generating a lot of configuration which
is not trivial to replicate manually. However, if we create some class
that would be able to deploy one task into the Kubernetes and monitor its
state we could replicate `ControllableTask` workflow and leave ECS
mostly intact for now, save a lot of work and focus on prototyping
Kubernetes operator pattern.

Thus [`KubectlTask`](/executor/executable/kubectltask.go) was created. This class
is written as a wrapper around `kubectl` utility to manage Kubernetes cluster.
It is based on following `kubectl` commands:

* `apply` => `kubectl apply -f manifest.yaml` - deploys resource described inside given manifest
* `delete` => `kubectl delete -f manifest.yaml` - deletes resource from cluster
* `patch` => `kubectl patch -f exampletask.yaml --type='json' -p='[{"op": "replace", "path": "/spec/state", "value": "running"}]` - changes the state of resource inside cluster
* `get` => `kubectl get -f manifest.yaml -o jsonpath='{.spec.state}'` - queries exact field of resource (`state` in the example) inside cluster.

These four commands allow us to deploy and monitor status of the deployed
resource without necessity to interact with it directly. However `KubectlTask`
expects that resource is the CRD [Task](/control-operator/api/v1alpha1/task_types.go).

In order to activate `KubectlTask` you need to change yaml template
inside the `ControlWorkflows` directory. Namely:

* add path to the kubectl manifest as the first argument in `.command.arguments` field
* change `.control.mode` to either `kubectl_direct` or `kubectl_fairmq`
You can find working template inside `control-operator/ecs-manifests/control-workflows/*-kube.yaml`

Working kubectl manifests are to be found in `control-operator/ecs-manifests/kubernetes-manifests`.
You can see `*test.yaml` for concrete deployable manifests by `kubectl apply`, the rest
are the templates with variables to be filled in in a `${var}` format. `KubectlTask`
fills these variables from env vars.
