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
#include <boost/uuid/uuid_generators.hpp>
#include <boost/uuid/uuid_io.hpp>
#include <boost/algorithm/string/predicate.hpp>
#include <boost/algorithm/string/join.hpp>
#include <boost/algorithm/string/split.hpp>

#include <condition_variable>
#include <iomanip>

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

    auto onDeviceStateChange = [&](fair::mq::PluginServices::DeviceState reachedState) {
        std::lock_guard<std::mutex> lock(writer_mu);
        auto state = fair::mq::PluginServices::ToStr(reachedState);

        OLOG(DEBUG) << "[EventStream] new state: " << state;

        if (state == "EXITING") {
            std::unique_lock<std::mutex> finished_lk(finished_mu);

            auto nilEvent = new pb::DeviceEvent();
            nilEvent->set_type(pb::NULL_DEVICE_EVENT);
            pb::EventStreamReply response;
            response.mutable_event()->CopyFrom(*nilEvent);

            writer->WriteLast(response, grpc::WriteOptions());
            delete nilEvent;
            finished.notify_one();
        }
    };

    auto id = generateSubscriptionId("EventStream");

    m_pluginServices->SubscribeToDeviceStateChange(id, onDeviceStateChange);
    DEFER({
        m_pluginServices->UnsubscribeFromDeviceStateChange(id);
    });

    {
        std::unique_lock<std::mutex> lk(finished_mu);
        finished.wait(lk);
    }

    return ::grpc::Status::OK;

}

grpc::Status
OccPluginServer::StateStream(grpc::ServerContext* context,
                             const pb::StateStreamRequest* request,
                             grpc::ServerWriter<pb::StateStreamReply>* writer)
{

    (void) context;
    (void) request;

    std::mutex writer_mu;
    std::condition_variable finished;
    std::mutex finished_mu;

    auto onDeviceStateChange = [&](fair::mq::PluginServices::DeviceState reachedState) {
        std::lock_guard<std::mutex> lock(writer_mu);
        auto state = fair::mq::PluginServices::ToStr(reachedState);
        pb::StateType sType = isIntermediateState(state) ? pb::STATE_INTERMEDIATE : pb::STATE_STABLE;

        pb::StateStreamReply response;
        response.set_type(sType);
        response.set_state(state);

        OLOG(DEBUG) << "[StateStream] new state: " << state << "; type: "
                    << pb::StateType_Name(sType);

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
        m_pluginServices->UnsubscribeFromDeviceStateChange(id);
    });

    {
        std::unique_lock<std::mutex> lk(finished_mu);
        finished.wait(lk);
    }
    return grpc::Status::OK;
}

