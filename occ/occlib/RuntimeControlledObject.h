/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
 *         Sylvain Chapeland <sylvain.chapeland@cern.ch>
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

#ifndef OCC_RUNTIMECONTROLLEDOBJECT_H
#define OCC_RUNTIMECONTROLLEDOBJECT_H

#include <cstdint> 
#include "OccState.h"
#include "occ_export.h"

namespace boost {
namespace property_tree
{
template < class Key, class Data, class KeyCompare >
class basic_ptree;

typedef basic_ptree< std::string, std::string, std::less<std::string> > ptree;
}
}

typedef uint32_t RunNumber;
const RunNumber RunNumber_UNDEFINED = 0;


class RuntimeControlledObjectPrivate;

class OCC_EXPORT RuntimeControlledObject {
public:
    /**
     * Creates a new RuntimeControlledObject instance, should be called by implementer's constructor.
     *
     * @param objectName A descriptive name for the state machine driven task. Should be
     *  alphanumeric, as it's a potentially user-visible string to identify the program. It does not
     *  have to be unique to this instance.
     */
    explicit RuntimeControlledObject(const std::string objectName);

    /**
     * Default destructor.
     */
    virtual ~RuntimeControlledObject();

    /**
     * Returns the name of the RuntimeControlledObject as set in the constructor.
     *
     * @return a std::string with the object's name.
     */
    const std::string getName() const;

    /**
     * Returns the current state of the controlled state machine.
     *
     * @return a t_State representing the state of the machine.
     *
     * @see OccState.h
     */
    t_State getState() const;

    /// Manually sets configuration ptree that is passed to user conde in the CONFIGURED state
    void setConfig(const boost::property_tree::ptree& properties);

    /// Configuration ptree getter
    boost::property_tree::ptree getConfig();

    /// Transition from standby to configured.
    /// 
    /// @param properties a boost::property_tree pushed by the control agent, containing
    ///  deployment-specific configuration (i.e. channel configuration and related).
    /// @return 0 if the transition completed successfully, any non-zero value immediately triggers
    ///  a transition to the error state.
    /// 
    /// The implementer should use this transition to move the machine from an unconfigured, bare
    /// state into a state where the dataflow may be initiated at any time.
    /// It is ok for this step to take some time if necessary.
    /// 
    /// Example properties tree with one inbound and one outbound channel:
    ///   {
    ///       "chans": {
    ///           "myOutboundCh": {
    ///               "0": {
    ///                   "address": "tcp://target.hostname.cern.ch:5555",
    ///                   "method": "connect",       // can be connect, bind
    ///                   "type": "pull",            // can be push, pull, pub, sub
    ///                   "transport": "default",
    ///                   "rateLogging": "0",
    ///                   "sndBufSize": "1000",
    ///                   "sndKernelSize": "0",
    ///                   "rcvBufSize": "1000",
    ///                   "rcvKernelSize": "0"
    ///               },
    ///               "numSockets": "1"
    ///           },
    ///           "myInboundCh": {
    ///               "0": {
    ///                   "address": "tcp://*:5555",
    ///                   "method": "bind",
    ///                   "type": "push",
    ///                   "transport": "default",
    ///                   "rateLogging": "0",
    ///                   "sndBufSize": "1000",
    ///                   "sndKernelSize": "0",
    ///                   "rcvBufSize": "1000",
    ///                   "rcvKernelSize": "0"
    ///               },
    ///               "numSockets": "1"
    ///           }
    ///       },
    ///       "additional non-channel properties": "go here"
    ///   }
    /// 
    /// Example of correspondence between Readout configuration file and the equivalent
    /// reconfiguration information pushed by AliECS:
    /// 
    ///   [consumer-fmq-wp5]
    ///   # session name must match --session parameter of all O2 devices in the chain
    ///   consumerType=FairMQChannel
    ///   enabled=0
    ///   sessionName=default                          \
    ///   transportType=shmem                           \
    ///   channelName=readout-out                        > can be overridden in incoming tree
    ///   channelType=pair                              /
    ///   channelAddress=ipc:///tmp/readout-pipe-0     /
    ///   unmanagedMemorySize=2G
    ///   disableSending=0
    ///   #need also a memory pool for headers and partial HBf chunks copies
    ///   memoryPoolNumberOfPages=100
    ///   memoryPoolPageSize=128k
    /// 
    /// Incoming tree:
    ///   {
    ///       "chans": {
    ///           "readout-out": {                   // should be matched against channelName
    ///               "0": {
    ///                   "method": "connect",       // can be connect, bind
    ///                   "address": "tcp://target.hostname.cern.ch:5555",  // if "method" is "bind", "address" can be e.g. "tcp://*:5555"
    ///                   "type": "push",            // can be push, pull, pub, sub
    ///                   "transport": "shmem",
    ///                   "rateLogging": "0",        // additional channel options not specified in config file
    ///                   "sndBufSize": "1000",
    ///                   "sndKernelSize": "0",
    ///                   "rcvBufSize": "1000",
    ///                   "rcvKernelSize": "0"
    ///               },
    ///               "numSockets": "1"              // this is always 1 because we enforce 1 connection per channel
    ///           },
    ///       },
    ///       "additional non-channel properties": "go here"
    ///   }
    /// 
    /// @note Only one of the transition functions will be called at any given time, and during a
    ///  transition all checks (iterateRunning/iterateCheck) are blocked until the transition
    ///  finishes and returns success or error.
    virtual int executeConfigure(const boost::property_tree::ptree& properties);

