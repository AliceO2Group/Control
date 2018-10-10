# Originally from https://raw.githubusercontent.com/wastl/cmarmotta/master/cmake/FindGRPC.cmake

find_program(GRPC_CPP_PLUGIN grpc_cpp_plugin) # Get full path to plugin

if (GRPC_CPP_PLUGIN)
    get_filename_component(GRPC_BINDIR ${GRPC_CPP_PLUGIN} DIRECTORY)
    get_filename_component(GRPC_ROOT ${GRPC_BINDIR} DIRECTORY)
    list(APPEND CMAKE_PREFIX_PATH ${GRPC_ROOT})
endif()

find_library(GRPC_LIBRARY NAMES grpc)
find_library(GRPCPP_LIBRARY NAMES grpc++)
find_library(GPR_LIBRARY NAMES gpr)
find_path(GRPC_INCLUDE_DIR grpcpp/grpcpp.h)

set(GRPC_LIBRARIES ${GRPCPP_LIBRARY} ${GRPC_LIBRARY} ${GPR_LIBRARY})
if (GRPC_LIBRARIES)
    message(STATUS "Found GRPC: ${GRPC_LIBRARIES}; plugin - ${GRPC_CPP_PLUGIN}")
endif ()

# Generate namespaced library targets
if(GRPC_LIBRARY)
    if(NOT TARGET grpc::grpc)
        add_library(grpc::grpc UNKNOWN IMPORTED)
        set_target_properties(grpc::grpc PROPERTIES
            INTERFACE_INCLUDE_DIRECTORIES "${GRPC_INCLUDE_DIR}")
        if(EXISTS "${GRPC_LIBRARY}")
            set_target_properties(grpc::grpc PROPERTIES
                IMPORTED_LOCATION "${GRPC_LIBRARY}")
        endif()
        if(EXISTS "${GRPC_LIBRARY_RELEASE}")
            set_property(TARGET grpc::grpc APPEND PROPERTY
                IMPORTED_CONFIGURATIONS RELEASE)
            set_target_properties(grpc::grpc PROPERTIES
                IMPORTED_LOCATION_RELEASE "${GRPC_LIBRARY_RELEASE}")
        endif()
        if(EXISTS "${GRPC_LIBRARY_DEBUG}")
            set_property(TARGET grpc::grpc APPEND PROPERTY
                IMPORTED_CONFIGURATIONS DEBUG)
            set_target_properties(grpc::grpc PROPERTIES
                IMPORTED_LOCATION_DEBUG "${GRPC_LIBRARY_DEBUG}")
        endif()
    endif()
endif()
if(GPR_LIBRARY)
    if(NOT TARGET grpc::gpr)
        add_library(grpc::gpr UNKNOWN IMPORTED)
        set_target_properties(grpc::gpr PROPERTIES
            INTERFACE_INCLUDE_DIRECTORIES "${Protobuf_INCLUDE_DIR}")
        if(EXISTS "${GPR_LIBRARY}")
            set_target_properties(grpc::gpr PROPERTIES
                IMPORTED_LOCATION "${GPR_LIBRARY}")
        endif()
        if(EXISTS "${GPR_LIBRARY_RELEASE}")
            set_property(TARGET grpc::gpr APPEND PROPERTY
                IMPORTED_CONFIGURATIONS RELEASE)
            set_target_properties(grpc::gpr PROPERTIES
                IMPORTED_LOCATION_RELEASE "${GPR_LIBRARY_RELEASE}")
        endif()
        if(EXISTS "${GPR_LIBRARY_DEBUG}")
            set_property(TARGET grpc::gpr APPEND PROPERTY
                IMPORTED_CONFIGURATIONS DEBUG)
            set_target_properties(grpc::gpr PROPERTIES
                IMPORTED_LOCATION_DEBUG "${GPR_LIBRARY_DEBUG}")
        endif()
    endif()
endif()
if(GRPCPP_LIBRARY)
    if(NOT TARGET grpc::grpc++)
        add_library(grpc::grpc++ UNKNOWN IMPORTED)
        set_target_properties(grpc::grpc++ PROPERTIES
            INTERFACE_INCLUDE_DIRECTORIES "${Protobuf_INCLUDE_DIR}")
        if(EXISTS "${GRPCPP_LIBRARY}")
            set_target_properties(grpc::grpc++ PROPERTIES
                IMPORTED_LOCATION "${GRPCPP_LIBRARY}")
        endif()
        if(EXISTS "${GRPCPP_LIBRARY_RELEASE}")
            set_property(TARGET grpc::grpc++ APPEND PROPERTY
                IMPORTED_CONFIGURATIONS RELEASE)
            set_target_properties(grpc::grpc++ PROPERTIES
                IMPORTED_LOCATION_RELEASE "${GRPCPP_LIBRARY_RELEASE}")
        endif()
        if(EXISTS "${GRPCPP_LIBRARY_DEBUG}")
            set_property(TARGET grpc::grpc++ APPEND PROPERTY
                IMPORTED_CONFIGURATIONS DEBUG)
            set_target_properties(grpc::grpc++ PROPERTIES
                IMPORTED_LOCATION_DEBUG "${GRPCPP_LIBRARY_DEBUG}")
        endif()
    endif()
