# Makefile Reference

AliECS comes with a `make`-based build system, with all `.PHONY` targets.

| Command     | Description |
|-------------|-------------|
| `make` | builds all AliECS components, output in `./bin` |
| `make all` | equivalent to `make vendor && make generate && make build`
| `make build` | same as `make` |
| `make generate` | runs `go generate` as configured (mostly for generating gRPC/Protobuf stub code) |
| `make test` | runs unit tests |
| `make debugtest` | runs unit tests in verbose mode |
| `make vet` | runs `go vet` on all source directories |
| `make fmt` | runs `go fmt` on all source directories |
| `make clean` | cleans the default build output directory (`./bin`) |
| `make cleanall` | cleans the default build output directory (`./bin`, same as `make clean`), as well as `./tools` and `./vendor` |
| `make vendor` | rebuilds/refreshes the tree of vendored dependencies (`go mod vendor`) and fetches 3rd party Protobuf files |
| `make tools` | ensures all build tools are present (currently only `protoc-gen-go`) |
| `make tools/protoc` | ensures `protoc-gen-go` is present (included in `make tools`) |
| `make doc`<br>`make docs` | regenerates command reference documentation for command line tools |
| `make help` | displays inline documentation |
| `make coverage` | builds a test coverage report |

The variable `WHAT` is obeyed by `make build` (or `make`) and `make install` in order to customize the components to build. For example `make WHAT=coconut install` builds and installs only `coconut`. By default `WHAT` includes all components.

Add `DEBUG=1` before `make` to enable non-optimized, debug builds.
