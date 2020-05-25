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

#include "OccPlugin.h"

#ifdef OCC_LITE_SERVICE
#include "OccLiteServer.h"
#else
#include "OccPluginServer.h"
#endif

#include "util/Logger.h"

#include <fairmq/PluginManager.h>

#include <grpcpp/grpcpp.h>

OccPlugin::OccPlugin(const std::string& name,
                     const fair::mq::Plugin::Version& version,
                     const std::string& maintainer,
                     const std::string& homepage,
                     fair::mq::PluginServices* pluginServices)
    : Plugin(name, version, maintainer, homepage, pluginServices)
{
    // Debug: list of FairMQ property keys
    //    auto pk = GetPropertyKeys();
    //    std::for_each( pk.begin(), pk.end(), [](auto it){OLOG(DEBUG) << "\t" << it; } );

    auto controlPort = std::to_string(OCC_DEFAULT_PORT);
    try {
        controlPort = GetPropertyAsString(OCC_CONTROL_PORT_ARG);
    }
    catch (std::exception& e) {
        OLOG(DEBUG) << "O² control port not specified, defaulting to " << OCC_DEFAULT_PORT;
    }

    try {
        TakeDeviceControl();
    }
    catch (fair::mq::PluginServices::DeviceControlError& e) {
        // If we are here, it means another plugin has taken control.
        OLOG(ERROR) << "Cannot take device control" << e.what();
    }

    m_grpcThread = std::thread(&OccPlugin::runServer, this, pluginServices, controlPort);
}

OccPlugin::~OccPlugin()
{
    std::for_each(m_teardownTasks.begin(),
                  m_teardownTasks.end(),
                  [](std::function<void()>& func) { func(); });

    m_grpcThread.join();
}

void OccPlugin::runServer(fair::mq::PluginServices* pluginServices, const std::string& controlPort)
{
    std::string serverAddress("0.0.0.0:"s + controlPort);
#ifdef OCC_LITE_SERVICE
    OccLite::Service service(pluginServices);
#else
    OccPluginServer service(pluginServices);
#endif

    grpc::ServerBuilder builder;
    builder.AddListeningPort(serverAddress, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);
    std::unique_ptr<grpc::Server> server(builder.BuildAndStart());

#ifdef OCC_LITE_SERVICE
    OLOG(DEBUG) << OCCLITE_PRODUCT_NAME << " v" << OCC_VERSION << " listening on port " << controlPort;
#else
    OLOG(DEBUG) << OCCPLUGIN_PRODUCT_NAME << " (legacy) v" << OCC_VERSION << " listening on port " << controlPort;
#endif
    std::function<void()> teardown = [&server]() {
        server->Shutdown();
    };
    addTeardownTask(teardown);
    server->Wait();
    OLOG(DEBUG) << "OCC control server stopped";
}

void OccPlugin::addTeardownTask(std::function<void()>& func)
{
    m_teardownTasks.push_back(func);
}
