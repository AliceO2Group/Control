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

#ifndef OCC_OCCINSTANCE_H
#define OCC_OCCINSTANCE_H

#include "OccGlobals.h"
#include "occ_export.h"

#include <boost/program_options.hpp>

#include <functional>
#include <thread>
#include <vector>

class RuntimeControlledObject;

/**
 * Main controller object of the OCC library.
 *
 * OccInstance spawns a gRPC server in a separate thread in order to receive and react to inbound
 * control commands. These commands are then executed on the global state machine of the process,
 * which the user of this library provides by implementing RuntimeControlledObject.
 */
class OCC_EXPORT OccInstance
{
public:
    /**
     * Creates a new OccInstance, with a control command server thread.
     *
     * @param rco Pointer to the state machine of the process, which must implement
     *  RuntimeControlledObject.
     * @param controlPort Optional inbound TCP port on which to receive control messages. If no port
     *  is provided, this constructor will try to read OCC_CONTROL_PORT from the environment, and
     *  if that too fails, it will fallback to a default control port.
     *
     * @see OccGlobals.h
     *
     * @note This constructor spawns two additional threads: a server thread for gRPC, and an event loop
     *  thread for managing the states of the RuntimeControlledObject (indirectly, via the OccServer
     *  constructor).
     *  Incoming message handlers are triggered from the server thread and eventually result in
     *  calls to the transition functions in RuntimeControlledObject.
     *  Additionally, the event loop thread runs in every state to allow the implementer to report
     *  an error or finished condition.
     *  The OccInstance destructor takes care of safely tearing down this server.
     */
    explicit OccInstance(RuntimeControlledObject *rco, int controlPort = 0);

    /**
     * @overload explicit OccInstance(RuntimeControlledObject *rco, int controlPort = 0);
     *
     * @param vm The boost::program_options::variables_map which the application may provide in order
     *  to simplify handling of the control port command line option.
     *
     * @see OccInstance::ProgramOptions
     */
    explicit OccInstance(RuntimeControlledObject *rco, const boost::program_options::variables_map& vm);

    /**
     * Tears down the OccInstance and its control command server.
     */
    virtual ~OccInstance();

    /**
     * Blocks until the state machine reaches t_State::done.
     *
     * Generally, the application's main function should instantiate the state machine (which
     *  must inherit from RuntimeControlledObject), pass it to the OccInstance constructor,
     *  and finally call wait() in order to yield control until the OCC controller is done.
     *
     * Usage example:
     * @code
     * int main(int argc, char* argv[]) {
     *     ControlledStateMachine csm; // inherits from RuntimeControlledObject
     *     OccInstance occ(&csm); // no control port set, will read it from env var
     *     occ.wait(); // block until all control is done
     *     printf("all done\n");
     * }
     * @endcode
     */
    void wait();

    /**
     * Convenience function for acquiring a control port from command line parameters.
     *
     * @return a boost::program_options::options_description which defines a single program option
     *  (by default --control-port). This object can be merged with the application's main
     *  boost::program_options::options_description, which is then used to parse argv.
     *
     * Usage example:
     * @code
     * boost::program_options::options_description desc("Program options");
     * desc.add(OccInstance::ProgramOptions());
     * boost::program_options::variables_map vm;
     * boost::program_options::store(po::parse_command_line(argc, argv, desc), vm);
     * boost::program_options::notify(vm);
     * ControlledStateMachine csm; // inherits from RuntimeControlledObject
     * OccInstance occ(&csm, vm);
     * @endcode
     */
    static boost::program_options::options_description ProgramOptions();

private:
    void runServer(RuntimeControlledObject *rco, int controlPort);

    void addTeardownTask(std::function<void()>& func);

    std::thread m_grpcThread;
    std::vector<std::function<void()>> m_teardownTasks;
    std::function<bool()> m_checkMachineDone;

    static int portFromVariablesMap(const boost::program_options::variables_map& vm);
};

#endif //OCC_OCCINSTANCE_H
