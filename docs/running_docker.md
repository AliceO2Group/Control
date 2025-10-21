# Running a Task Inside a Docker Container

> ⚠️ **Warning**
> This method is **not intended for production use**.
> It serves only as a **proof of concept** for testing Docker images as part of an existing pipeline.
>
> Currently, it was tested with the `alma9-flp-node` image running the *readout* component with CRU.

---

## How To

### 1. Manual Setup

Before running tasks in Docker, ensure that the host machine has Docker installed.
At the time of writing, Docker must be installed **manually**.

> ⚠️ **Security Note**
> The `flp` user must be able to run `sudo` **without a password**, because Docker requires root privileges because of inter process communication requirements
>
> This setup is **potentially unsafe** for production systems. There exists rootless mode in Podman (alias for Docker at RHEL) which might solve safety
> issues. However, we were not able to make this work for more than one container because of ipc requirements.

Run the following commands as `root` to add `flp` user to sudoers:

```bash
usermod -aG wheel flp
echo '%wheel ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/90-wheel-nopasswd
```

---

### 2. Modifying ControlWorkflows

To run a task inside a Docker container on the executor, wrap the binary call in a `docker run` command within the [ControlWorkflows](https://github.com/AliceO2Group/ControlWorkflows) repository.

For example, to run readout, modify the `_plain_command` section of [`readout.yaml`](https://github.com/AliceO2Group/ControlWorkflows/blob/master/tasks/readout.yaml) by adding a Docker command.

> 🧩 **Note**
> You must already have a Docker image that includes the required binary and configuration.
> (Creating such an image is outside the scope of this document.)

#### Example Command

When running readout, we successfully used the following command inside the `alma9-flp-node` image:

```bash
sudo -E docker run --name readout --replace \
  --user flp -v /etc/group:/etc/group:ro -v /etc/passwd:/etc/passwd:ro \
  --privileged \
  --network=host --ipc=host \
  -v /tmp:/tmp \
  -v /lib/modules/$(uname -r):/lib/modules/$(uname -r) \
  -e O2_DETECTOR -e O2_PARTITION -e OCC_CONTROL_PORT -e O2_SYSTEM -e O2_ROLE \
  gitlab-registry.cern.ch/aliceo2group/dockerfiles/alma9-flp-node:2 \
  /opt/o2/bin/o2-readout-exe
```

> 🧩 **Note**
> We are not claiming that this is the most efficient way how to run this image, just that it works.
>
#### Explanation of command

Let's explain all parts of the `docker run` command and what is their purpose. As RHEL uses Podman by default
instead of docker we are going to comment on Podman parameters, but docker should be equivalent or pretty close.

- `sudo -E` switch to the rootful mode of Podman and pass all of the environment variables
- `docker run --name readout --replace` start container with the name readout and replace already existing container
- `--privileged` runs container with extended privileges and allows access to the devices and other resources
on host system with the same privileges as user running the container (`flp` in our case). Readout communicates
directly with CRUs and others connected via PCIe so we need to make these available inside the container.
- `--user flp -v /etc/group:/etc/group:ro -v /etc/passwd:/etc/passwd:ro` run container as flp user provide same user
and group settings as on a host machine. This is necessary as readout is using shared memory which is mapped under
flp user and needs access to [BAR](https://github.com/AliceO2Group/ReadoutCard?tab=readme-ov-file#bar-interface)
interfaces of CRU which belong to the `pda` group. We used this way of setting up user privileges and ids as there is
no guarantee that ids would match if we would hardwire user into docker image
- `--network=host` bind container's network to the host's. This allows services running outside of docker to communicate
with internals (eg. gRPC, IL, ... ). It should be changed to open just required ports.
- `--ipc=host` readout uses FairMQ which communicates through shared memory Inter-Process Communication.
As we are now running readout inside the docker with the rest of data distribution running bare metal we
need to have access to the hosting OS shared memory. This is the main requirement for running as rootful.
If we run container in rootless mode with this flag set, Podman switches into elevated privileges mode that blocks
other cli commands from the same user command until this command is finished unless running under root.
We tried to just bind `/dev/shm/` but readout failed on permission errors.
- `-v /tmp:/tmp` binds `/tmp` of host to the container for monitoring purposes.
- `-v /lib/modules/$(uname -r):/lib/modules/$(uname -r)` binds folder with host's kernel modules for the usage of PDA
used by readout itself.
- `e ...` pass environment variables required by readout to the container

#### Environment Variables

---

## Tips and Tricks

- To identify all required environment variables:

1. Open the **ECS GUI**.
2. Go to the **Environment Details** page for the relevant task.
3. Review the variables defined there — these match those used when running the binary outside Docker.

- Production systems running RHEL do not install native Docker via:

  ```bash
  dnf install docker
  ```

  Instead, they use Podman, which emulates Docker’s behavior but may differ in certain aspects.

- To check whether ECS has started a container, run:

  ```bash
  docker ps -a
  ```

  This lists all containers that have run (or are currently running) under the current user.

  > ECS typically runs as the `flp` user, so to inspect its containers, switch users first:
  >
  > ```bash
  > su - flp
  > ```

- To view container logs directly from Docker:

  ```bash
  docker logs <container-id|name>
  ```

---
