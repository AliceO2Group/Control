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


#include "OccPluginServer.h"

#include "plugin/litestructs/Transition.h"
#include "util/Defer.h"
#include "util/Logger.h"

#include <fairmq/PluginServices.h>

#include <boost/algorithm/string/join.hpp>
#include <boost/algorithm/string/predicate.hpp>
#include <boost/algorithm/string/split.hpp>

#include <condition_variable>
#include <iomanip>

#include <sys/types.h>
#include <unistd.h>

OccPluginServer::OccPluginServer(fair::mq::PluginServices* pluginServices)
    : Service(), m_pluginServices(pluginServices)
{

}

grpc::Status
OccPluginServer::EventStream(grpc::ServerContext* context,
                             const occ_pb::EventStreamRequest* request,
                             grpc::ServerWriter<occ_pb::EventStreamReply>* writer)
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

        if (state == "EXITING") {
            std::unique_lock<std::mutex> finished_lk(finished_mu);

            auto nilEvent = new occ_pb::DeviceEvent();
            nilEvent->set_type(occ_pb::NULL_DEVICE_EVENT);
            occ_pb::EventStreamReply response;
            response.mutable_event()->CopyFrom(*nilEvent);

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

grpc::Status
OccPluginServer::StateStream(grpc::ServerContext* context,
                             const occ_pb::StateStreamRequest* request,
                             grpc::ServerWriter<occ_pb::StateStreamReply>* writer)
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
        occ_pb::StateType sType = isIntermediateFMQState(state) ? occ_pb::STATE_INTERMEDIATE : occ_pb::STATE_STABLE;

        occ_pb::StateStreamReply response;
        response.set_type(sType);
        response.set_state(state);

        OLOG(debug) << "[StateStream] new state: " << state << "; type: "
                    << occ_pb::StateType_Name(sType);

        if (state != "EXITING") {
            writer->Write(response);
        } else {
            std::unique_lock<std::mutex> finished_lk(finished_mu);
            writer->WriteLast(response, grpc::WriteOptions());
            finished.notify_one();
        }
    };

    auto id = generateSubscriptionId("StateStream");

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
    return grpc::Status::OK;
}

grpc::Status OccPluginServer::GetState(grpc::ServerContext* context,
                                       const occ_pb::GetStateRequest* request,
                                       occ_pb::GetStateReply* response)
{
    std::lock_guard<std::mutex> lock(m_mu);

    (void) context;
    (void) request;

    auto state = fair::mq::PluginServices::ToStr(m_pluginServices->GetCurrentDeviceState());
    pid_t pid = getpid();

    response->set_state(state);
    response->set_pid(pid);

    return grpc::Status::OK;
}

/**
 * Transition requests a state transition from the FairMQ device, and blocks until success or failure.
 *
 * @param context server context
 * @param request the request, as generated and wrapped by Protobuf
 * @param response the response, as generated and wrapped by Protobuf
 * @return the status, either grpc::Status::OK or an error status
 */
grpc::Status
OccPluginServer::Transition(grpc::ServerContext* context,
                            const occ_pb::TransitionRequest* request,
                            occ_pb::TransitionReply* response)
{
    // Valid FairMQ state machine transitions, mapped to DeviceStateTransition objects:
    //    {DeviceStateTransition::Auto,         "Auto"},            // ever needed?
    //    {DeviceStateTransition::InitDevice,   "INIT DEVICE"},
    //    {DeviceStateTransition::CompleteInit, "COMPLETE INIT"},   // automatic?
    //    {DeviceStateTransition::Bind,         "BIND"},
    //    {DeviceStateTransition::Connect,      "CONNECT"},
    //    {DeviceStateTransition::InitTask,     "INIT TASK"},
    //    {DeviceStateTransition::Run,          "RUN"},
    //    {DeviceStateTransition::Stop,         "STOP"},
    //    {DeviceStateTransition::ResetTask,    "RESET TASK"},
    //    {DeviceStateTransition::ResetDevice,  "RESET DEVICE"},
    //    {DeviceStateTransition::End,          "END"},
    //    {DeviceStateTransition::ErrorFound,   "ERROR FOUND"},
    //
    // Valid FairMQ device states, mapped to DeviceState objects:
    //    {DeviceState::Ok,                 "OK"},
    //    {DeviceState::Error,              "ERROR"},
    //    {DeviceState::Idle,               "IDLE"},
    //    {DeviceState::InitializingDevice, "INITIALIZING DEVICE"},
    //    {DeviceState::Initialized,        "INITIALIZED"},
    //    {DeviceState::Binding,            "BINDING"},
    //    {DeviceState::Bound,              "BOUND"},
    //    {DeviceState::Connecting,         "CONNECTING"},
    //    {DeviceState::DeviceReady,        "DEVICE READY"},
    //    {DeviceState::InitializingTask,   "INITIALIZING TASK"},
    //    {DeviceState::Ready,              "READY"},
    //    {DeviceState::Running,            "RUNNING"},
    //    {DeviceState::ResettingTask,      "RESETTING TASK"},
    //    {DeviceState::ResettingDevice,    "RESETTING DEVICE"},
    //    {DeviceState::Exiting,            "EXITING"}


    std::lock_guard<std::mutex> lock(m_mu);

    (void) context;
    if (!request) {
        return grpc::Status(grpc::INVALID_ARGUMENT, "null request received");
    }

    OccLite::nopb::TransitionRequest nopbReq;
    nopbReq.srcState = request->srcstate();
    nopbReq.transitionEvent = request->transitionevent();
    for (const auto &kv : request->arguments()) {
        auto ce = OccLite::nopb::ConfigEntry();
        ce.key = kv.key();
        ce.value = kv.value();
        nopbReq.arguments.push_back(ce);
    }

    auto transitionOutcome = doTransition(m_pluginServices, nopbReq);
    auto grpcStatus = std::get<1>(transitionOutcome);
    if (!grpcStatus.ok()) {
        return grpcStatus;
    }

    auto nopbResponse = std::get<0>(transitionOutcome);
    response->set_state(nopbResponse.state);
    response->set_trigger(static_cast<occ_pb::StateChangeTrigger>(nopbResponse.trigger));
    response->set_transitionevent(nopbResponse.transitionEvent);
    response->set_ok(nopbResponse.ok);

    return grpc::Status::OK;
}
