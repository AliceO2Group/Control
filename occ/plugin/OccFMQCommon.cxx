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

#include "util/Common.h"
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
        OLOG(warning) << "[generateSubscriptionId] boost::uuids::entropy_error: " << err.what() << "  falling back to std::time";
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

    OLOG(debug) << "transition src: " << srcState
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
                    std::vector<std::string> split;
                    boost::split(split, key, std::bind(std::equal_to<>(), '.', std::placeholders::_1));
                    if (std::find(intKeys.begin(), intKeys.end(), split.back()) != intKeys.end()) {
                        auto intValue = std::stoi(value);
                        m_pluginServices->SetProperty(key, intValue);
                        OLOG(debug) << "SetProperty(chan int) called " << key << ":" << intValue;
                    }
                    else {
                        m_pluginServices->SetProperty(key, value);
                        OLOG(debug) << "SetProperty(chan string) called " << key << ":" << value;
                    }
                }
                else if (boost::starts_with(key, "__ptree__:")) {
                    // we need to ptreefy whatever payload we got under this kind of key, on a best-effort basis
                    auto [newKey, newValue] = propMapEntryToPtree(key, value);
                    if (newKey == key) { // Means something went wrong and the called function already printed out the message
                        continue;
                    }

                    m_pluginServices->SetProperty(newKey, newValue);
                    OLOG(debug) << "SetProperty(ptree) called " << newKey << ":" << value;
                }
                else { // default case, 1 k-v ==> 1 SetProperty
                    m_pluginServices->SetProperty(key, value);
                    OLOG(debug) << "SetProperty(string) called " << key << ":" << value;
                }
            }
        }

        std::unique_lock<std::mutex> lk(cv_mu);
        newStates.push_back(fair::mq::PluginServices::ToStr(reachedState));
        OLOG(debug) << "transition newStates vector: " << boost::algorithm::join(newStates, ", ");
        cv.notify_one();
    };

    auto id = generateSubscriptionId("Transition");

    m_pluginServices->SubscribeToDeviceStateChange(id, onDeviceStateChange);
    DEFER({
        m_pluginServices->UnsubscribeFromDeviceStateChange(id);
    });

    try {
        auto evt = fair::mq::PluginServices::ToDeviceStateTransition(event);

        // Run number must be pushed immediately before RUN transition
        if (evt == fair::mq::PluginServices::DeviceStateTransition::Run) {
            try {
                for (auto const& entry : arguments) {
                    m_pluginServices->SetProperty(entry.key, entry.value);
                }
            }
            catch (std::runtime_error &e) {
                OLOG(warning) << "transition cannot push RUN transition arguments, reason:" << e.what();
            }
        }
        m_pluginServices->ChangeDeviceState(FMQ_CONTROLLER_NAME, evt);
    }
    catch (fair::mq::PluginServices::DeviceControlError& e) {
        OLOG(error) << "transition cannot request transition: " << e.what();
        return std::make_tuple(OccLite::nopb::TransitionResponse(), grpc::Status(grpc::INTERNAL, "cannot request transition, OCC plugin has no device control"));
    }
    catch (std::out_of_range& e) {
        OLOG(error) << "transition invalid event name: " << event;
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
                    OLOG(error) << "[request Transition] notify condition met but no states written";
                    break;
                }

                OLOG(debug) << "transition notify condition met, reached state: " << newStates.back();
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
            OLOG(debug) << std::setw(30) << k << " = " + m_pluginServices->GetPropertyAsString(k);
        }
        auto chi = m_pluginServices->GetChannelInfo();
        OLOG(debug) << "channel info:";
        for (const auto &k : chi) {
            OLOG(debug) << k.first << " : " << k.second;
        }
    }

    if (newStates.back() == "EXITING") {
        m_pluginServices->ReleaseDeviceControl(FMQ_CONTROLLER_NAME);
        OLOG(debug) << "releasing device control";
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

    OLOG(debug) << "transition done, states visited: " << boost::algorithm::join(newStates, ", ");
    return std::make_tuple(response, grpc::Status::OK);
}
