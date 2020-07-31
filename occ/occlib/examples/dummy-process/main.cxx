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

#include "ControlledStateMachine.h"

#include <OccInstance.h>

#include <boost/program_options.hpp>
#include <Configuration/ConfigurationFactory.h>

namespace po = boost::program_options;
using o2::configuration::ConfigurationFactory;

int main(int argc, char* argv[]) {
    po::options_description desc("Program options");
    // Define your own program options here
    // ...
    // finally, the ones from OccInstance must be appended in order to handle --control-port:
    desc.add(OccInstance::ProgramOptions());

    desc.add_options()("config", po::value<std::string>(), "Config file URL");

    // Boost::program_options boilerplate...
    po::variables_map vm;
    po::store(po::parse_command_line(argc, argv, desc), vm);
    po::notify(vm);

    // Instantiate your state machine which inherits from RuntimeControlledObject:
    ControlledStateMachine csm{};
    // Nothing is happening yet, the state machine starts in t_State::undefined.

    // Habdle loading file from config
    if (vm.count("config")) {
      csm.setConfig(ConfigurationFactory::getConfiguration(vm["config"].as<std::string>())->getRecursive(""));
    }

    // Instantiate the O² Control and Configuration interface:
    OccInstance occ(&csm, vm);
    // The OccInstance constructor immediately starts the gRPC server thread, which
    // in turn creates an internal OccServer instance with its own state checker
    // event loop.
    // The end of the OccInstance constructor does not guarantee that the gRPC
    // server is ready to accept requests or that the OccServer checker loop thread
    // has started.
    // However, the gRPC server only accepts requests after the the machine state
    // has already become t_State::standby.

    // Block until t_State::done is reached:
    occ.wait();

    // No need for further cleanup, the OccInstance should destroy its gRPC
    // interface and extra threads gracefully when it goes out of scope.
}