endif()

# Set all variables with FPHSA
include(FindPackageHandleStandardArgs)
find_package_handle_standard_args(GRPC
    REQUIRED_VARS GRPC_INCLUDE_DIR GPR_LIBRARY GRPC_LIBRARY GRPCPP_LIBRARY GRPC_CPP_PLUGIN
    VERSION_VAR GRPC_VERSION)

if(GRPC_FOUND)
  set(GRPC_INCLUDE_DIRS ${GRPC_INCLUDE_DIR})
  set(GRPC_INCLUDE_DIRS ${GRPC_INCLUDE_DIR})
  if(MSVC)
    set(GRPC_LIBRARIES ${GPR_LIBRARY} ${GRPC_LIBRARY} ${GRPCPP_LIBRARY} ws2_32)
  else()
    set(GRPC_LIBRARIES ${GPR_LIBRARY} ${GRPC_LIBRARY} ${GRPCPP_LIBRARY})
  endif()
endif()

# Protobuf+gRPC generator wrapper
function(PROTOBUF_GENERATE_GRPC_CPP SRCS HDRS)
    if (NOT ARGN)
        message(SEND_ERROR "Error: PROTOBUF_GENERATE_GRPC_CPP() called without any proto files")
        return()
    endif ()

    if (PROTOBUF_GENERATE_CPP_APPEND_PATH) # This variable is common for all types of output.
        # Create an include path for each file specified
        foreach (FIL ${ARGN})
            get_filename_component(ABS_FIL ${FIL} ABSOLUTE)
            get_filename_component(ABS_PATH ${ABS_FIL} PATH)
            list(FIND _protobuf_include_path ${ABS_PATH} _contains_already)
            if (${_contains_already} EQUAL -1)
                list(APPEND _protobuf_include_path -I ${ABS_PATH})
            endif ()
        endforeach ()
    else ()
        set(_protobuf_include_path -I ${CMAKE_CURRENT_SOURCE_DIR})
    endif ()

    if (DEFINED PROTOBUF_IMPORT_DIRS)
        foreach (DIR ${PROTOBUF_IMPORT_DIRS})
            get_filename_component(ABS_PATH ${DIR} ABSOLUTE)
            list(FIND _protobuf_include_path ${ABS_PATH} _contains_already)
            if (${_contains_already} EQUAL -1)
                list(APPEND _protobuf_include_path -I ${ABS_PATH})
            endif ()
        endforeach ()
    endif ()

    set(${SRCS})
    set(${HDRS})
    foreach (FIL ${ARGN})
        get_filename_component(ABS_FIL ${FIL} ABSOLUTE)
        get_filename_component(FIL_WE ${FIL} NAME_WE)

        list(APPEND ${SRCS} "${CMAKE_CURRENT_BINARY_DIR}/${FIL_WE}.grpc.pb.cc")
        list(APPEND ${HDRS} "${CMAKE_CURRENT_BINARY_DIR}/${FIL_WE}.grpc.pb.h")

        add_custom_command(
            OUTPUT "${CMAKE_CURRENT_BINARY_DIR}/${FIL_WE}.grpc.pb.cc"
            "${CMAKE_CURRENT_BINARY_DIR}/${FIL_WE}.grpc.pb.h"
            COMMAND ${PROTOBUF_PROTOC_EXECUTABLE}
            ARGS --grpc_out=${CMAKE_CURRENT_BINARY_DIR}
            --plugin=protoc-gen-grpc=${GRPC_CPP_PLUGIN}
            ${_protobuf_include_path} ${ABS_FIL}
            DEPENDS ${ABS_FIL} ${PROTOBUF_PROTOC_EXECUTABLE}
            COMMENT "Running gRPC C++ protocol buffer compiler on ${FIL}"
            VERBATIM)
    endforeach ()

    set_source_files_properties(${${SRCS}} ${${HDRS}} PROPERTIES GENERATED TRUE)
    set(${SRCS} ${${SRCS}} PARENT_SCOPE)
    set(${HDRS} ${${HDRS}} PARENT_SCOPE)
endfunction()

