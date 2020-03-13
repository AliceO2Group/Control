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

#include "RuntimeControlledObject.h"

#include "RuntimeControlledObjectPrivate.h"

#include <boost/property_tree/ptree.hpp>

#include <thread>

using namespace std::chrono_literals;

RuntimeControlledObject::RuntimeControlledObject(const std::string objectName)
    : dPtr(new RuntimeControlledObjectPrivate(objectName))
{
    if (dPtr == nullptr) {
        throw __LINE__;
    }
}

RuntimeControlledObject::~RuntimeControlledObject()
{
    if (dPtr != nullptr) {
        delete dPtr;
        dPtr = nullptr;
    }
}

t_State RuntimeControlledObject::getState() const
{
    return dPtr->mCurrentState;
}

const std::string RuntimeControlledObject::getName() const
{
    return dPtr->mName;
}

int RuntimeControlledObject::executeConfigure(const boost::property_tree::ptree& properties)
{
    return 0;
}

int RuntimeControlledObject::executeReset()
{
    return 0;
}

int RuntimeControlledObject::executeRecover()
{
    return 0;
}

int RuntimeControlledObject::executeStart()
{
    return 0;
}

int RuntimeControlledObject::executeStop()
{
    return 0;
}

int RuntimeControlledObject::executePause()
{
    return 0;
}

int RuntimeControlledObject::executeResume()
{
    return 0;
}

int RuntimeControlledObject::executeExit()
{
    return 0;
}

int RuntimeControlledObject::iterateRunning()
{
    std::this_thread::sleep_for(1s);
    return 0;
}

int RuntimeControlledObject::iterateCheck()
{
    return 0;
}

RunNumber RuntimeControlledObject::getRunNumber() const {
    return dPtr->mCurrentRunNumber;
}

std::string RuntimeControlledObject::getRole() const {
    return dPtr->mRole;
}

void RuntimeControlledObject::setState(t_State state)
{
    dPtr->setState(state);
}

void RuntimeControlledObject::setRole(const std::string& role)
{
    dPtr->mRole = role;
}