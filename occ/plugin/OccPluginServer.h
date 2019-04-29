/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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

#ifndef OCCPLUGIN_OCCPLUGINSERVER_H
#define OCCPLUGIN_OCCPLUGINSERVER_H

#include "protos/occ.pb.h"
#include "protos/occ.grpc.pb.h"

#include <mutex>

// We have to force boost::uuids to rely on /dev/*random instead of getrandom(2) or getentropy(3)
// otherwise on some systems we'd get boost::uuids::entropy_error
#define BOOST_UUID_RANDOM_PROVIDER_FORCE_POSIX

namespace pb = occ_pb;

namespace fair
{
namespace mq
{
class PluginServices;
}
}

using namespace std::literals;

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

class OccPluginServer final : public pb::Occ::Service
{
public:
    explicit OccPluginServer(fair::mq::PluginServices*);

    virtual ~OccPluginServer()
    {}

    grpc::Status EventStream(grpc::ServerContext* context,
                             const pb::EventStreamRequest* request,
                             grpc::ServerWriter<pb::EventStreamReply>* writer) override;

    grpc::Status StateStream(grpc::ServerContext* context,
                             const pb::StateStreamRequest* request,
                             grpc::ServerWriter<pb::StateStreamReply>* writer) override;

    grpc::Status GetState(grpc::ServerContext* context,
                          const pb::GetStateRequest* request,
                          pb::GetStateReply* response) override;

    grpc::Status Transition(grpc::ServerContext* context,
                            const pb::TransitionRequest* request,
                            pb::TransitionReply* response) override;

private:
    bool isIntermediateState(const std::string& state);
    std::string generateSubscriptionId(const std::string& prefix = "");

    fair::mq::PluginServices* m_pluginServices;
    std::mutex m_mu;
};


#endif //OCCPLUGIN_OCCPLUGINSERVER_H
