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

#include "util/Defer.h"
#include "util/Logger.h"

#include <fairmq/PluginServices.h>

#include <boost/algorithm/string/join.hpp>

#include <uuid/uuid.h>

#include <condition_variable>

OccPluginServer::OccPluginServer(fair::mq::PluginServices* pluginServices)
    : Service(), m_pluginServices(pluginServices)
{

}

grpc::Status
OccPluginServer::StateStream(grpc::ServerContext* context,
                             const pb::StateStreamRequest* request,
                             grpc::ServerWriter<pb::StateStreamReply>* writer)
{
    OLOG(DEBUG) << "[request StateStream] handler BEGIN";

    (void) context;
    (void) request;

    std::mutex writer_mu;
    std::condition_variable finished;
    std::mutex finished_mu;

    auto onDeviceStateChange = [&](fair::mq::PluginServices::DeviceState reachedState) {
        OLOG(DEBUG) << "[request StateStream] onDeviceStateChange BEGIN";
        std::lock_guard<std::mutex> lock(writer_mu);
        auto state = fair::mq::PluginServices::ToStr(reachedState);
        pb::StateType sType = isIntermediateState(state) ? pb::STATE_INTERMEDIATE : pb::STATE_STABLE;

        pb::StateStreamReply response;
        response.set_type(sType);
        response.set_state(state);

        OLOG(DEBUG) << "[request StateStream] onDeviceStateChange RESPONSE state: " << state << "; type: "
                    << pb::StateType_Name(sType);

        if (state != "EXITING") {
            writer->Write(response);
        } else {
            std::unique_lock<std::mutex> finished_lk(finished_mu);
            writer->WriteLast(response, grpc::WriteOptions());
            finished.notify_one();
            OLOG(DEBUG) << "[request StateStream] onDeviceStateChange NOTIFY FINISHED";
        }
        OLOG(DEBUG) << "[request StateStream] onDeviceStateChange END";
    };

    uuid_t uuid;
    uuid_generate_time_safe(uuid);
    char uuid_str[37];
    uuid_unparse_lower(uuid, uuid_str);
    std::string id = "OCC_StateStream_"s + std::string(uuid_str);
    OLOG(DEBUG) << "[request StateStream] subscribe, id: " << id;
    m_pluginServices->SubscribeToDeviceStateChange(id, onDeviceStateChange);
    DEFER({
        m_pluginServices->UnsubscribeFromDeviceStateChange(id);
        OLOG(DEBUG) << "[request StateStream] unsubscribe, id: " << id;
    });

    {
        OLOG(DEBUG) << "[request StateStream] blocking until END transition";
        std::unique_lock<std::mutex> lk(finished_mu);
        finished.wait(lk);
        OLOG(DEBUG) << "[request StateStream] transitioned to END, closing stream";
    }
    OLOG(DEBUG) << "[request StateStream] handler END";
    return grpc::Status::OK;
}

