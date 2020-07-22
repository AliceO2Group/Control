/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

#include "GetState.h"

bool OccLite::nopb::GetStateResponse::Serialize(rapidjson::Writer<rapidjson::StringBuffer>* writer) const
{
    writer->StartObject();

    writer->String("state");
    writer->String(state);

    writer->String("pid");
    writer->Int(pid);

    writer->EndObject();
    return true;
}

bool OccLite::nopb::GetStateResponse::Deserialize(const rapidjson::Value& obj)
{
    state = obj["state"].GetString();
    pid = obj["pid"].GetInt();
    return true;
}

bool OccLite::nopb::GetStateRequest::Serialize(rapidjson::Writer<rapidjson::StringBuffer>* writer) const
{
    writer->StartObject();
    writer->EndObject();
    return true;
}

bool OccLite::nopb::GetStateRequest::Deserialize(const rapidjson::Value& obj)
{
    return true;
}
