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

#include "OccFMQCommon.h"

#include "protos/occ.pb.h"
#include "protos/occ.grpc.pb.h"

#include <mutex>

namespace fair
{
namespace mq
{
class PluginServices;
}
}

class OccPluginServer final : public occ_pb::Occ::Service
{
public:
    explicit OccPluginServer(fair::mq::PluginServices*);

    virtual ~OccPluginServer()
    {}

    grpc::Status EventStream(grpc::ServerContext* context,
                             const occ_pb::EventStreamRequest* request,
                             grpc::ServerWriter<occ_pb::EventStreamReply>* writer) override;

    grpc::Status StateStream(grpc::ServerContext* context,
                             const occ_pb::StateStreamRequest* request,
                             grpc::ServerWriter<occ_pb::StateStreamReply>* writer) override;

    grpc::Status GetState(grpc::ServerContext* context,
                          const occ_pb::GetStateRequest* request,
                          occ_pb::GetStateReply* response) override;

    grpc::Status Transition(grpc::ServerContext* context,
                            const occ_pb::TransitionRequest* request,
                            occ_pb::TransitionReply* response) override;

private:
    fair::mq::PluginServices* m_pluginServices;
    std::mutex m_mu;
};


#endif //OCCPLUGIN_OCCPLUGINSERVER_H