    /**
     * Transition from configured to standby.
     *
     * @return 0 if the transition completed successfully, any non-zero value immediately triggers
     *  a transition to the error state.
     *
     * The implementer should use this transition to move the machine from a configured state where
     * the dataflow is ready to start (or has recently ended) into a bare, unconfigured state.
     * Care should be taken to either correctly clear all configuration in this transition, or to
     * make the (opposite) Configure transition idempotent in order to avoid keeping hidden state
     * data.
     */
    virtual int executeReset();

    /**
     * Transition from error to standby.
     *
     * @return 0 if the transition completed successfully, any non-zero value immediately triggers
     *  a transition to the error state.
     *
     * The implementer should use this transition to recover from the error state.
     */
    virtual int executeRecover();

    /**
     * Transition from configured to running.
     *
     * @return 0 if the transition completed successfully, any non-zero value immediately triggers
     *  a transition to the error state.
     *
     * The implementer should use this transition to initiate the data flow. When this function
     * exits successfully, the running state is reached, in which iterateRunning is called
     * periodically to drive the data processing.
     */
    virtual int executeStart();

    /**
     * Transition from running or paused to configured.
     *
     * @return 0 if the transition completed successfully, any non-zero value immediately triggers
     *  a transition to the error state.
     *
     * The implementer should use this transition to terminate the data flow.
     */
    virtual int executeStop();

    /**
     * Transition from running to paused.
     *
     * @return 0 if the transition completed successfully, any non-zero value immediately triggers
     *  a transition to the error state.
     *
     * The implementer should use this transition to temporarily pause data processing. The paused
     * state should not imply a change in configuration, the only difference compared with the
     * running state is the absence of periodic iterateRunning calls.
     */
    virtual int executePause();

    /**
     * Transition from paused to running.
     *
     * @return 0 if the transition completed successfully, any non-zero value immediately triggers
     *  a transition to the error state.
     *
     * The implementer should use this transition to resume data processing after a temporary pause.
     */
    virtual int executeResume();

    /**
     * Transition from standby or configured to done.
     *
     * @return 0 if the transition completed successfully, any non-zero value immediately triggers
     *  a transition to the error state.
     *
     * The implementer should use this transition to safely release all resources in preparation for
     * process exit.
     */
    virtual int executeExit();


    /**
     * Execute periodic actions, as required by the running state.
     *
     * @return 0 if the operation completed successfully and the machine can stay in the running state,
     *  1 if all data processing is done and the implementer wishes to notify the machine control
     *  mechanism of this condition (send END_OF_STREAM event), or any other value which immediately
     *  triggers a transition to the error state.
     *
     * This function is called continuously by OccServer::runChecker if the state machine is in the
     * state t_State::running. It is never called outside this state.
     */
    virtual int iterateRunning();

    /**
     * Perform periodic checks during every state except ERROR.
     *
     * @return 0 if the check completed successfully and the machine can stay in the current state,
     *  or any other value to immediately trigger a transition to the error state.
     *
     * This function is called continuously by OccServer::runChecker in any state except ERROR, including
     * t_State::running. Its purpose is for the implementer to report an unusual condition in
     * order to trigger a transition to t_State::error.
     */
    virtual int iterateCheck();

protected:
    /**
     * Acquire the current run number if a run is underway.
     *
     * @return the run number as an unsigned integer, or RunNumber_UNDEFINED
     */
    RunNumber getRunNumber() const;

    /**
     * Get the role for this task/machine.
     * @return the role, as a string (length < 32).
     */
    std::string getRole() const;

private:
    RuntimeControlledObjectPrivate *dPtr;

    void setRole(const std::string& role);
    void setState(t_State state);

    friend class OccServer;
    friend class OccInstance;
};


#endif //OCC_RUNTIMECONTROLLEDOBJECT_H
