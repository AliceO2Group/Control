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

#include "OccLiteServer.h"

#include "litestructs/GetState.h"
#include "litestructs/Transition.h"
#include "litestructs/EventStream.h"

#include "util/Defer.h"
#include "util/Logger.h"

#include <fairmq/PluginServices.h>

#include <grpcpp/impl/codegen/method_handler.h>

#include <boost/algorithm/string/join.hpp>
#include <boost/algorithm/string/predicate.hpp>
#include <boost/algorithm/string/split.hpp>

#include <sys/types.h>
#include <unistd.h>

#include <condition_variable>
#include <iomanip>

OccLite::Service::Service(fair::mq::PluginServices* pluginServices)
    : ::grpc::Service(), m_pluginServices(pluginServices)
{
    auto getStateHandler = new ::grpc::internal::RpcMethodHandler<OccLite::Service, OccLite::nopb::GetStateRequest, OccLite::nopb::GetStateResponse>(&OccLite::Service::GetState, this);
    auto getState = new ::grpc::internal::RpcServiceMethod(
        "GetState",
        ::grpc::internal::RpcMethod::NORMAL_RPC,
        getStateHandler);
    AddMethod(getState);

    auto transitionHandler = new ::grpc::internal::RpcMethodHandler<OccLite::Service, OccLite::nopb::TransitionRequest, OccLite::nopb::TransitionResponse>(&OccLite::Service::Transition, this);
    auto transition = new ::grpc::internal::RpcServiceMethod(
        "Transition",
        ::grpc::internal::RpcMethod::NORMAL_RPC,
        transitionHandler);
    AddMethod(transition);

    auto eventStreamHandler = new ::grpc::internal::ServerStreamingHandler<OccLite::Service, OccLite::nopb::EventStreamRequest, OccLite::nopb::EventStreamResponse>(&OccLite::Service::EventStream, this);
    auto eventStream = new ::grpc::internal::RpcServiceMethod(
        "EventStream",
        ::grpc::internal::RpcMethod::SERVER_STREAMING,
        eventStreamHandler);
    AddMethod(eventStream);
}

::grpc::Status OccLite::Service::GetState(::grpc::ServerContext* context,
                                          const OccLite::nopb::GetStateRequest* request,
                                          OccLite::nopb::GetStateResponse* response)
{
    std::lock_guard<std::mutex> lock(m_mu);

    (void) context;
    (void) request;

    OLOG(detail) << "Incoming GetState request: " << request->JsonMessage::Serialize();

    auto state = fair::mq::PluginServices::ToStr(m_pluginServices->GetCurrentDeviceState());
    pid_t pid = getpid();

    response->state = state;
    response->pid = pid;
    OLOG(detail) << "GetState response: " << response->state;

    return grpc::Status::OK;
}

::grpc::Status OccLite::Service::Transition(::grpc::ServerContext* context,
                                            const OccLite::nopb::TransitionRequest* request,
                                            OccLite::nopb::TransitionResponse* response)
{
    OLOG(detail) << "Incoming Transition request: " << request->JsonMessage::Serialize();

    auto transitionOutcome = doTransition(m_pluginServices, *request);
    ::grpc::Status grpcStatus = std::get<1>(transitionOutcome);
    if (!grpcStatus.ok()) {
        OLOG(error) << "Transition failed with error: " << grpcStatus.error_code() << " " << grpcStatus.error_message() << " " << grpcStatus.error_details();
        return grpc::Status::CANCELLED;
    }

    auto nopbResponse = std::get<0>(transitionOutcome);
    *response = nopbResponse;
    OLOG(detail) << "Transition response: " << response->state << " ok: " << response->ok;

    return grpc::Status::OK;
}

::grpc::Status

OccLite::Service::EventStream(::grpc::ServerContext* context, const OccLite::nopb::EventStreamRequest* request,
                              ::grpc::ServerWriter<OccLite::nopb::EventStreamResponse>* writer)
{
    (void) context;
    (void) request;

    std::mutex writer_mu;
    std::condition_variable finished;
    std::mutex finished_mu;
    std::string last_known_state;

    auto onDeviceStateChange = [&](fair::mq::PluginServices::DeviceState reachedState) {
        std::lock_guard<std::mutex> lock(writer_mu);
        auto state = fair::mq::PluginServices::ToStr(reachedState);
        last_known_state = state;

        OLOG(debug) << "[EventStream] new state: " << state;
        // for FairMQ, EXITING and ERROR are both final states and plugins are expected to quit at this point
        // see octrl-888 for more details
        if (state == "EXITING" || state == "ERROR") {
            std::unique_lock<std::mutex> finished_lk(finished_mu);

            auto nilEvent = new nopb::DeviceEvent;
            nilEvent->type = nopb::NULL_DEVICE_EVENT;
            nopb::EventStreamResponse response;
            response.event = *nilEvent;

            writer->WriteLast(response, grpc::WriteOptions());
            delete nilEvent;
            finished.notify_one();
        }
    };

    auto id = generateSubscriptionId("EventStream");

    m_pluginServices->SubscribeToDeviceStateChange(id, onDeviceStateChange);
    DEFER({
        if (last_known_state == "EXITING") {
            m_pluginServices->UnsubscribeFromDeviceStateChange(id);
        }
    });

    {
        std::unique_lock<std::mutex> lk(finished_mu);
        finished.wait(lk);
    }

    return ::grpc::Status::OK;
}


