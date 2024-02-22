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
cmake_policy(SET CMP0074 NEW)
cmake_policy(SET CMP0077 NEW)

### HACK
# lib is lib64 on CC7, but we want lib to be lib.
# It is probably bad to set this here: it should be system-provided
# or set in the aliBuild recipe.
set(CMAKE_INSTALL_LIBDIR lib)


###
### Product information
###

set(KAKI_ORGANIZATION_NAME "O²")
set(KAKI_ORGANIZATION_DOMAIN "https://github.com/AliceO2Group/Control")

set(KAKI_PRODUCT_NAME "O² kaki")
set(KAKI_PRODUCT_MAINTAINER "Michal Tichak <michal.tichak@cern.ch>")
set(KAKI_DESCRIPTION_SUMMARY "Binary used for reading kafka topics and translating them to InfoLogger")

# Read version number from VERSION file in parent dir
set(KAKI_VERSION_MAJOR 0)
set(KAKI_VERSION_MINOR 0)
set(KAKI_VERSION_PATCH 0)

set(VERSION_FILE "${CMAKE_SOURCE_DIR}/../VERSION")
file(READ "${VERSION_FILE}" VERSION_DATA)

string(REGEX MATCH "VERSION_MAJOR := ([0-9]*)" _ ${VERSION_DATA})
set(KAKI_VERSION_MAJOR ${CMAKE_MATCH_1})
string(REGEX MATCH "VERSION_MINOR := ([0-9]*)" _ ${VERSION_DATA})
set(KAKI_VERSION_MINOR ${CMAKE_MATCH_1})
string(REGEX MATCH "VERSION_PATCH := ([0-9]*)" _ ${VERSION_DATA})
set(KAKI_VERSION_PATCH ${CMAKE_MATCH_1})

set(KAKI_VERSION ${KAKI_VERSION_MAJOR}.${KAKI_VERSION_MINOR}.${KAKI_VERSION_PATCH})
set(KAKI_VERSION_SHORT ${KAKI_VERSION})

set(CMAKE_MODULE_PATH
    ${CMAKE_MODULE_PATH}
    ${CMAKE_SOURCE_DIR}/cmake
    ${CMAKE_SOURCE_DIR}/cmake/modules)
include(FeatureSummary)
include(GenerateExportHeader)

message(STATUS "Build type: ${CMAKE_BUILD_TYPE}")
set(GIT_REPOSITORY_ROOT "${CMAKE_SOURCE_DIR}/..")
if (NOT (CMAKE_BUILD_TYPE STREQUAL Release) AND EXISTS "${GIT_REPOSITORY_ROOT}/.git/")
    include(CMakeDateStamp)
    set(KAKI_VERSION_DATE "${CMAKE_DATESTAMP_YEAR}${CMAKE_DATESTAMP_MONTH}${CMAKE_DATESTAMP_DAY}")
    if (KAKI_VERSION_DATE GREATER 0)
        set(KAKI_VERSION ${KAKI_VERSION}.${KAKI_VERSION_DATE})
    endif ()

    include(CMakeVersionSource)
    if (CMAKE_VERSION_SOURCE)
        set(KAKI_VERSION ${KAKI_VERSION}-${CMAKE_VERSION_SOURCE})
    endif ()
endif ()
message(STATUS "KAKI version: ${KAKI_VERSION}")


project(kaki
  VERSION ${KAKIR_VERSION_SHORT}
  DESCRIPTION "${KAKI_DESCRIPTION_SUMMARY}"
    LANGUAGES CXX)

if (CMAKE_COMPILER_IS_GNUCXX)
    if (CMAKE_CXX_COMPILER_VERSION VERSION_GREATER 4.9 OR
        CMAKE_CXX_COMPILER_VERSION VERSION_EQUAL 4.9)
        message(STATUS "Found GNU g++ ${CMAKE_CXX_COMPILER_VERSION}, enabling colorized error messages.")
        set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fdiagnostics-color=auto")
    endif ()
endif ()

# Set the default build type to "RelWithDebInfo"
if(NOT CMAKE_BUILD_TYPE AND NOT CMAKE_CONFIGURATION_TYPES)
  set(CMAKE_BUILD_TYPE "RelWithDebInfo"
    CACHE
    STRING "Choose the type of build, options are: Debug Release RelWithDebInfo MinSizeRel Coverage."
    FORCE
  )
endif()

add_subdirectory(src)
add_subdirectory(external)
add_subdirectory(test)
