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


#ifndef OCC_OCCSERVER_H
#define OCC_OCCSERVER_H

#include "OccState.h"

#include "protos/occ.pb.h"
#include "protos/occ.grpc.pb.h"

#include <boost/lockfree/queue.hpp>

#include <mutex>
#include <thread>


namespace boost {
namespace property_tree
{
template < class Key, class Data, class KeyCompare >
class basic_ptree;

typedef basic_ptree< std::string, std::string, std::less<std::string> > ptree;
}
}

class RuntimeControlledObject;

const std::unordered_map<std::string, std::string> EXPECTED_FINAL_STATE = {
    {"CONFIGURE",    "CONFIGURED"},
    {"RESET",        "STANDBY"},
    {"START",        "RUNNING"},
    {"STOP",         "CONFIGURED"},
    {"EXIT",         "DONE"},
    {"GO_ERROR",     "ERROR"},
    {"RECOVER",      "STANDBY"},
};

class OccServer final : public occ_pb::Occ::Service
{
public:
    /**
     * Instantiate the gRPC-based control message server.
     *
     * @param rco a pointer to the RuntimeControlledObject-derived state machine.
     *
     * @note This constructor spawns an additional thread which acts as event loop to check the
     *  state of the RuntimeControlledObject.
     *
     * @see OccInstance
     */
    explicit OccServer(RuntimeControlledObject* rco);

    /**
     * Tears down the OccServer.
     */
    virtual ~OccServer();

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

    bool checkMachineDone();

private:
    t_State processStateTransition(const std::string& evt, const boost::property_tree::ptree& properties);
    void updateState(t_State s);

    void publishState(t_State s);
    void pushEvent(occ_pb::DeviceEvent* event);

    void runChecker();

    RuntimeControlledObject* m_rco;
    std::mutex m_mu;

    std::thread m_checkerThread;
    bool m_destroying;
    bool m_machineDone;

    std::unordered_map<std::string, boost::lockfree::queue<t_State>* > m_stateQueues;
    std::unordered_map<std::string, boost::lockfree::queue<occ_pb::DeviceEvent*>* > m_eventQueues;
};


#endif //OCC_OCCSERVER_H
