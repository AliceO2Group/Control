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

#include "OccState.h"

#include <boost/algorithm/string.hpp>

t_State getStateFromString(const std::string& str)
{
    std::string s = boost::algorithm::to_upper_copy(str);
    if (s=="STANDBY") {
        return t_State::standby;
    } else if (s=="CONFIGURED") {
        return t_State::configured;
    } else if (s=="RUNNING") {
        return t_State::running;
    } else if (s=="PAUSED") {
        return t_State::paused;
    } else if (s=="ERROR") {
        return t_State::error;
    } else if (s=="DONE") {
        return t_State::done;
    }
    return t_State::undefined;
}

const std::string getStringFromState(t_State s)
{
    if (s==t_State::standby) {
        return "STANDBY";
    } else if (s==t_State::configured) {
        return "CONFIGURED";
    } else if (s==t_State::running) {
        return "RUNNING";
    } else if (s==t_State::paused) {
        return "PAUSED";
    } else if (s==t_State::error) {
        return "ERROR";
    } else if (s==t_State::done) {
        return "DONE";
    }
    return "UNDEFINED";
}
