[![JIRA](https://img.shields.io/badge/JIRA-Report%20issue-blue.svg)](https://alice.its.cern.ch/jira/secure/CreateIssue.jspa?pid=11232&issuetype=1)
[![godoc](https://img.shields.io/badge/godoc-Reference-5272B4.svg)](https://godoc.org/github.com/AliceO2Group/Control)
# AliECS

The ALICE Experiment Control System

## Install instructions

What is your use case?

* I want to run AliECS (and other O²/FLP software) on a **single machine**

    :arrow_right: [Single node O²/FLP software deployment instructions](https://gitlab.cern.ch/AliceO2Group/system-configuration/blob/master/ansible/docs/O2_INSTALL_FLP_STANDALONE.md)

* I want to run AliECS (and other O²/FLP software) in a **multi-node setup**

    :construction: No user instructions yet - contact [alice-o2-flp-support](mailto:alice-o2-flp-support@cern.ch) for assistance with provisioning and deployment.
    
* I want to ensure AliECS can **run and control my process**

    * My software is based on FairMQ and/or O² DPL
    
        :palm_tree: Nothing to do, AliECS natively supports FairMQ (and DPL) devices.
    
    * My software does not use FairMQ and/or DPL, but should be controlled through a state machine
    
        :telescope: See [the OCC documentation](occ/README.md) to learn how to integrate the O² Control and Configuration library with your software. [Readout](https://github.com/AliceO2Group/Readout) is currently the only example of this setup.
    
* I want to build and run AliECS for **development** purposes

    :hammer_and_wrench: [Building instructions](hacking/BUILDING.md)
    
    :arrow_right: [Running instructions](hacking/RUNNING.md)

## Using AliECS

There are two ways of interacting with AliECS:
 
* The AliECS GUI - not in this repository, but included in most deployments, recommended

    :arrow_right: [AliECS GUI documentation](hacking/COG.md)
    
* `coconut` - the command-line control and configuration utility, included with AliECS core

    :arrow_right: [Using `coconut`](coconut/README.md)
    
    :arrow_right: [`coconut` command reference](coconut/doc/coconut.md)
