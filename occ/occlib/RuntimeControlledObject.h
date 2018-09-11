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

#ifndef OCC_RUNTIMECONTROLLEDOBJECT_H
#define OCC_RUNTIMECONTROLLEDOBJECT_H

#include "OccState.h"
#include "occ_export.h"

class RuntimeControlledObjectPrivate;

class OCC_EXPORT RuntimeControlledObject {
public:
    explicit RuntimeControlledObject(const std::string objectName);
    virtual ~RuntimeControlledObject();

    const std::string getName();

    t_State getState();

    virtual int executeConfigure(const PropertyMap& properties); // to go from standby to configured
    virtual int executeReset();   // to go from configured to standby
    virtual int executeRecover(); // to go from error to standby
    virtual int executeStart();   // to go from configured to running
    virtual int executeStop();    // to go from running/paused to configured
    virtual int executePause();   // to go from running to paused
    virtual int executeResume();  // to go from paused to running
    virtual int executeExit();    // to go from standby/configured to done

    // ↓ called by event loop in OccServer
    virtual int iterateRunning();     // called continuously in state 'running'
    virtual int iterateCheck();       // called periodically in any state

private:
    RuntimeControlledObjectPrivate *dPtr;

    void setState(t_State state);

    friend class OccServer;
};


#endif //OCC_RUNTIMECONTROLLEDOBJECT_H
