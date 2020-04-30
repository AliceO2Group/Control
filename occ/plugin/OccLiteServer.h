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

#ifndef OCC_OCCLITESERVER_H
#define OCC_OCCLITESERVER_H

#include "OccFMQCommon.h"

#include <grpcpp/impl/codegen/service_type.h>
#include <grpcpp/impl/codegen/sync_stream.h>
#include <grpcpp/server_impl.h>

#include <mutex>

namespace fair
{
namespace mq
{
class PluginServices;
}
}

namespace OccLite {
namespace nopb
{
class GetStateRequest;
class GetStateResponse;
class TransitionRequest;
class TransitionResponse;
class EventStreamRequest;
class EventStreamResponse;
}

class Service : public ::grpc::Service
{
public:
    explicit Service(fair::mq::PluginServices*);

    virtual ~Service()
    {}

    ::grpc::Status EventStream(::grpc_impl::ServerContext* context,
                               const nopb::EventStreamRequest* request,
                               ::grpc::ServerWriter<nopb::EventStreamResponse>* writer);

    ::grpc::Status GetState(::grpc_impl::ServerContext*,
                            const nopb::GetStateRequest*,
                            nopb::GetStateResponse*);

    ::grpc::Status Transition(::grpc_impl::ServerContext* context,
                              const nopb::TransitionRequest* request,
                              nopb::TransitionResponse* response);

private:
    fair::mq::PluginServices* m_pluginServices;
    std::mutex m_mu;
};
} // ns OccLite

#endif //OCC_OCCLITESERVER_H
