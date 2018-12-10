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
#include "OccVersion.h"
#include "RuntimeControlledObject.h"

#include <grpcpp/grpcpp.h>

using namespace std::literals;

OccInstance::OccInstance(RuntimeControlledObject *rco, int controlPort)
{
    if (!controlPort) {
        if (const char* env_controlPort = std::getenv(OCC_CONTROL_PORT_ENV)) {
            controlPort = std::atoi(env_controlPort);
        }
        else {
            controlPort = OCC_DEFAULT_PORT;
            std::cout << "no control port configured, defaulting to " << OCC_DEFAULT_PORT;
        }
    }
    m_grpcThread = std::thread(&OccInstance::runServer, this, rco, controlPort);
}

OccInstance::OccInstance(RuntimeControlledObject *rco, const boost::program_options::variables_map& vm)
    : OccInstance(rco, portFromVariablesMap(vm))
{}

OccInstance::~OccInstance()
{
    std::for_each(m_teardownTasks.begin(),
                  m_teardownTasks.end(),
                  [](std::function<void()>& func) { func(); });

    if (m_grpcThread.joinable()) {
        m_grpcThread.join();
    }
}

void OccInstance::wait()
{
    if (m_grpcThread.joinable()) {
        while (m_checkMachineDone == nullptr || !m_checkMachineDone()) {
            std::this_thread::sleep_for(2ms);
        }
    } else {
        throw std::runtime_error("gRPC server not running");
    }
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
    m_checkMachineDone = std::bind(&OccServer::checkMachineDone, &service);
    server->Wait();
    std::cout << "gRPC server stopped" << std::endl;
}

void OccInstance::addTeardownTask(std::function<void()>& func)
{
    m_teardownTasks.push_back(func);
}

boost::program_options::options_description OccInstance::ProgramOptions()
{
    auto plugin_options = boost::program_options::options_description{OCCLIB_DESCRIPTION_SUMMARY};
    plugin_options.add_options()
        (OCC_CONTROL_PORT_ARG,
         boost::program_options::value<std::string>(),
         "Port on which the gRPC service will accept connections.");
    return plugin_options;
}

int OccInstance::portFromVariablesMap(const boost::program_options::variables_map& vm)
{
    int controlPort = OCC_DEFAULT_PORT;
    if (vm.count(OCC_CONTROL_PORT_ARG)) {
        auto controlPortStr = vm[OCC_CONTROL_PORT_ARG].as<std::string>();
        try {
            controlPort = std::stoi(controlPortStr);
        }
        catch (const std::invalid_argument& e) {
            std::cerr << "bad program argument " << OCC_CONTROL_PORT_ARG << " error: " << e.what() << std::endl;
            std::exit(1);
        }
        catch (const std::out_of_range& e) {
            std::cerr << "control port out of range " << OCC_CONTROL_PORT_ARG << " error: " << e.what() << std::endl;
            std::exit(1);
        }
    }
    return controlPort;
}
