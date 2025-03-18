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

#include "JsonMessage.h"
#include <grpc/slice.h>


std::string OccLite::nopb::JsonMessage::Serialize() const
{
    rapidjson::StringBuffer ss;
    rapidjson::Writer<rapidjson::StringBuffer> writer(ss);
    if (Serialize(&writer))
        return ss.GetString();
    return "";
}

bool OccLite::nopb::JsonMessage::Deserialize(const std::string& s)
{
    rapidjson::Document doc;
    if (!InitDocument(s, doc)) {
        return false;
    }

    Deserialize(doc);

    return true;
}

bool OccLite::nopb::JsonMessage::InitDocument(const std::string& s, rapidjson::Document& doc)
{
    if (s.empty())
        return false;

    std::string validJson(s);

    return !doc.Parse(validJson.c_str()).HasParseError() ? true : false;
}

bool OccLite::nopb::JsonMessage::Deserialize(::grpc::ByteBuffer* byte_buffer)
{
    auto slices = new std::vector<::grpc::Slice>;
    auto status = byte_buffer->Dump(slices);

    if (!status.ok()) {
        OLOG(error) << "Cannot dump JsonMessage slices, error code " << status.error_code() << " " << status.error_message() << " " << status.error_details();
        delete slices;
        return false;
    }

    std::stringstream ss;
    for (auto sl = slices->cbegin(); sl != slices->cend(); sl++) {
        auto rawSlice = sl->c_slice();
        std::string str = grpc::StringFromCopiedSlice(rawSlice);
        ss << str;
        ::grpc_slice_unref(rawSlice);
    }

    OLOG(detail) << "Deserialized JsonMessage: " << ss.str();
    delete slices;
    return Deserialize(ss.str());
}

::grpc::ByteBuffer* OccLite::nopb::JsonMessage::SerializeToByteBuffer() const
{
    std::string str = Serialize();
    OLOG(detail) << "Serialized JsonMessage: " << str;

    // grpc::string = std::string
    // We build a Slice(grpc::string) and we add it to the ByteBuffer
    // as first and only item.
    auto slice = new grpc::Slice(str);
    auto ownBb = new grpc::ByteBuffer(slice, 1);
    return ownBb;
}
