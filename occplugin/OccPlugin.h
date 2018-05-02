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


#ifndef OCCPLUGIN_OCCPLUGIN_H
#define OCCPLUGIN_OCCPLUGIN_H

#include "OccPluginVersion.h"


#include <fairmq/Plugin.h>
#include <grpc++/grpc++.h>

#include <thread>

using namespace std::literals;

#define OCC_DEFAULT_PORT 47100

class OccPlugin : public fair::mq::Plugin
{
public:
    OccPlugin(const std::string& name,
              const Version& version,
              const std::string& maintainer,
              const std::string& homepage,
              fair::mq::PluginServices* pluginServices);

    virtual ~OccPlugin();

private:
    void runServer(fair::mq::PluginServices*, const std::string&);

    void addTeardownTask(std::function<void()>& func);

    std::thread m_grpcThread;
    std::vector<std::function<void()>> m_teardownTasks;
};

fair::mq::Plugin::ProgOptions OccPluginProgramOptions()
{
    auto plugin_options = boost::program_options::options_description{OCCPLUGIN_DESCRIPTION_SUMMARY};
    plugin_options.add_options()
        ("controlport",
         boost::program_options::value<std::string>(),
         "Port on which the gRPC service will accept connections.");
    return plugin_options;
}


REGISTER_FAIRMQ_PLUGIN(
    OccPlugin,
    OCC,
    (fair::mq::Plugin::Version{OCCPLUGIN_VERSION_MAJOR,
                               OCCPLUGIN_VERSION_MINOR,
                               OCCPLUGIN_VERSION_PATCH}),
    OCCPLUGIN_PRODUCT_MAINTAINER,
    OCCPLUGIN_ORGANIZATION_DOMAIN,
    OccPluginProgramOptions
)


#endif //OCCPLUGIN_OCCPLUGIN_H
