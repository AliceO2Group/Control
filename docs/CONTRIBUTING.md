# Contributing

Thank you for your interest in contributing to the project.
This document provides guidelines and information to help you contribute effectively.

If you are not in contact with the project maintainers, please reach out to them before proposing any changes.
We use JIRA for issue tracking and project management.
This software component is part of the O²/FLP project in the ALICE experiment.

## Getting started

Getting acquainted with the introduction chapters is absolutely essential, glossing over the whole documentation is highly advised.

A development environment setup will be necessary for compiling binaries and running unit tests, see [Building](/docs/building.md) for details.

## Testing

Run unit tests in the Control project with `make test`.
To obtain test coverage reports, run `make coverage`.

Typically, you will also want to prepare a test setup in form of an [FLP suite deployment](https://alice-flp.docs.cern.ch/system-configuration/utils/o2-flp-setup/) on a virtual machine.
Since AliECS interacts with many other project components, the last testing step might involve replacing the modified binary on the test VM and trying out the new functionality or the fix.

The binaries are installed at `/opt/o2/bin`.

`o2-aliecs-core` and `o2-apricot` are ran as systemd services, so you will need to restart them after replacing the binary.

`o2-aliecs-executor` is started by `mesos-slave` if it is not running already at environment creation.
To make sure that the replaced binary is used, kill the running process (`pkill -f o2-aliecs-executor`).

## Pull requests guidelines

- Make sure your work has a corresponding JIRA ticket and it is assigned to yourself.
Trivial PRs are acceptable without a ticket.

- Work on your changes in your fork on a dedicated branch with a descriptive name.

- Make focused, logically atomic commits with clear messages and descriptions explaining the design choices.
Multiple commits per pull request are allowed.
However, please make sure that the project can be built and the tests pass at any commit.

- Commit message or description should include the JIRA ticket number

- Add tests for your changes whenever possible.
Gomega/Ginkgo tests are preferred, but other style of tests are also welcome.

- Add documentation for new features.

- Your contribution will be reviewed by the project maintainers once the PR is marked as ready for review.

## Documentation guidelines

The markdown documentation is aimed to be browsed on GitHub, but it also on the aggregated [FLP documentation](https://alice-flp.docs.cern.ch) based on [MkDocs](https://www.mkdocs.org/).
Consequently, any changes in the documentation structure should be reflected in the Table of Contents in the main README.md, as well as `mkdocs.yml` and `mkdocs.yml`.

The AliECS MkDocs documentation is split into two aforementioned files to follow the split between "Products" and "Developers" tabs in the FLP documentation.
The `mkdocs-dev.yml` uses a symlink `aliecs-dev` to `aliecs` directory to avoid complaints about duplicated site names.

Because of the dual target of the documentation, the points below are important to keep in mind:

- Absolute paths in links to other files do not always work, they should be avoided.
- When referencing source files in the repository, use full URIs to GitHub.
- In MkDocs layouts, one cannot reference specific sections within markdown files. Only links to entire markdown files are possible.