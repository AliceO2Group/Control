# Dummy process example for OCC library

This example is built from the top-level CMakeLists.txt when `BUILD_EXAMPLES` is true.

For instructions on running it, see [Run example](../../../README.md#run-example).

## Standalone build

For guidelines on building the example as a standalone project, see [CMakeLists.txt.example](https://github.com/AliceO2Group/Control/blob/master/occ/occlib/examples/dummy-process/CMakeLists.txt.example).

Dependencies in aliBuild:

* Control-OCCPlugin (provides the OCC library), which in turn requires
    * boost (for boost::program_options)
    * grpc
    * protobuf
    * FairMQ + FairLogger (only for the OCC plugin, not linked by OCC library)
