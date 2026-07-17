# ECS with Kubernetes

> ⚠️ **Warning**
> All Kubernetes work done is in a stage of prototype.

## Kubernetes Cluster

While prototyping we used many Kubernetes clusters, namely [`kind`](https://kind.sigs.k8s.io/), [`minikube`](https://minikube.sigs.k8s.io/docs/) and [`k3s`](https://k3s.io/)
in both local and remote cluster deployment. We used Openstack for remote deployment.
Follow the guides at the individual distributions in order to create the desired cluster setup.
k3s is recommended to run this prototype, as it is lightweight and easily installed distribution 
which is also [`CNCF`](https://www.cncf.io/training/certification/) certified. However kubernetes 
operator patterns used in this repo should be k8s distribution agnostic.

All settings of `k3s` were used as default except one: locked-in-memory size. Use `ulimit -l` to learn
what is the limit for the current user and `LimitMEMLOCK` inside the k3s systemd service config
to set it for correct value. Right now the `flp` user should have unlimited size (`LimitMEMLOCK=infinity`).
This config is necessary because even if you are running Pods with the privileged security context
under user flp, Kubernetes still sets limits according to its internal settings and doesn't
respect linux settings.

Another setup we expect at this moment to be present at the target nodes
is ability to run Pods with privileged permissions and also under user `flp`.
This means that the machine has to have `flp` user setup the same way as
if you would do the installation with [`o2-flp-setup`](https://alice-flp.docs.cern.ch/Operations/Experts/system-configuration/utils/o2-flp-setup/).

## Operator pattern

Any communication between kubernetes and OCC tasks and ECS and kubernetes is based on [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).
All operator pattern code implemented and expected to be used and deployed by ECS is located in the folder `control-operator`
with [README.md](/control-operator/README.md).

## Direct ECS <-> Kubernetes bridge

Apart from the old way of deploying tasks via Apache Mesos ECS now supports deployment of tasks 
directly to the Kubernetes as well and it is up to Admin/User to decide where the task should run. 
In order for Task to be deployed in Kubernetes cluster we need to have environment and task managers 
running inside the cluster as these are responsible for managing Custom Resource Definitions of Task and Environment.

The bridge works as follows:

1. **ECS core** (running outside the cluster) uses a typed Kubernetes client (`control-operator/pkg/client`) backed by `controller-runtime` to interact with the cluster.
2. When ECS deploys a task whose control mode is `kubernetes_direct` or `kubernetes_fairmq`, it creates a **Task CRD** and, if needed, an **Environment CRD** in the cluster instead of submitting a Mesos offer.
3. Inside the cluster, the **task-manager** and **environment-manager** operators (see `control-operator/cmd/`) watch these CRDs and reconcile the desired state — scheduling Pods, driving OCC gRPC state transitions, and writing status back into the CRD.
4. ECS watches the CRDs via the Kubernetes Watch API and reacts to status changes exactly as it would to Mesos task updates, keeping the rest of the ECS state machine intact.

This design lets ECS reuse its existing environment lifecycle, configuration generation, and monitoring logic while outsourcing Pod scheduling and OCC communication to the in-cluster operators. The relevant code can be found in manager.go and managerk8s.go. 
The manager.go implements the split between Mesos and K8s tasks while managerk8s.go implements actual creation, deployment, transitioning and updating ECS structure. 

To deploy a Kubernetes task you can in principle reuse existing control-workflow task YAML manifests by changing the control mode to `kubernetes_direct` or `kubernetes_fairmq`. But in practice there can be issues
with reusing existing manifests, for example bad quotation of arguments: Mesos runs tasks via shell, which interpreted and stripped quotation from commands. For this reason you can find ready-to-use manifests with all necessary changes
inside `control-operator/ecs-manifests/control-workflows/*kube-direct*`.

## Running tasks (`KubectlTask`)

This method is obsoleted by direct ECS <-> Kubernetes bridge. However it should still work
for debugging purposes. It is maintained on a best-effort basis only.

To use KubectlTasks you need to have Task controller running in k8s cluster. 
You can find the details about the usage in the [documentation](/control-operator/README.md).

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
