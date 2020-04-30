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

#include "Transition.h"

bool OccLite::nopb::TransitionRequest::Serialize(rapidjson::Writer<rapidjson::StringBuffer>* writer) const
{
    writer->StartObject();

    writer->String("srcState"); writer->String(srcState);
    writer->String("transitionEvent"); writer->String(transitionEvent);
    writer->String("arguments"); writer->StartArray();
    for (const auto& it : arguments) {
        writer->StartObject();
        writer->String("key"); writer->String(it.first);
        writer->String("value"); writer->String(it.second);
        writer->EndObject();
    }
    writer->EndArray();
    writer->EndObject();
    return true;
}

bool OccLite::nopb::TransitionRequest::Deserialize(const rapidjson::Value& obj)
{
    OLOG(INFO) << "Deserializing TransitionRequest";

    srcState = obj["srcState"].GetString();
    transitionEvent = obj["transitionEvent"].GetString();
    OLOG(INFO) << "state and transitionEvent ok";

    if (obj.HasMember("arguments")) {
        auto array = obj["arguments"].GetArray();
        for (const auto& it : array) {
            auto thisItem = it.GetObject();
            arguments[thisItem["key"].GetString()] = thisItem["value"].GetString();
        }
    }
    OLOG(INFO) << "Deserialized TransitionRequest:";
    OLOG(INFO) << JsonMessage::Serialize();

    return true;
}

bool OccLite::nopb::TransitionResponse::Serialize(rapidjson::Writer<rapidjson::StringBuffer>* writer) const
{
    writer->StartObject();

    writer->String("trigger"); writer->Uint(trigger);
    writer->String("state"); writer->String(state);
    writer->String("transitionEvent"); writer->String(transitionEvent);
    writer->String("ok"); writer->Bool(ok);

    writer->EndObject();
    return true;
}

bool OccLite::nopb::TransitionResponse::Deserialize(const rapidjson::Value& obj)
{
    return false;
}
