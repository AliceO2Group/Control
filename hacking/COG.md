# AliECS GUI overview

If you are using the [Single node O²/FLP software deployment instructions](https://gitlab.cern.ch/AliceO2Group/system-configuration/blob/master/ansible/docs/O2_INSTALL_FLP_STANDALONE.md), the AliECS GUI is automatically installed along with the full O²/FLP suite.

For development instructions, see [the AliECS GUI README](https://github.com/AliceO2Group/WebUi/blob/dev/Control/README.md).

## How do I...

* **view available workflow templates?**

On the left side, click on "Workflows". The listed workflow templates can be loaded into a new environment in order to start a run.

* **create a new environment?**

An environment is an instance of a workflow, plus any associated configuration data.

To create one, click on "Environments" on the left side. Make sure the lock icon in the top left is green (if not, click it to take the lock), then click on the "+" icon on the top right.

In the drop-down box, select a workflow template to use for the new environment. For example, `readout-qc-1` is a single-node workflow with an instance of Readout which feeds a QualityControl chain. When you are ready, click on "Create". This may take up to a minute, depending on your hardware and software configuration.

If the configuration is correct and the required resources are available, a new environment is created. Otherwise, an error message is shown. The new environment has an ID and a state. It has no run number, since a run number is associated with a time-constrained interval in the `RUNNING` state and not with the environment as a whole.

The new environment starts in state `CONFIGURED`.

* **control an environment?**

A newly created environment can immediately be controlled via the transition buttons at the top of the "Environment details" page. If you wish to control another environment, click on "Environments" on the left side, and then on "Details" for the environment you wish to control.

Clicking `START` performs the relevant transition for this environment. A run number is also generated, and all the tasks (listed at the bottom of the "Environment details" page) transition to the `RUNNING` state.

Clicking `STOP` performs the `STOP` transition, which stops the run and invalidates the run number.

* **destroy an environment?**

An environment must be in the `CONFIGURED` or `STANDBY` state in order to be destroyed. If this is not the case, transition the environment to one of these states before continuing.

Click on "Environments" on the left side, and then on "Shutdown" for the environment you wish to destroy. As this operation is irreversible, a pop-up prompt will ask you to confirm the environment shutdown.

By default, destroying an environment kills all the tasks involved.

* **modify the configuration of a specific task?**

In production, AliECS will manage and push all configuration to active tasks, but this is not handled by AliECS yet.

Every task still has their own configuration file, with paths such as `/etc/flp.d/qc/*.json` for QualityControl and `/home/flp/readout.cfg` for Readout. These paths can be edited by the user, and any changes affect all newly launched instances of the task.

All configuration file paths used by tasks can be found in the task descriptors of the workflow configuration repository in use. For more information on workflow configuration repositories, see [the `coconut repository` reference](/coconut/doc/coconut_repository.md). The default workflow configuration repository which comes pre-loaded with AliECS is accessible at [AliceO2Group/ControlWorkflows](https://github.com/AliceO2Group/ControlWorkflows) (all task descriptor files are found in the `tasks` directory).

* **modify an existing workflow or task?**

You are free to keep as many workflow configuration repositories as you wish in your AliECS instance. For more information on workflow configuration repositories, see [the `coconut repository` reference](/coconut/doc/coconut_repository.md).

Changes to a configuration repository are immediately available after running `coconut repo refresh`. There is no support in the AliECS GUI at this time.

* **run the same task with a different configuration?**

The template mechanism required for this is under development, but in the meantime the easiest way to achieve the "same task - different configurations" setup is by making a copy of the task descriptor (i.e. the file in the `tasks` directory which describes a task) with a different name, and changing the configuration inside (e.g. configuration file path, command line parameters, etc.).

Both task files are then available to be used in your workflow templates.
