[![JIRA](https://img.shields.io/badge/JIRA-Report%20issue-blue.svg)](https://alice.its.cern.ch/jira/secure/CreateIssue.jspa?pid=11232&issuetype=1)
[![godoc](https://img.shields.io/badge/godoc-Reference-5272B4.svg)](https://godoc.org/github.com/AliceO2Group/Control)
# AliECS

The ALICE Experiment Control System (**AliECS**) is the piece of software to drive and control data taking activities in the experiment.
It is a distributed system that combines state of the art cluster resource management and experiment control functionalities into a single comprehensive solution.

Please refer to the [CHEP 2023 paper](https://doi.org/10.1051/epjconf/202429502027) for the latest design overview.

## How to get started

Regardless of your particular interests, it is recommended to get acquainted with the main [AliECS concepts](docs/handbook/concepts.md).

After that, please find your concrete use case:

### I want to **run AliECS** and other O²/FLP software

See [O²/FLP Suite deployment instructions](https://alice-flp.docs.cern.ch/system-configuration/utils/o2-flp-setup/)

These instructions apply to both single-node and multi-node deployments.
Contact [alice-o2-flp-support](mailto:alice-o2-flp-support@cern.ch) for assistance with provisioning and deployment.

There are two ways of interacting with AliECS:

- The AliECS GUI (a.k.a. Control GUI, COG) - not in this repository, but included in most deployments, recommended

  :arrow_right: [AliECS GUI documentation](hacking/COG.md)

- `coconut` - the command-line control and configuration utility, included with AliECS core, typically for developers and advanced users

  :arrow_right: [Using `coconut`](https://alice-flp.docs.cern.ch/aliecs/coconut/)

  :arrow_right: [`coconut` command reference](https://alice-flp.docs.cern.ch/aliecs/coconut/doc/coconut/)
    
### I want to ensure AliECS can **run and control my process**

* **My software is based on FairMQ and/or O² DPL (Data Processing Later)**
  
    AliECS natively supports FairMQ (and DPL) devices.
    Head to [ControlWorkflows](https://github.com/AliceO2Group/ControlWorkflows) for instructions on how to configure your software to be controlled by AliECS.
  
* **My software does not use FairMQ and/or DPL, but should be controlled through a state machine**
  
    See [the OCC documentation](occ/README.md) to learn how to integrate the O² Control and Configuration library with your software. [Readout](https://github.com/AliceO2Group/Readout) is an example of this setup.

    Once ready, head to [ControlWorkflows](https://github.com/AliceO2Group/ControlWorkflows) for instructions on how to configure it to be controlled by AliECS.

* **My software is a command line utility with no state machine**
  
    AliECS natively supports generic commands.
    Head to [ControlWorkflows](https://github.com/AliceO2Group/ControlWorkflows) for instructions to have your command ran by AliECS.
    Make sure the task template for your command sets the control mode to `basic` ([see example](https://github.com/AliceO2Group/ControlWorkflows/blob/master/tasks/o2-roc-cleanup.yaml)).
    
### I want to develop AliECS

:hammer_and_wrench: Welcome to the team, please head to [contributing instructions](/docs/CONTRIBUTING.md)

### I want to receive updates about environments or services controlled by AliECS

:pager: Learn more about the [kafka event service](/docs/kafka.md)

### I want my application to send requests to AliECS

:scroll: See the API docs of AliECS components:

- [core gRPC server](/docs/apidocs_aliecs.md)
- [apricot gRPC server](/docs/apidocs_apricot.md)
- [apricot HTTP server](/apricot/docs/apricot_http_service.md)

### I want my service to be sent requests by AliECS

:electric_plug: Learn more about the [plugin system](/core/integration/README.md)

## Table of Contents

* Introduction
  * [Basic Concepts](/docs/handbook/concepts.md#basic-concepts)
    * [Tasks](/docs/handbook/concepts.md#tasks)
    * [Workflows, roles and environments](/docs/handbook/concepts.md#workflows-roles-and-environments)
  * [Design Overview](/docs/handbook/overview.md#design-overview)
    * [AliECS Structure](/docs/handbook/overview.md#aliecs-structure)
    * [Resource Management](/docs/handbook/overview.md#resource-management)
    * [FairMQ](/docs/handbook/overview.md#fairmq)
    * [State machines](/docs/handbook/overview.md#state-machines)

* Component reference
  * AliECS GUI
    * [AliECS GUI overview](/hacking/COG.md)
  * AliECS core
    * [Workflow Configuration](/docs/handbook/configuration.md#workflow-configuration)
      * [The AliECS workflow template language](/docs/handbook/configuration.md#the-aliecs-workflow-template-language)
      * [Workflow template structure](/docs/handbook/configuration.md#workflow-template-structure)
        * [Task roles](/docs/handbook/configuration.md#task-roles)
        * [Call roles](/docs/handbook/configuration.md#call-roles)
        * [Aggregator roles](/docs/handbook/configuration.md#aggregator-roles)
        * [Iterator roles](/docs/handbook/configuration.md#iterator-roles)
        * [Include roles](/docs/handbook/configuration.md#include-roles)
        * [Template expressions](/docs/handbook/configuration.md#template-expressions)
    * [Task Configuration](/docs/handbook/configuration.md#task-configuration)
      * [Task template structure](/docs/handbook/configuration.md#task-template-structure)
      * [Variables pushed to controlled tasks](/docs/handbook/configuration.md#variables-pushed-to-controlled-tasks)
      * [Resource wants and limits](/docs/handbook/configuration.md#resource-wants-and-limits)
      * [EPN workflow generation](/docs/handbook/configuration.md#epn-workflow-generation)
    * [Integration plugins](/core/integration/README.md#integration-plugins)
      * [Plugin system overview](/core/integration/README.md#plugin-system-overview)
    * [Integrated service operations](/core/integration/README.md#integrated-service-operations)
      * [Bookkeeping](/core/integration/README.md#bookkeeping)
      * [CCDB](/core/integration/README.md#ccdb)
      * [DCS](/core/integration/README.md#dcs)
        * [DCS operations](/core/integration/README.md#dcs-operations)
        * [DCS PrepareForRun behaviour](/core/integration/README.md#dcs-prepareforrun-behaviour)
        * [DCS StartOfRun behaviour](/core/integration/README.md#dcs-startofrun-behaviour)
        * [DCS EndOfRun behaviour](/core/integration/README.md#dcs-endofrun-behaviour)
        * [ECS2DCS2ECS mock server](/core/integration/README.md#ecs2dcs2ecs-mock-server)
      * [DD Scheduler](/core/integration/README.md#dd-scheduler)
      * [Kafka (legacy)](/core/integration/README.md#kafka-legacy)
      * [LHC](/core/integration/README.md)
      * [ODC](/core/integration/README.md#odc)
      * [Test plugin](/core/integration/README.md#test-plugin)
      * [Trigger](/core/integration/README.md#trigger)
    * [Environment operation order](/docs/handbook/operation_order.md#environment-operation-order)
      * [State machine triggers](/docs/handbook/operation_order.md#state-machine-triggers)
      * [START_ACTIVITY (Start Of Run)](/docs/handbook/operation_order.md#start_activity-start-of-run)
      * [STOP_ACTIVITY (End Of Run)](/docs/handbook/operation_order.md#stop_activity-end-of-run)
      * [Virtual states and transitions](/docs/handbook/operation_order.md#virtual-states-and-transitions)
    * [Protocol documentation](/docs/apidocs_aliecs.md)
  * coconut
    * [The O² control and configuration utility overview](/coconut/README.md#the-o-control-and-configuration-utility-overview)
      * [Configuration file](/coconut/README.md#configuration-file)
      * [Using coconut](/coconut/README.md#using-coconut)
        * [Creating an environment](/coconut/README.md#creating-an-environment)
        * [Controlling an environment](/coconut/README.md#controlling-an-environment)
    * [Command reference](/coconut/doc/coconut.md)
  * apricot
    * [ALICE configuration service overview](/apricot/README.md#alice-configuration-service-overview)
    * [HTTP service](/apricot/docs/apricot_http_service.md#apricot-http-service)
      * [Configuration](/apricot/docs/apricot_http_service.md#configuration)
      * [Usage and options](/apricot/docs/apricot_http_service.md#usage-and-options)
      * [Examples](/apricot/docs/apricot_http_service.md#examples)
    * [Protocol documentation](/docs/apidocs_apricot.md)
    * [Command reference](/apricot/docs/apricot.md)
  * occ
    * [O² Control and Configuration Components](/occ/README.md#o-control-and-configuration-components)
      * [Developer quick start instructions for OCClib](/occ/README.md#developer-quick-start-instructions-for-occlib)
      * [Manual build instructions](/occ/README.md#manual-build-instructions)
      * [Run example](/occ/README.md#run-example)
      * [The OCC state machine](/occ/README.md#the-occ-state-machine)
      * [Single process control with peanut](/occ/README.md#single-process-control-with-peanut)
      * [OCC API debugging with grpcc](/occ/README.md#occ-api-debugging-with-grpcc)
    * [Dummy process example for OCC library](/occ/occlib/examples/dummy-process/README.md#dummy-process-example-for-occ-library)
    * [Protocol documentation](/docs/apidocs_occ.md)
  * peanut
    * [Process control and execution utility overview](/occ/peanut/README.md)
  * Event service
    * [Kafka producer functionality in AliECS core](/docs/kafka.md#kafka-producer-functionality-in-aliecs-core)
      * [Making sure that AliECS sends messages](/docs/kafka.md#making-sure-that-aliecs-sends-messages)
      * [Currently available topics](/docs/kafka.md#currently-available-topics)
      * [Decoding the messages](/docs/kafka.md#decoding-the-messages)
    * [Legacy events: Kafka plugin](/docs/kafka.md#legacy-events-kafka-plugin)
      * [Making sure that AliECS sends messages](/docs/kafka.md#making-sure-that-aliecs-sends-messages-1)
      * [Currently available topics](/docs/kafka.md#currently-available-topics-1)
      * [Decoding the messages](/docs/kafka.md#decoding-the-messages-1)
      * [Getting Start of Run and End of Run notifications](/docs/kafka.md#getting-start-of-run-and-end-of-run-notifications)
      * [Using Kafka debug tools](/docs/kafka.md#using-kafka-debug-tools)

* Developer documentation
  * [Contributing](/docs/CONTRIBUTING.md)
  * [Package pkg.go.dev documentation](https://pkg.go.dev/github.com/AliceO2Group/Control)
  * [Building AliECS](/docs/building.md#building-aliecs)
    * [Overview](/docs/building.md#overview)
    * [Building with aliBuild](/docs/building.md#building-with-alibuild)
    * [Manual build](/docs/building.md#manual-build)
      * [Go environment](/docs/building.md#go-environment)
      * [Clone and build (Go components only)](/docs/building.md#clone-and-build-go-components-only)
  * [Makefile reference](/docs/makefile_reference.md)
  * [Component Configuration](/docs/handbook/appconfiguration.md#component-configuration)
    * [Apache Mesos](/docs/handbook/appconfiguration.md#apache-mesos)
      * [Connectivity to controlled nodes](/docs/handbook/appconfiguration.md#connectivity-to-controlled-nodes)
  * [Running AliECS as a developer](/docs/running.md#running-aliecs-as-a-developer)
    * [Running the AliECS core](/docs/running.md#running-the-aliecs-core)
  * [Running AliECS in production](/docs/running.md#running-aliecs-in-production)
    * [Health checks](/docs/running.md#health-checks)
  * [Development Information](/docs/development.md#development-information)
    * [Release Procedure](/docs/development.md#release-procedure)
  * [Metrics in ECS](/docs/metrics.md#metrics-in-ecs)
    * [Overview and simple usage](/docs/metrics.md#overview-and-simple-usage)
    * [Types and aggregation of metrics](/docs/metrics.md#types-and-aggregation-of-metrics)
      * [Metric types](/docs/metrics.md#metric-types)
      * [Aggregation](/docs/metrics.md#aggregation)
    * [Implementation details](/docs/metrics.md#implementation-details)
      * [Event loop](/docs/metrics.md#event-loop)
      * [Hashing to aggregate](/docs/metrics.md#hashing-to-aggregate)
      * [Sampling reservoir](/docs/metrics.md#sampling-reservoir)
  * [OCC API debugging with grpcc](/docs/using_grpcc_occ.md#occ-api-debugging-with-grpcc)
  * [Running tasks inside docker](/docs/running_docker.md#running-a-task-inside-a-docker-container)
* Resources
  * T. Mrnjavac et. al, [AliECS: A New Experiment Control System for the ALICE Experiment](https://doi.org/10.1051/epjconf/202429502027), CHEP23

