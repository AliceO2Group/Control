/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * In applying this license CERN does not waive the privileges and
 * immunities granted to it by virtue of its status as an
 * Intergovernmental Organization or submit itself to any jurisdiction.
 */

#ifndef OCC_OCCFMQCOMMON_H
#define OCC_OCCFMQCOMMON_H

// We have to force boost::uuids to rely on /dev/*random instead of getrandom(2) or getentropy(3)
// otherwise on some systems we'd get boost::uuids::entropy_error
#define BOOST_UUID_RANDOM_PROVIDER_FORCE_POSIX

#include "plugin/litestructs/Transition.h"

#include <string>
#include <unordered_map>
#include <ctime>

#include <fairmq/PluginServices.h>

#include <boost/uuid/uuid_generators.hpp>
#include <boost/uuid/uuid_io.hpp>

#include "util/Logger.h"

using namespace std::literals;

#ifdef OCC_LITE_SERVICE
#define FMQ_CONTROLLER_NAME "OCClite"
#else
#define FMQ_CONTROLLER_NAME "OCC"
#endif

std::string generateSubscriptionId(const std::string& prefix = "");
bool isIntermediateFMQState(const std::string& state);
std::tuple<OccLite::nopb::TransitionResponse, ::grpc::Status> doTransition(fair::mq::PluginServices* pluginServices, const OccLite::nopb::TransitionRequest& request);

const std::unordered_map<std::string, std::string> EXPECTED_FINAL_STATE = {
    {"INIT DEVICE",  "INITIALIZING DEVICE"},
    {"COMPLETE INIT","INITIALIZED"},
    {"BIND",         "BOUND"},
    {"CONNECT",      "DEVICE READY"},
    {"INIT TASK",    "READY"},
    {"RUN",          "RUNNING"},
    {"STOP",         "READY"},
    {"RESET TASK",   "DEVICE READY"},
    {"RESET DEVICE", "IDLE"},
    {"END",          "EXITING"},
    {"ERROR FOUND",  "ERROR"},
};

#endif //OCC_OCCFMQCOMMON_H