grpc::Status OccPluginServer::GetState(grpc::ServerContext* context,
                                       const pb::GetStateRequest* request,
                                       pb::GetStateReply* response)
{
    std::lock_guard<std::mutex> lock(m_mu);
    OLOG(DEBUG) << "[request GetState] handler BEGIN";

    (void) context;
    (void) request;

    auto state = fair::mq::PluginServices::ToStr(m_pluginServices->GetCurrentDeviceState());
    response->set_state(state);

    OLOG(DEBUG) << "[request GetState] handler END";
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
                            const pb::TransitionRequest* request,
                            pb::TransitionReply* response)
{
    // Valid FairMQ state machine transitions, mapped to DeviceStateTransition objects:
    //    {DeviceStateTransition::InitDevice,  "INIT DEVICE"},
    //    {DeviceStateTransition::InitTask,    "INIT TASK"},
    //    {DeviceStateTransition::Run,         "RUN"},
    //    {DeviceStateTransition::Pause,       "PAUSE"},
    //    {DeviceStateTransition::Resume,      "RESUME"},
    //    {DeviceStateTransition::Stop,        "STOP"},
    //    {DeviceStateTransition::ResetTask,   "RESET TASK"},
    //    {DeviceStateTransition::ResetDevice, "RESET DEVICE"},
    //    {DeviceStateTransition::End,         "END"},
    //    {DeviceStateTransition::ErrorFound,  "ERROR FOUND"},
    //
    // Valid FairMQ device states, mapped to DeviceState objects:
    //    {DeviceState::Ok,                 "OK"},
    //    {DeviceState::Error,              "ERROR"},
    //    {DeviceState::Idle,               "IDLE"},
    //    {DeviceState::InitializingDevice, "INITIALIZING DEVICE"},
    //    {DeviceState::DeviceReady,        "DEVICE READY"},
    //    {DeviceState::InitializingTask,   "INITIALIZING TASK"},
    //    {DeviceState::Ready,              "READY"},
    //    {DeviceState::Running,            "RUNNING"},
    //    {DeviceState::Paused,             "PAUSED"},
    //    {DeviceState::ResettingTask,      "RESETTING TASK"},
    //    {DeviceState::ResettingDevice,    "RESETTING DEVICE"},
    //    {DeviceState::Exiting,            "EXITING"}


    std::lock_guard<std::mutex> lock(m_mu);
    OLOG(DEBUG) << "[request Transition] handler BEGIN";

    (void) context;
    if (!request) {
        return grpc::Status(grpc::INVALID_ARGUMENT, "null request received");
    }

    std::string srcState = request->srcstate();
    std::string event = request->event();
    auto arguments = request->arguments();

    std::string currentState = fair::mq::PluginServices::ToStr(m_pluginServices->GetCurrentDeviceState());
    if (srcState != currentState) {
        return grpc::Status(grpc::INVALID_ARGUMENT,
                            "transition not possible: state mismatch: source: " + srcState + " current: " +
                            currentState);
    }

    OLOG(DEBUG) << "[request Transition] src: " << srcState
                << " currentState: " << currentState
                << " event: " << event;

    std::vector<std::string> newStates;
    const std::string finalState = EXPECTED_FINAL_STATE.at(event);

    OLOG(DEBUG) << "[request Transition] finalState: " << finalState;

    std::condition_variable cv;
    std::mutex cv_mu;

    auto onDeviceStateChange = [&](fair::mq::PluginServices::DeviceState reachedState) {
        OLOG(DEBUG) << "[request Transition] onDeviceStateChange BEGIN";

        if (reachedState == fair::mq::PluginServices::DeviceState::InitializingDevice) {
            for (auto it = arguments.cbegin(); it != arguments.cend(); ++it) {
                m_pluginServices->SetProperty(it->key(), it->value());
            }
        }

        std::unique_lock<std::mutex> lk(cv_mu);
        newStates.push_back(fair::mq::PluginServices::ToStr(reachedState));
        OLOG(DEBUG) << "[request Transition] newStates vector: " << boost::algorithm::join(newStates, ", ");
        cv.notify_one();
        OLOG(DEBUG) << "[request Transition] onDeviceStateChange END";
    };

    uuid_t uuid;
    uuid_generate_time_safe(uuid);
    char uuid_str[37];
    uuid_unparse_lower(uuid, uuid_str);
    std::string id = "OCC_Transition_"s + std::string(uuid_str);
    OLOG(DEBUG) << "[request Transition] subscribe, id: " << id;
    m_pluginServices->SubscribeToDeviceStateChange(id, onDeviceStateChange);
    DEFER({
        m_pluginServices->UnsubscribeFromDeviceStateChange(id);
        OLOG(DEBUG) << "[request Transition] unsubscribe, id: " << id;
    });

    try {
        m_pluginServices->ChangeDeviceState("OCC", fair::mq::PluginServices::ToDeviceStateTransition(event));
    }
    catch (fair::mq::PluginServices::DeviceControlError& e) {
        OLOG(ERROR) << "[request Transition] cannot request transition: " << e.what();
        return grpc::Status(grpc::INTERNAL, "cannot request transition, OCC plugin has no device control");
    }
    catch (std::out_of_range& e) {
        OLOG(ERROR) << "[request Transition] invalid transition name: " << request->event();
        return grpc::Status(grpc::INVALID_ARGUMENT, "argument " + request->event() + " is not a valid transition name");
    }

    {
        std::unique_lock<std::mutex> lk(cv_mu);
        OLOG(DEBUG) << "[request Transition] states locked, last known state: "
                    << (newStates.size() ? newStates.back() : "NIL");

        // IF we have no states in list yet, OR
        //    we have some states, and the last one is an intermediate state (for which an autotransition is presumably about to happen)
        if (newStates.empty() ||
            !newStates.empty() && isIntermediateState(newStates.back())) {
            // We need to block until the transitions are complete
            for (;;) {
                OLOG(DEBUG) << "[request Transition] transitions expected, blocking";
                cv.wait(lk);
                if (newStates.empty()) {
                    OLOG(ERROR) << "[request Transition] notify condition met but no states written";
                    break;
                }

                OLOG(DEBUG) << "[request Transition] notify condition met, reached state: " << newStates.back();
                if (isIntermediateState(newStates.back())) { //if it's an auto state
                    continue;
                } else {
                    break;
                }
            }
        }
    }

    if (newStates.empty()) {
        return grpc::Status(grpc::INTERNAL,
                            "no transitions made, current state stays " + srcState);
    }

    response->set_state(newStates.back());
    response->set_event(request->event());
    response->set_ok(newStates.back() == finalState);
    if (newStates.back() == "ERROR") {              // ERROR state
        response->set_trigger(pb::DEVICE_ERROR);
    } else if (newStates.back() == finalState) {    // correct destination state
        response->set_trigger(pb::EXECUTOR);
    } else {                                        // some other state, for whatever reason - we assume DEVICE_INTENTIONAL
        response->set_trigger(pb::DEVICE_INTENTIONAL);
    }

    OLOG(DEBUG) << "[request Transition] handler END, states visited: " << boost::algorithm::join(newStates, ", ");
    return grpc::Status::OK;
}

bool OccPluginServer::isIntermediateState(const std::string& state)
{
    return state.find("INITIALIZING") != std::string::npos ||
           state.find("RESETTING") != std::string::npos;
}
