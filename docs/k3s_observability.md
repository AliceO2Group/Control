# K3s Observability

> ⚠️ **Warning**
> This observability setup is a prototype configured for a specific OpenSearch instance. Adjust `opensearch-config.yml` before deploying to a different environment.

All manifests and server-side configuration live in `control-operator/k3s-observability/`:

```
k3s-observability/
├── manifests/          # applied via kubectl
│   ├── opensearch-config.yml
│   ├── fluent-bit-events.yml
│   ├── fluent-bit-logs.yml
│   └── fluent-bit-audit.yml
└── other/              # deployed manually on the k3s server node
    └── audit-policy.yaml
```

## Overview

Three fluent-bit components run inside the k3s cluster and forward data to an external observability stack via the Fluent Forward protocol (port 24224 on `OPENSEARCH_HOST`):

| Manifest | Kind | What it collects |
|---|---|---|
| `fluent-bit-events.yml` | Deployment | Kubernetes `Event` objects — pod lifecycle, gRPC connections, controller-emitted events |
| `fluent-bit-logs.yml` | DaemonSet | Container stdout/stderr from all pods |
| `fluent-bit-audit.yml` | DaemonSet (control-plane only) | Kubernetes API audit log — full CRD specs on create/update/delete |

The external observability stack (Fluent Bit → OTel Collector → Data Prepper → OpenSearch) receives and processes the forwarded data.

`OPENSEARCH_HOST` and `OPENSEARCH_PORT` in `opensearch-config.yml` point at the observability Fluent Bit forward input, not at OpenSearch directly. All k3s Fluent Bit components read these via `envFrom`.

[Reloader](https://github.com/stakater/Reloader) can be used to automatically restart any pods whenever their ConfigMap changes including the fluent-bit ones. However it is not required for fluent-bit deployment. Each Deployment/DaemonSet has the annotation `reloader.stakater.com/auto: "true"` on the pod template.

## Deployment

### First-time setup

**1. Configure OpenSearch endpoint**

Edit `manifests/opensearch-config.yml` with the correct host and port, then apply all manifests:

```bash
kubectl apply -f control-operator/k3s-observability/manifests/
```

**2. Set up audit logging on the k3s server node**

Copy the audit policy to the server:
```bash
scp control-operator/k3s-observability/other/audit-policy.yaml <server>:/etc/rancher/k3s/audit-policy.yaml
```

Create `/etc/rancher/k3s/config.yaml` on the server (create it if it doesn't exist):
```yaml
kube-apiserver-arg:
  - "audit-log-path=/var/log/k3s-audit.log"
  - "audit-policy-file=/etc/rancher/k3s/audit-policy.yaml"
  - "audit-log-maxage=7"
  - "audit-log-maxbackup=3"
  - "audit-log-maxsize=100"
```

Restart k3s. If leftover containerd-shim processes block the restart:
```bash
/usr/local/bin/k3s-killall.sh && systemctl start k3s
```

**(OPTIONAL) 3. Install Reloader**
```bash
kubectl apply -f https://raw.githubusercontent.com/stakater/Reloader/master/deployments/kubernetes/reloader.yaml
```

### Updating config

After any change to the manifests:
```bash
kubectl apply -f control-operator/k3s-observability/manifests/
```

Reloader will automatically restart affected pods when their ConfigMap changes.

## What is recorded and where

### Kubernetes Events (`fluent-bit-events`)

Watches the Kubernetes `Event` API directly. Captures events emitted by kubelet and the ALIECS controllers:

- Pod lifecycle: `Created`, `Started`, `Killing` (explicit kill), `BackOff` (crash loop)
- Task controller: pod IP assignment, gRPC connection established, pod failure detected
- Notable gap: containers that exit on their own do not generate a kubelet `Killing` event — their exit is only visible in pod status. The task controller emits a `PodFailed` event to fill this gap.

Query in OpenSearch: `WHERE attributes.kind = 'Event'`

### Container logs (`fluent-bit-logs`)

Tails `/var/log/containers/*.log` on every node. Captures stdout/stderr from all containers including the task and environment managers.

The ALIECS controllers are configured with `--zap-encoder=json` so their log lines are pure JSON. The fluent-bit `merge_log: on` option parses these automatically, lifting structured fields as queryable attributes. The OTel Collector further normalises controller logs — including mapping the Go `level` field (`debug`/`info`/`warn`/`error`) to OTLP `severity_text` and `severity_number` so that log level filtering works correctly in OpenSearch Dashboards:

### Audit log (`fluent-bit-audit`)

Tails `/var/log/k3s-audit.log` on the control-plane node. Records every API server interaction matching the audit policy.

**What is captured:**

| Resource | Level | Verbs |
|---|---|---|
| ALIECS CRDs (Task, Environment, TaskTemplate) | `RequestResponse` (full spec) | create, update, patch, delete |
| Pods | `Metadata` (no body) | create, delete |

`RequestResponse` means the full request and response body is logged — i.e. the complete spec of every Task and Environment CRD at the time it was created or modified. This gives a persistent record of what was deployed even after the CRD is deleted.

`managedFields` is stripped at source via `omitManagedFields: true` in the audit policy. This field uses `.` as a JSON key (Kubernetes FieldsV1 format), which OpenSearch rejects. Removing it at the kube-apiserver level is cleaner than filtering it in the pipeline.

Pod deletion (which sets the pod to Terminating) is captured at `Metadata` level via `verb: delete`.

What is **not** captured: pod status transitions (Running → Terminating → Succeeded/Failed) — these are `patch` operations on the Pod object and are excluded to avoid noise.

## Audit policy

The audit policy at `other/audit-policy.yaml` is a server-side file read by the kube-apiserver at startup — it is **not** a Kubernetes resource and cannot be applied with `kubectl`. Any change to it requires copying the file to the server and restarting k3s.

Noise excluded by policy: lease updates, node heartbeats, health/metrics endpoints. `managedFields` is excluded from all captured events via `omitManagedFields: true` on the ALIECS CRD rule.
