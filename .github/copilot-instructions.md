## ALICE Experiment Control System (AliECS)

The ALICE Experiment Control System (**AliECS**) is the piece of software to drive and control data taking activities in the experiment.
It is a distributed system that combines state of the art cluster resource management and experiment control functionalities into a single comprehensive solution.

### Copilot agent instructions

- Running `make` in the main directory lets you build all golang components of AliECS.
- Running `make test` in the main directory lets you run all golang tests of AliECS.
- Do not attempt to build C++ components unless you already have access to their dependencies (FairMQ and FairLogger) and the task requires you to do it.
- When changing API interfaces, run `make docs` to regenerate the API documentation.
- When changing gRPC interfaces and proto files, run `make generate` to regenerate the gRPC code.
- Do not include regenerated Go files in your commits unless you are changing the API or proto files. If you do, include only the files which are related to your task.
- Do not include modified go.mod and go.sum files in your commits unless you are adding or upgrading a dependency.
- When adding features or fixing bugs, add a corresponding unit test if feasible. Use Ginkgo/Gomega, unless a package already uses a different testing framework.
- When adding a new feature, extend the documentation accordingly, follow the existing style and structure, and make sure that Tables of Contents are updated. See [Documentation guidelines](docs/CONTRIBUTING.md#documentation-guidelines) for some more details.
- When providing a fix, explain what was causing the issue and how it was fixed.
- Avoid using abbreviations and acronyms for variable and function names. They are acceptable if they are commonly used in a given domain, such as "cfg" for "configura    +++tion" or "LHC" for "Large Hadron Collider".
