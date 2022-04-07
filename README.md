[![JIRA](https://img.shields.io/badge/JIRA-Report%20issue-blue.svg)](https://alice.its.cern.ch/jira/secure/CreateIssue.jspa?pid=11232&issuetype=1)
[![godoc](https://img.shields.io/badge/godoc-Reference-5272B4.svg)](https://godoc.org/github.com/AliceO2Group/Control)
# AliECS

The ALICE Experiment Control System

## Install instructions

What is your use case?

* I want to **run AliECS** and other O²/FLP software

    :arrow_right: [O²/FLP Suite deployment instructions](https://alice-flp.docs.cern.ch/system-configuration/utils/o2-flp-setup/)

    These instructions apply to both single-node and multi-node deployments.

    Contact [alice-o2-flp-support](mailto:alice-o2-flp-support@cern.ch) for assistance with provisioning and deployment.
    
* I want to ensure AliECS can **run and control my process**

    * My software is based on FairMQ and/or O² DPL
    
        :palm_tree: Nothing to do, AliECS natively supports FairMQ (and DPL) devices.
    
    * My software does not use FairMQ and/or DPL, but should be controlled through a state machine
    
        :telescope: See [the OCC documentation](occ/README.md) to learn how to integrate the O² Control and Configuration library with your software. [Readout](https://github.com/AliceO2Group/Readout) is currently the only example of this setup.
        
    * My software is a command line utility with no state machine
    
        :palm_tree: Nothing to do, AliECS natively supports generic commands. Make sure the task template for your command sets the control mode to `basic` ([see example](https://github.com/AliceO2Group/ControlWorkflows/blob/basic-tasks/tasks/sleep.yaml)).
    
* I want to build and run AliECS for **development** purposes

    :hammer_and_wrench: [Building instructions](https://alice-flp.docs.cern.ch/aliecs/building/)
    
    :arrow_right: [Running instructions](https://alice-flp.docs.cern.ch/aliecs/running/)

* I want to communicate with AliECS via one of the plugins
    
    * [Receive updates on running environments via Kafka](docs/kafka.md)

## Using AliECS

There are two ways of interacting with AliECS:
 
* The AliECS GUI - not in this repository, but included in most deployments, recommended

    :arrow_right: [AliECS GUI documentation](hacking/COG.md)
    
* `coconut` - the command-line control and configuration utility, included with AliECS core

    :arrow_right: [Using `coconut`](https://alice-flp.docs.cern.ch/aliecs/coconut/)
    
    :arrow_right: [`coconut` command reference](https://alice-flp.docs.cern.ch/aliecs/coconut/doc/coconut/)
