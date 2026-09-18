This document describes k8s installation on our virtual nodes:
`alio2-bld4-mx-k8s[01..05].cern.ch`. Node `k8s01` was chosen as a control-plane and the rest as worker nodes. These VMs have RHEL 10 installed.

Author first created his own user so the commands used won't be in history of `root`
`alio2-bld4-mx-k8s01.cern.ch` was chosen as a control plane node and the rest as workers

## kubeadm/kubelet/kubectl installation

[kubeadm/kubelet/kubectl rhel guide](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/#installing-kubeadm-kubelet-and-kubectl)

- Disable SElinux

One important thing is that SE linux needs to be disabled until SElinux is better supported in kubelet
It is possible to install kubeadm/cluster with SE enabled, but for now I chose to follow guide to the t.

```
sudo setenforce 0
sudo sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config
```

- Verify all required kernel modules are loaded

RHEL 10 can have problems with having all required kernel modules, you can check:
```
lsmod | grep -E 'overlay|br_netfilter|nf_conntrack'
```
if there is module not loaded you can try to load it with 
```
sudo modprobe br_netfliter
```
if you get error that the module is missing:
```
sudo dnf install -y kernel-modules-extra
```
load all modules now that they are there:
```
sudo modprobe overlay
sudo modprobe br_netfilter
sudo modprobe nf_conntrack
lsmod | grep -E 'overlay|br_netfilter|nf_conntrack'
```
make the loading persistent:
```
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
nf_conntrack
EOF
```

- Add Kubernetes yum repo

```
cat <<EOF | sudo tee /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://pkgs.k8s.io/core:/stable:/v1.36/rpm/
enabled=1
gpgcheck=1
gpgkey=https://pkgs.k8s.io/core:/stable:/v1.36/rpm/repodata/repomd.xml.key
exclude=kubelet kubeadm kubectl cri-tools kubernetes-cni
EOF
```

- Install `kubeadm`, `kubectl` and `kubelet`

Our lab VMware setup uses `dnf4`:
```
sudo yum install -y kubelet kubeadm kubectl --disableexcludes=kubernetes
```

Enable kubelet service:
```
sudo systemctl enable kubelet.service
```

- Enable ipv4 forwarding for all nodes and contorl-plane machines
If 
```
sysctl net.ipv4.ip_forward
```
returns 0, this sysctl param needs to be added as it is required by setup, param persist across reboots:
```
cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.ipv4.ip_forward = 1
EOF
```

Apply sysctl params without reboot
```
sudo sysctl --system
```

- Container runtime

[`containerd`](https://github.com/containerd/containerd/blob/main/docs/getting-started.md) was chosen for it's popularity.
```
sudo dnf install -y containerd runc
```
enable the systemd service
```
systemctl daemon-reload
systemctl enable --now containerd
```
RHEL 10 is running cgroups v2 as a default, so we need containerd to use systemd as cgroups driver so add following to the
`/etc/containerd/config.toml` together with some other changes (like bin dir and other), contents of the config.toml are:
```
cat /etc/containerd/config.toml
version = 2

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".cni]
      bin_dir = "/opt/cni/bin"
      conf_dir = "/etc/cni/net.d"
    [plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.runc]
      [plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.runc.options]
        SystemdCgroup = true
  [plugins."io.containerd.internal.v1.opt"]
    path = "/var/lib/containerd/opt"
```


and restart the service
```
sudo systemctl restart containerd
sudo systemctl status containerd
```

# To init Worker node

You need to have Control-plane node ready to proceed!!

- Open following ports: [k8s](https://kubernetes.io/docs/reference/networking/ports-and-protocols/), [calico](https://docs.tigera.io/calico-enterprise/latest/operations/troubleshoot/troubleshooting)

```
sudo firewall-cmd --permanent --add-port=10250/tcp
sudo firewall-cmd --permanent --add-port=10256/tcp
sudo firewall-cmd --permanent --add-port=30000-32767/tcp
sudo firewall-cmd --permanent --add-port=179/tcp
sudo firewall-cmd --permanent --add-port=4789/udp
sudo firewall-cmd --permanent --add-masquerade
sudo firewall-cmd --permanent --zone=trusted --add-source=192.168.0.0/16
sudo firewall-cmd --permanent --zone=trusted --add-source=10.96.0.0/12
sudo firewall-cmd --reload
```

On control-plane node run
```
kubeadm token create --print-join-command
```
this command will give you the exact command you are supposed to run on on worker node and it whould look like
```
(only demonstrative example) kubeadm join ip.addr:port ...
```


# To init Control plane node
- Open following ports: [k8s](https://kubernetes.io/docs/reference/networking/ports-and-protocols/), [calico](https://docs.tigera.io/calico-enterprise/latest/operations/troubleshoot/troubleshooting)
```
sudo firewall-cmd --permanent --add-port=6443/tcp
sudo firewall-cmd --permanent --add-port=2379-2380/tcp
sudo firewall-cmd --permanent --add-port=10250/tcp
sudo firewall-cmd --permanent --add-port=10259/tcp
sudo firewall-cmd --permanent --add-port=10257/tcp
sudo firewall-cmd --permanent --add-port=5473/tcp
sudo firewall-cmd --permanent --add-port=179/tcp
sudo firewall-cmd --permanent --add-port=4789/udp
sudo firewall-cmd --permanent --add-masquerade
sudo firewall-cmd --permanent --zone=trusted --add-source=192.168.0.0/16
sudo firewall-cmd --permanent --zone=trusted --add-source=10.96.0.0/12
sudo firewall-cmd --reload
```

- run `kubeadm init`

pre run checks on control-plane node:
```
swapon --show          # should be empty
systemctl status containerd   # should be active/running
which runc              # should resolve
sysctl net.ipv4.ip_forward    # should be = 1

```

It is advised to use following config file to be used with kubeadm:
```
apiVersion: kubeadm.k8s.io/v1beta4
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: "10.163.42.208"
---
apiVersion: kubeadm.k8s.io/v1beta4
kind: ClusterConfiguration
networking:
  podSubnet: "192.168.0.0/16"
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
cgroupDriver: systemd
```

Explanation of the settings:
`localAPIEndpoint`: VMware network has two NICs, one connected to gpn and other for internal communication. this IP is the internal one, which is not a default so it is better to specifically list it 
`networking`: this defines the pod IP range, and it has to satisfy two things — (1) it must not overlap your existing networks (10.163.0.0/16 internal net or 128.141.175.0/24 gpn), and (2) it must exactly match the CIDR configured in Calico's own custom-resources.yaml (spec.calicoNetwork.ipPools[].cidr),
`cgroupDriver`: kubelet will use systemd as it needs to match with containerd and systemd is recommended as cgroup driver for cgroups v2

kubeadm invocation
```
sudo kubeadm init --config kubeadm-config.yaml
```

IMPORTANT!! read the output of kubeadm as there might be other requirements which are not fulfilled. 

At this moment we can start "using" the cluster:
```
To start using your cluster, you need to run the following as a regular user:

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config

Alternatively, if you are the root user, you can run:

  export KUBECONFIG=/etc/kubernetes/admin.conf
```

after copying cluster config you can check that control-plane is responding (eg.):
```
kubectl get nodes
NAME                          STATUS     ROLES           AGE   VERSION
alio2-bld4-mx-k8s01.cern.ch   NotReady   control-plane   17m   v1.36.3
```

The reason why cluster is responding NotReady is that we have no network plugin.

- Calico

documentation for quickstart is [here](https://docs.tigera.io/calico/latest/getting-started/kubernetes/quickstart)
Right now we are using default settings as the author has no experience with tuning Calico.

Install Calico CRDs:
```
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.32.1/manifests/v1_crd_projectcalico_org.yaml
```
Install Tigera operator:
```
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.32.1/manifests/tigera-operator.yaml
```
Install Calico by creating the necessary custom resources.
```
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.32.1/manifests/custom-resources.yaml
```