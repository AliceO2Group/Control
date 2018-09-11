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


#ifndef OCC_CONTROLLEDSTATEMACHINE_H
#define OCC_CONTROLLEDSTATEMACHINE_H

#include <RuntimeControlledObject.h>

class ControlledStateMachine : public RuntimeControlledObject
{
public:
    explicit ControlledStateMachine() : RuntimeControlledObject("Dummy Process") {}

    int executeConfigure(const PropertyMap& properties) override; // to go from standby to configured
    int executeReset() override;   // to go from configured to standby
    int executeRecover() override; // to go from error to standby
    int executeStart() override;   // to go from configured to running
    int executeStop() override;    // to go from running/paused to configured
    int executePause() override;   // to go from running to paused
    int executeResume() override;  // to go from paused to running
    int executeExit() override;    // to go from standby/configured to done

    // ↓ called by event loop in OccServer
    int iterateRunning() override;     // called continuously in state 'running'
    int iterateCheck() override;       // called periodically in any state
};


#endif //OCC_CONTROLLEDSTATEMACHINE_H
