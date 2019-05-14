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

class RuntimeControlledObjectPrivate {
    explicit RuntimeControlledObjectPrivate(const std::string objectName)
        : mCurrentState(t_State::undefined)
        , mName(objectName)
        , mCurrentRunNumber(RunNumber_UNDEFINED)
    {}

    virtual ~RuntimeControlledObjectPrivate() = default;

    friend class RuntimeControlledObject;
    friend class OccServer;
private:
    t_State mCurrentState;
    std::string mName;
    RunNumber mCurrentRunNumber;

    int getState(t_State &currentState)
    {
        currentState=mCurrentState;
        return 0;
    }

    void setState(t_State newState)
    {
        mCurrentState=newState;
    }
};
