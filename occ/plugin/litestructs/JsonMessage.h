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

#ifndef OCC_JSONMESSAGE_H
#define OCC_JSONMESSAGE_H

#define RAPIDJSON_HAS_STDSTRING 1

#include "util/Logger.h"

#include <string>
#include <rapidjson/writer.h>
#include <rapidjson/document.h>
#include <grpcpp/impl/codegen/byte_buffer.h>

namespace OccLite
{
namespace nopb
{

class JsonMessage
{
public:
    virtual bool Deserialize(const std::string& s);
    virtual bool Deserialize(::grpc::ByteBuffer* byte_buffer);

    virtual std::string Serialize() const;
    virtual ::grpc::ByteBuffer* SerializeToByteBuffer() const;

    virtual bool Deserialize(const rapidjson::Value& obj) = 0;
    virtual bool Serialize(rapidjson::Writer<rapidjson::StringBuffer>* writer) const = 0;

protected:
    bool InitDocument(const std::string& s, rapidjson::Document& doc);
};

} // namespace nopb
} // namespace OccLite

#endif //OCC_JSONMESSAGE_H
