# Running a Task Inside a Docker Container

> ⚠️ **Warning**
> This method is **not intended for production use**.
> It serves only as a **proof of concept** for testing Docker images as part of an existing pipeline.
>
> Currently, it has been tested with the `alma9-flp-node` image running the *readout* component.

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
sudo /usr/bin/docker run --name readout --replace \
  --user "$(id -u flp):$(id -u flp)" \
  --network=host --ipc=host \
  -e O2_DETECTOR -e O2_PARTITION -e OCC_CONTROL_PORT \
  -e O2_SYSTEM -e O2_ROLE \
  gitlab-registry.cern.ch/aliceo2group/dockerfiles/alma9-flp-node:2 \
  /opt/o2/bin/o2-readout-exe
```

> 🧩 **Note**
> We are not claiming that this is the most efficient way how to run this image, just that it works.

#### Environment Variables

To identify all required environment variables:

1. Open the **ECS GUI**.
2. Go to the **Environment Details** page for the relevant task.
3. Review the variables defined there — these match those used when running the binary outside Docker.

#### Shared Memory Communication

To enable shared memory communication between processes, add the `--ipc=host` flag when running the container.
However, doing so requires **elevated privileges**.

While **Podman** can run without root privileges, it pauses other Podman processes for the same user.
This means commands like `podman ps -a` or starting multiple containers in parallel will not work.

Therefore, you should run containers using the same user as the rest of the pipeline:

```bash
--user "$(id -u flp):$(id -u flp)"
```

This ensures shared memory segments are created under the same user context.

---

## Tips and Tricks

* Production systems running RHEL do not install native Docker via:

  ```bash
  dnf install docker
  ```

  Instead, they use **Podman**, which emulates Docker’s behavior but may differ in certain aspects.

* To check whether ECS has started a container, run:

  ```bash
  docker ps -a
  ```

  This lists all containers that have run (or are currently running) under the current user.

  > ECS typically runs as the `flp` user, so to inspect its containers, switch users first:
  >
  > ```bash
  > su - flp
  > ```

* To view container logs directly from Docker:

  ```bash
  docker logs <container-id|name>
  ```

---
