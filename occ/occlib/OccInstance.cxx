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

#include "OccInstance.h"

#include "OccServer.h"
#include "RuntimeControlledObject.h"

#include <grpc++/grpc++.h>

using namespace std::literals;

OccInstance::OccInstance(RuntimeControlledObject *rco, int controlPort)
{
    m_grpcThread = std::thread(&OccInstance::runServer, this, rco, controlPort);
}

OccInstance::~OccInstance()
{
    std::for_each(m_teardownTasks.begin(),
                  m_teardownTasks.end(),
                  [](std::function<void()>& func) { func(); });

    m_grpcThread.join();
}

void OccInstance::runServer(RuntimeControlledObject *rco, int controlPort)
{
    std::string serverAddress("0.0.0.0:"s + std::to_string(controlPort));
    OccServer service(rco);

    grpc::ServerBuilder builder;
    builder.AddListeningPort(serverAddress, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);
    std::unique_ptr<grpc::Server> server(builder.BuildAndStart());
    std::cout << "gRPC server listening on port " << controlPort << std::endl;
    std::function<void()> teardown = [&server]() {
        server->Shutdown();
    };
    addTeardownTask(teardown);
    server->Wait();
    std::cout << "gRPC server stopped" << std::endl;
}

void OccInstance::addTeardownTask(std::function<void()>& func)
{
    m_teardownTasks.push_back(func);
}
