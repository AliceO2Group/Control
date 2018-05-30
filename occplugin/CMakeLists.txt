# === This file is part of ALICE O² ===
#
# Copyright 2018 CERN and copyright holders of ALICE O².
# Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
#
# In applying this license CERN does not waive the privileges and
# immunities granted to it by virtue of its status as an
# Intergovernmental Organization or submit itself to any jurisdiction.

cmake_minimum_required(VERSION 3.9)
cmake_policy(SET CMP0028 NEW)

project(occplugin LANGUAGES CXX)

###
### Preamble: compiler stuff, CMake module paths
###

set(CMAKE_CXX_STANDARD 14)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_C_STANDARD 99)
set(CMAKE_C_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF) #we want std=c++14, not std=gnu++14

if (CMAKE_COMPILER_IS_GNUCXX)
    if (CMAKE_CXX_COMPILER_VERSION VERSION_GREATER 4.9 OR
        CMAKE_CXX_COMPILER_VERSION VERSION_EQUAL 4.9)
        message(STATUS "Found GNU g++ ${CMAKE_CXX_COMPILER_VERSION}, enabling colorized error messages.")
        set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fdiagnostics-color=auto")
    endif ()
endif ()

option(FAIRROOTPATH "The path to the FairRoot install directory" /usr)
set(FairRoot_DIR ${FAIRROOTPATH})

option(PROTOBUFPATH "The path to the Protobuf install directory" /usr)
option(GRPCPATH "The path to the gRPC install directory" /usr)
option(BOOSTPATH "The path to the Boost install directory" /usr)

set(CMAKE_PREFIX_PATH
    ${PROTOBUFPATH}
    ${GRPCPATH}
    ${BOOSTPATH}
    ${CMAKE_PREFIX_PATH})

set(CMAKE_MODULE_PATH
    ${CMAKE_MODULE_PATH}
    ${CMAKE_SOURCE_DIR}/cmake
    ${CMAKE_SOURCE_DIR}/cmake/modules
    ${FAIRROOTPATH}/share/fairbase/cmake/modules
    ${FAIRROOTPATH}/share/fairbase/cmake/modules_old)

include(FeatureSummary)

###
### Product information
###

set(OCCPLUGIN_ORGANIZATION_NAME "O²")
set(OCCPLUGIN_ORGANIZATION_DOMAIN "github.com/AliceO2Group/Control")
set(OCCPLUGIN_PRODUCT_NAME "OCC Plugin")
set(OCCPLUGIN_PRODUCT_MAINTAINER "Teo Mrnjavac <teo.mrnjavac@cern.ch>")
set(OCCPLUGIN_DESCRIPTION_SUMMARY "The O² Configuration and Control plugin for FairMQ devices")

set(OCCPLUGIN_VERSION_MAJOR 0)
set(OCCPLUGIN_VERSION_MINOR 1)
set(OCCPLUGIN_VERSION_PATCH 0)

set(OCCPLUGIN_VERSION ${OCCPLUGIN_VERSION_MAJOR}.${OCCPLUGIN_VERSION_MINOR}.${OCCPLUGIN_VERSION_PATCH})
set(OCCPLUGIN_VERSION_SHORT ${OCCPLUGIN_VERSION})

message(STATUS "Build type: ${CMAKE_BUILD_TYPE}")
set(GIT_REPOSITORY_ROOT "${CMAKE_SOURCE_DIR}/..")
if (NOT (CMAKE_BUILD_TYPE STREQUAL Release) AND EXISTS "${GIT_REPOSITORY_ROOT}/.git/")
    include(CMakeDateStamp)
    set(OCCPLUGIN_VERSION_DATE "${CMAKE_DATESTAMP_YEAR}${CMAKE_DATESTAMP_MONTH}${CMAKE_DATESTAMP_DAY}")
    if (OCCPLUGIN_VERSION_DATE GREATER 0)
        set(OCCPLUGIN_VERSION ${OCCPLUGIN_VERSION}.${OCCPLUGIN_VERSION_DATE})
    endif ()

    include(CMakeVersionSource)
    if (CMAKE_VERSION_SOURCE)
        set(OCCPLUGIN_VERSION ${OCCPLUGIN_VERSION}-${CMAKE_VERSION_SOURCE})
    endif ()
endif ()
message(STATUS "Plugin version: ${OCCPLUGIN_VERSION}")

###
### Find dependencies
###

find_package(FairRoot REQUIRED)
find_package(Protobuf 3.5.0 REQUIRED)
find_package(GRPC 1.9.1 REQUIRED)
find_package(Boost 1.62 COMPONENTS program_options log signals)

###
### Protobuf + gRPC
###
set(PROTOFILES
    protos/OccPlugin.proto)

# NOTE: we want the files to end up in a subdirectory "protos" in the build dir,
#       but the Protobuf and gRPC generator functions don't support custom output
#       directories.
#       As a workaround, we rewrite CMAKE_CURRENT_BINARY_DIR for Protobuf/gRPC
#       and then restore it to its real CMake-managed value.
file(MAKE_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}/protos)
set(CMAKE_CURRENT_BINARY_DIR_OLD ${CMAKE_CURRENT_BINARY_DIR} )
set(CMAKE_CURRENT_BINARY_DIR ${CMAKE_CURRENT_BINARY_DIR}/protos)

protobuf_generate_cpp(PROTO_SOURCES PROTO_HEADERS ${PROTOFILES})
protobuf_generate_grpc_cpp(GRPC_SOURCES GRPC_HEADERS ${PROTOFILES})

set(CMAKE_CURRENT_BINARY_DIR ${CMAKE_CURRENT_BINARY_DIR_OLD})

###
### Build target OCCPLUGIN
###

set(OCCPLUGIN FairMQPlugin_OCC)

configure_file(${CMAKE_CURRENT_SOURCE_DIR}/OccPluginVersion.h.in
               ${CMAKE_CURRENT_BINARY_DIR}/OccPluginVersion.h)

set(OCCPLUGIN_SOURCES
    OccPlugin.cxx
    OccPluginServer.cxx)

add_library(${OCCPLUGIN} SHARED
    ${OCCPLUGIN_SOURCES}
    ${PROTO_SOURCES}
    ${GRPC_SOURCES})

target_link_libraries(${OCCPLUGIN}
    FairRoot::FairMQ
    grpc::grpc++
    protobuf::libprotobuf
    Boost::program_options)

target_include_directories(${OCCPLUGIN}
    PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_BINARY_DIR}
    ${PROTO_SOURCE_DIR}
    ${FAIRROOT_INCLUDE_DIR}
    ${GRPC_INCLUDE_DIRS})

set_target_properties(${OCCPLUGIN} PROPERTIES
    CXX_VISIBILITY_PRESET hidden
    VERSION ${OCCPLUGIN_VERSION_SHORT}
    SOVERSION ${OCCPLUGIN_VERSION_SHORT})

install(TARGETS ${OCCPLUGIN} DESTINATION lib)