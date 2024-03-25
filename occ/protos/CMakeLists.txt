# === This file is part of ALICE O² ===
#
# Copyright 2024 CERN and copyright holders of ALICE O².
# Author: Michal Tichak <michal.tichak@cern.ch>
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

set(PROTO_GENERATED_DIR "${CMAKE_CURRENT_BINARY_DIR}/protos")
set(PROTO_IMPORT_DIR "${CMAKE_CURRENT_LIST_DIR}")

add_library(Occ-proto OBJECT "${CMAKE_CURRENT_LIST_DIR}/occ.proto")
target_link_libraries(Occ-proto PUBLIC 
  protobuf::libprotobuf 
  gRPC::grpc++
  gRPC::grpc++_reflection
)
target_include_directories(Occ-proto PUBLIC "$<BUILD_INTERFACE:${CMAKE_CURRENT_BINARY_DIR}>")

protobuf_generate(
  TARGET Occ-proto 
  IMPORT_DIRS ${PROTO_IMPORT_DIR}
  PROTOC_OUT_DIR "${PROTO_GENERATED_DIR}")

protobuf_generate(
  TARGET Occ-proto
  LANGUAGE grpc
  GENERATE_EXTENSIONS .grpc.pb.h .grpc.pb.cc
  PLUGIN "protoc-gen-grpc=\$<TARGET_FILE:gRPC::grpc_cpp_plugin>"
  IMPORT_DIRS ${PROTO_IMPORT_DIRS}
  PROTOC_OUT_DIR "${PROTO_GENERATED_DIR}")

install(TARGETS Occ-proto EXPORT OccTargets)
