set(protobuf_MODULE_COMPATIBLE TRUE)
find_package(Protobuf CONFIG REQUIRED)

protobuf_generate_cpp(PROTO_SRCS PROTO_HDRS test.proto)

add_library(o2-kaki-protos STATIC)
target_sources(o2-kaki-protos PRIVATE ${PROTO_SRCS} ${PROTO_HDRS})
target_compile_features(o2-kaki-protos PUBLIC cxx_std_20)
target_include_directories(o2-kaki-protos PUBLIC ${CMAKE_CURRENT_BINARY_DIR})
target_link_libraries(o2-kaki-protos PUBLIC protobuf::libprotobuf)
