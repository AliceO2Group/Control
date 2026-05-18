# K8s-native task implementation notes

Goal: deploy and control tasks directly via Kubernetes (no Mesos, no KubectlTask wrapper).
ECS core talks to K8s API directly, operator handles OCC gRPC transitions.

---

## Key files

| File | Role |
|------|------|
| `common/controlmode/controlmode.go` | add new `KUBERNETES` control mode |
| `core/task/task.go` | `Task` struct, `IsLocked()`, `IsClaimable()` |
| `core/task/manager.go` | `acquireTasks`, `configureTasks`, `transitionTasks` |
| `core/task/taskclass/class.go` | `Class.Control.Mode` parsed from YAML |
| `control-operator/pkg/client/client.go` | K8s client (already done in OCTRL-1090) |
| `control-operator/api/v1alpha1/task_types.go` | Task CRD spec |

---

## Step 1 — new control mode

In `common/controlmode/controlmode.go`, add:

```go
KUBERNETES_DIRECT
KUBERNETES_FAIRMQ
```

Handle in `String()` and `UnmarshalText()`.

Task YAML would then use:
```yaml
control:
  mode: kubernetes_direct
```

---

## Step 2 — Task struct identity fields

`IsLocked()` currently checks five Mesos-specific fields:
```go
len(hostname) > 0 && len(agentId) > 0 && len(offerId) > 0 &&
len(taskId) > 0 && len(executorId) > 0 && parent != nil
```

For K8s tasks `agentId`, `offerId`, `executorId` are meaningless.

Options:
- Add a `deploymentBackend` field + K8s-specific fields (`k8sName`, `k8sNamespace`, `k8sNode`)
- Update `IsLocked()` to branch on backend
- `taskId` can reuse the K8s CRD name, `hostname` can be the node name

Simplest approach: add optional fields, keep `taskId`/`hostname`, set sentinel
values for `agentId`/`offerId`/`executorId` (e.g. `"kubernetes"`) so `IsLocked()`
still works without changes. Revisit if it gets messy.

---

## Step 3 — fork in `acquireTasks`

`acquireTasks` in `manager.go` is where deployment is triggered.
Split `taskDescriptors` by control mode **before** the Mesos offer loop:

```go
var mesosBound, k8sBound Descriptors
for _, desc := range tasksToRun {
    class, _ := m.classes.GetClass(desc.TaskClassName)
    if isKubernetesMode(class.Control.Mode) {
        k8sBound = append(k8sBound, desc)
    } else {
        mesosBound = append(mesosBound, desc)
    }
}
```

Then:
- `mesosBound` → existing Mesos offer loop (unchanged)
- `k8sBound` → new `deployKubernetesTasks(ctx, envId, k8sBound)`

Both produce a `DeploymentMap` (`*Task → *Descriptor`).
Merge them before the shared tail that wires parents (line ~656 in manager.go):

```go
for taskPtr, descriptor := range allDeployed {
    taskPtr.SetParent(descriptor.TaskRole)
    taskPtr.GetParent().SetTask(taskPtr)
}
```

### Task reuse check
The existing reuse loop (line ~404) checks `m.AgentCache` which is Mesos-only.
K8s tasks can be reused too (same `IsClaimable()` logic), but constraint checking
must skip AgentCache. Either skip reuse for K8s initially, or implement a
separate K8s node attribute cache later.

---

## Step 4 — `deployKubernetesTasks`

New function in `manager.go`:

```go
func (m *Manager) deployKubernetesTasks(ctx context.Context, envId uid.ID, descriptors Descriptors) (DeploymentMap, error)
```

For each descriptor:
1. Build `v1alpha1.Task` spec from the descriptor (class name, args, image, etc.)
2. `m.k8sClient.CreateTask(ctx, &k8sTask)`
3. Watch via `m.k8sClient.WatchTasks(ctx)` until `status.state == STANDBY`
   (operator sets this once the pod is up and OCC is responsive)
4. Call `newTaskForKubernetes(descriptor, crdName, nodeName)` to get `*Task`
5. Add to `DeploymentMap`, add to roster

`newTaskForKubernetes` mirrors `newTaskForMesosOffer` but fills in K8s identity:
```go
t = &Task{
    name:       fmt.Sprintf("%s#%s", descriptor.TaskClassName, newId),
    parent:     descriptor.TaskRole,
    className:  descriptor.TaskClassName,
    hostname:   nodeName,          // from pod spec once scheduled
    taskId:     crdName,           // K8s CRD name
    agentId:    "kubernetes",      // sentinel
    offerId:    "kubernetes",      // sentinel
    executorId: "kubernetes",      // sentinel
    state:      sm.STANDBY,
    status:     ACTIVE,
    ...
}
```

Wire the K8s client into `Manager` at startup (alongside `schedulerState`).

---

## Step 5 — OCC transitions (`configureTasks` / `transitionTasks`)

Currently both functions do:
```
GetMesosCommandTargets → CommandQueue → Servent → sendCommand → Mesos MESSAGE → executor → OCC gRPC
```

For K8s tasks, the operator already handles OCC when you patch the CRD.
Branch on `task.GetControlMode()`:

```go
var mesosTasks, k8sTasks Tasks
for _, t := range tasks {
    if isKubernetesMode(t.GetControlMode()) {
        k8sTasks = append(k8sTasks, t)
    } else {
        mesosTasks = append(mesosTasks, t)
    }
}
// existing path for mesosTasks
// new path for k8sTasks: patch spec.state + spec.arguments, then watch status.state
```

K8s transition:
1. `client.GetTask(ctx, task.taskId)` → get current CRD
2. Patch `spec.state` = target state, `spec.arguments` = args
3. `client.UpdateTask(ctx, ...)`
4. Poll/watch `status.state` until it matches target (or timeout)

---

## Step 6 — status updates

Currently Mesos pushes `TaskStatus` events → `updateTaskStatus`.

Run a background goroutine (started in `Manager.Start`) that watches K8s Task CRDs:

```go
watcher, _ := m.k8sClient.WatchTasks(ctx)
for event := range watcher.ResultChan() {
    k8sTask := event.Object.(*v1alpha1.Task)
    m.updateTaskState(k8sTask.Name, k8sTask.Status.State)
}
```

---

## Step 7 — kill

`killTask` in manager.go currently calls `schedulerState.killTask` (Mesos KILL).

Branch on control mode → `m.k8sClient.DeleteTask(ctx, task.taskId)`.

---

## What does NOT change

- `taskRole` in workflow — agnostic to deployment backend
- Role tree, environment state machine, variable stacks
- `Descriptor` / `AcquireRoles` call path
- `IsClaimable()` — same semantics, works for both backends
- gRPC server, environment lifecycle

---

## Open questions

- Where to put `k8sClient` on `Manager` — field alongside `schedulerState`?
- Namespace: from config/viper or from task class YAML?
- Watch reconnect logic if the watcher drops
- K8s node constraint matching (replacement for `AgentCache`)
- Whether to support task reuse for K8s tasks initially
