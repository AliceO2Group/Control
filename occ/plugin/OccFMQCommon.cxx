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

#include "OccFMQCommon.h"

#include "util/Defer.h"

#include <boost/algorithm/string.hpp>
#include <iomanip>

std::string generateSubscriptionId(const std::string& prefix)
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

bool isIntermediateFMQState(const std::string& state)
{
    return state.find("INITIALIZING TASK") != std::string::npos ||
           state.find("RESETTING") != std::string::npos ||
           state.find("BINDING") != std::string::npos ||
           state.find("CONNECTING") != std::string::npos;
}

std::tuple<OccLite::nopb::TransitionResponse, ::grpc::Status> doTransition(fair::mq::PluginServices* m_pluginServices, const OccLite::nopb::TransitionRequest& request)
{
    std::string srcState = request.srcState;
    std::string event = request.transitionEvent;
    auto arguments = request.arguments;

    std::string currentState = fair::mq::PluginServices::ToStr(m_pluginServices->GetCurrentDeviceState());
    if (srcState != currentState) {
        return std::make_tuple(OccLite::nopb::TransitionResponse(), grpc::Status(grpc::INVALID_ARGUMENT,
                            "transition not possible: state mismatch: source: " + srcState + " current: " +
                            currentState));
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
                std::string key = it->key;
                std::string value = it->value;
                if (boost::starts_with(key, "chans.")) {
                    key.erase(0, 6);
                    std::vector<std::string> split;
                    boost::split(split, key, std::bind1st(std::equal_to<char>(), '.'));
                    if (std::find(intKeys.begin(), intKeys.end(), split.back()) != intKeys.end()) {
                        auto intValue = std::stoi(value);
                        m_pluginServices->SetProperty(key, intValue);
                    }
                    else {
                        m_pluginServices->SetProperty(key, value);
                    }
                }
                else {
                    m_pluginServices->SetProperty(key, value);
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
                std::string key = it->key;
                std::string value = it->value;
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
                    channels[name][propKey] = value;
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
                for (auto const& entry : arguments) {
                    m_pluginServices->SetProperty(entry.key, entry.value);
                }
            }
            catch (std::runtime_error &e) {
                OLOG(WARNING) << "transition cannot push RUN transition arguments, reason:" << e.what();
            }
        }
        m_pluginServices->ChangeDeviceState(FMQ_CONTROLLER_NAME, evt);
    }
    catch (fair::mq::PluginServices::DeviceControlError& e) {
        OLOG(ERROR) << "transition cannot request transition: " << e.what();
        return std::make_tuple(OccLite::nopb::TransitionResponse(), grpc::Status(grpc::INTERNAL, "cannot request transition, OCC plugin has no device control"));
    }
    catch (std::out_of_range& e) {
        OLOG(ERROR) << "transition invalid event name: " << event;
        return std::make_tuple(OccLite::nopb::TransitionResponse(), grpc::Status(grpc::INVALID_ARGUMENT, "argument " + event + " is not a valid transition name"));
    }

    {
        std::unique_lock<std::mutex> lk(cv_mu);

        // IF we have no states in list yet, OR
        //    we have some states, and the last one is an intermediate state (for which an autotransition is presumably about to happen)
        if (newStates.empty() || isIntermediateFMQState(newStates.back())) {
            // We need to block until the transitions are complete
            for (;;) {
                cv.wait(lk);
                if (newStates.empty()) {
                    OLOG(ERROR) << "[request Transition] notify condition met but no states written";
                    break;
                }

                OLOG(DEBUG) << "transition notify condition met, reached state: " << newStates.back();
                if (isIntermediateFMQState(newStates.back())) { //if it's an auto state
                    continue;
                } else {
                    break;
                }
            }
        }
    }

    if (newStates.empty()) {
        return std::make_tuple(OccLite::nopb::TransitionResponse(), grpc::Status(grpc::INTERNAL,
                            "no transitions made, current state stays " + srcState));
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

    auto response = OccLite::nopb::TransitionResponse();
    response.state = newStates.back();
    response.transitionEvent = event;
    response.ok = (newStates.back() == finalState);
    if (newStates.back() == "ERROR") {              // ERROR state
        response.trigger = OccLite::nopb::DEVICE_ERROR;
    } else if (newStates.back() == finalState) {    // correct destination state
        response.trigger = OccLite::nopb::EXECUTOR;
    } else {                                        // some other state, for whatever reason - we assume DEVICE_INTENTIONAL
        response.trigger = OccLite::nopb::DEVICE_INTENTIONAL;
    }

    OLOG(DEBUG) << "transition done, states visited: " << boost::algorithm::join(newStates, ", ");
    return std::make_tuple(response, grpc::Status::OK);
}