grpc::Status OccPluginServer::GetState(grpc::ServerContext* context,
                                       const pb::GetStateRequest* request,
                                       pb::GetStateReply* response)
{
    std::lock_guard<std::mutex> lock(m_mu);

    (void) context;
    (void) request;

    auto state = fair::mq::PluginServices::ToStr(m_pluginServices->GetCurrentDeviceState());
    response->set_state(state);

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

    std::string srcState = request->srcstate();
    std::string event = request->transitionevent();
    auto arguments = request->arguments();

    std::string currentState = fair::mq::PluginServices::ToStr(m_pluginServices->GetCurrentDeviceState());
    if (srcState != currentState) {
        return grpc::Status(grpc::INVALID_ARGUMENT,
                            "transition not possible: state mismatch: source: " + srcState + " current: " +
                            currentState);
    }

    OLOG(DEBUG) << "transition src: " << srcState
                << " currentState: " << currentState
                << " event: " << event;

    std::vector<std::string> newStates;
    const std::string finalState = EXPECTED_FINAL_STATE.at(event);

    std::condition_variable cv;
    std::mutex cv_mu;

    auto onDeviceStateChange = [&](fair::mq::PluginServices::DeviceState reachedState) {
        // CONFIGURE arguments must be pushed during InitializingDevice
        if (reachedState == fair::mq::PluginServices::DeviceState::InitializingDevice) {

            // FIXME: workaround which special cases a stoi for certain properties
            // which must be pushed as int.
            // This should be removed once SetPropertyAsString becomes available.
            const std::vector<std::string> intKeys = {
                "rateLogging",
                "rcvBufSize",
                "sndBufSize",
                "linger",
                "rcvKernelSize",
                "sndKernelSize"
            };
            for (auto it = arguments.cbegin(); it != arguments.cend(); ++it) {
                auto key = it->key();
                if (boost::starts_with(key, "chans.")) {
                    key.erase(0, 6);
                    std::vector<std::string> split;
                    boost::split(split, key, std::bind1st(std::equal_to<char>(), '.'));
                    if (std::find(intKeys.begin(), intKeys.end(), split.back()) != intKeys.end()) {
                        auto intValue = std::stoi(it->value());
                        m_pluginServices->SetProperty(it->key(), intValue);
                    }
                    else {
                        m_pluginServices->SetProperty(it->key(), it->value());
                    }
                }
                else {
                    m_pluginServices->SetProperty(it->key(), it->value());
                }
            }
        }

        std::unique_lock<std::mutex> lk(cv_mu);
        newStates.push_back(fair::mq::PluginServices::ToStr(reachedState));
        OLOG(DEBUG) << "transition newStates vector: " << boost::algorithm::join(newStates, ", ");
        cv.notify_one();
    };

    auto id = generateSubscriptionId("Transition");

    m_pluginServices->SubscribeToDeviceStateChange(id, onDeviceStateChange);
    DEFER({
        m_pluginServices->UnsubscribeFromDeviceStateChange(id);
    });

    try {
        auto evt = fair::mq::PluginServices::ToDeviceStateTransition(event);

        // FIXME: big ugly workaround over here
        // Since FairMQ currently (11/2018) can't yet implicitly create channels when receiving
        // chans.* properties during INITIALIZING DEVICE, we must fake a --channel-config cli
        // parameter during INIT and before the INIT DEVICE event.
        // We extract channel related properties from the OCC transition arguments vector and we
        // build up a vector of strings which mimics stuff along the lines of
        //    --channel-config name=data,type=push,method=bind,address=tcp://*:5555,rateLogging=0"
        // See https://github.com/FairRootGroup/FairMQ/pull/111
        // When the relevant FairMQ 1.4.x version implements implicit channel creation, this whole
        // block should be removed with no loss of functionality.
        if (evt == fair::mq::PluginServices::DeviceStateTransition::InitDevice) {
            std::unordered_map<std::string, std::unordered_map<std::string, std::string>> channels;
            for (auto it = arguments.cbegin(); it != arguments.cend(); ++it) {
                auto key = it->key();
                if (boost::starts_with(key, "chans.")) {
                    key.erase(0, 6);
                    std::vector<std::string> split;
                    boost::split(split, key, std::bind1st(std::equal_to<char>(), '.'));
                    if (split.size() != 3)
                        continue;
                    auto name = split[0];
                    auto propKey = split[2];
                    if (channels.find(name) == channels.end()) // if map for this chan doesn't exist yet
                        channels[name] = std::unordered_map<std::string, std::string>();
                    channels[name][propKey] = it->value();
                }
            }

            std::vector<std::string> channelLines;
            for (auto it = channels.cbegin(); it != channels.cend(); ++it) {
                std::vector<std::string> line;
                line.push_back("name=" + it->first);
                for (auto jt = it->second.cbegin(); jt != it->second.cend(); ++jt) {
                    line.push_back(jt->first + "=" + jt->second);
                }
                channelLines.push_back(boost::join(line, ","));
                OLOG(DEBUG) << "transition pushing channel configuration " << channelLines.back();
            }
            if (!channelLines.empty()) {
                m_pluginServices->SetProperty("channel-config", channelLines);
            }
        }
        // Run number must be pushed immediately before RUN transition
        else if (evt == fair::mq::PluginServices::DeviceStateTransition::Run) {
            try {
                for (auto it = arguments.cbegin(); it != arguments.cend(); ++it) {
                    m_pluginServices->SetProperty(it->key(), it->value());
                }
            }
            catch (std::runtime_error &e) {
                OLOG(WARNING) << "transition cannot push RUN transition arguments, reason:" << e.what();
            }
        }
        m_pluginServices->ChangeDeviceState("OCC", evt);
    }
    catch (fair::mq::PluginServices::DeviceControlError& e) {
        OLOG(ERROR) << "transition cannot request transition: " << e.what();
        return grpc::Status(grpc::INTERNAL, "cannot request transition, OCC plugin has no device control");
    }
    catch (std::out_of_range& e) {
        OLOG(ERROR) << "transition invalid event name: " << request->transitionevent();
        return grpc::Status(grpc::INVALID_ARGUMENT, "argument " + request->transitionevent() + " is not a valid transition name");
    }

    {
        std::unique_lock<std::mutex> lk(cv_mu);

        // IF we have no states in list yet, OR
        //    we have some states, and the last one is an intermediate state (for which an autotransition is presumably about to happen)
        if (newStates.empty() || isIntermediateState(newStates.back())) {
            // We need to block until the transitions are complete
            for (;;) {
                cv.wait(lk);
                if (newStates.empty()) {
                    OLOG(ERROR) << "[request Transition] notify condition met but no states written";
                    break;
                }

                OLOG(DEBUG) << "transition notify condition met, reached state: " << newStates.back();
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

    if (srcState == "IDLE" && newStates.back() == "DEVICE READY") {
        // Debug: list of FairMQ property keys
        auto pk = m_pluginServices->GetPropertyKeys();
        for (const auto &k : pk) {
            OLOG(DEBUG) << std::setw(30) << k << " = " + m_pluginServices->GetPropertyAsString(k);
        }
        auto chi = m_pluginServices->GetChannelInfo();
        OLOG(DEBUG) << "channel info:";
        for (const auto &k : chi) {
            OLOG(DEBUG) << k.first << " : " << k.second;
        }
    }

    response->set_state(newStates.back());
    response->set_transitionevent(request->transitionevent());
    response->set_ok(newStates.back() == finalState);
    if (newStates.back() == "ERROR") {              // ERROR state
        response->set_trigger(pb::DEVICE_ERROR);
    } else if (newStates.back() == finalState) {    // correct destination state
        response->set_trigger(pb::EXECUTOR);
    } else {                                        // some other state, for whatever reason - we assume DEVICE_INTENTIONAL
        response->set_trigger(pb::DEVICE_INTENTIONAL);
    }

    OLOG(DEBUG) << "transition done, states visited: " << boost::algorithm::join(newStates, ", ");
    return grpc::Status::OK;
}

bool OccPluginServer::isIntermediateState(const std::string& state)
{
    return state.find("INITIALIZING TASK") != std::string::npos ||
           state.find("RESETTING") != std::string::npos ||
           state.find("BINDING") != std::string::npos ||
           state.find("CONNECTING") != std::string::npos;
}

std::string OccPluginServer::generateSubscriptionId(const std::string& prefix)
{
    std::string id;
    try {
        boost::uuids::random_generator gen;
        id = boost::uuids::to_string(gen());
    } catch(const boost::uuids::entropy_error &err) {
        OLOG(WARNING) << "[generateSubscriptionId] boost::uuids::entropy_error: " << err.what() << "  falling back to std::time";
        id = std::to_string(std::time(nullptr));
    }
    return "OCC_"s + (prefix.size() ? (prefix + "_") : "") + id;
}
