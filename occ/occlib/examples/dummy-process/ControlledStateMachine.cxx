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

#include <boost/property_tree/json_parser.hpp>

#include <iostream>

#define LOG_SCOPE RaiiLogEntry obj ## __LINE__ (__FUNCTION__);

struct RaiiLogEntry
{
    RaiiLogEntry(const char *f) : f_(f) { printf("BEGIN function %s\n", f_); }
    ~RaiiLogEntry() { printf("END function %s\n", f_); }
    const char *f_;
};


int ControlledStateMachine::executeConfigure(const boost::property_tree::ptree& properties)
{
    LOG_SCOPE
    printf("received runtime configuration:\n");
    std::stringstream ss;
    boost::property_tree::json_parser::write_json(ss, properties);
    printf("%s\n", ss.str().c_str());

    return RuntimeControlledObject::executeConfigure(properties);
}

int ControlledStateMachine::executeReset()
{
    LOG_SCOPE
    return RuntimeControlledObject::executeReset();
}

int ControlledStateMachine::executeRecover()
{
    LOG_SCOPE
    return RuntimeControlledObject::executeRecover();
}

int ControlledStateMachine::executeStart()
{
    LOG_SCOPE
    return RuntimeControlledObject::executeStart();
}

int ControlledStateMachine::executeStop()
{
    LOG_SCOPE
    return RuntimeControlledObject::executeStop();
}

int ControlledStateMachine::executePause()
{
    LOG_SCOPE
    return RuntimeControlledObject::executePause();
}

int ControlledStateMachine::executeResume()
{
    LOG_SCOPE
    return RuntimeControlledObject::executeResume();
}

int ControlledStateMachine::executeExit()
{
    LOG_SCOPE
    return RuntimeControlledObject::executeExit();
}

int ControlledStateMachine::iterateRunning()
{
    LOG_SCOPE
    return RuntimeControlledObject::iterateRunning();
}

int ControlledStateMachine::iterateCheck()
{
    //LOG_SCOPE
    return RuntimeControlledObject::iterateCheck();
}
