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

#ifndef OCC_TRANSITION_H
#define OCC_TRANSITION_H


#include "JsonMessage.h"

#include <grpcpp/impl/codegen/serialization_traits.h>
#include <grpcpp/impl/codegen/service_type.h>

#include <rapidjson/prettywriter.h>
#include <sstream>

namespace OccLite
{
namespace nopb
{

struct ConfigEntry : public JsonMessage {
    std::string key;
    std::string value;

    bool Serialize(rapidjson::Writer<rapidjson::StringBuffer>* writer) const override;
    bool Deserialize(const rapidjson::Value& obj) override;
};
struct TransitionRequest : public JsonMessage
{
    std::string srcState;
    std::string transitionEvent;
    std::vector<ConfigEntry> arguments;

    bool Serialize(rapidjson::Writer<rapidjson::StringBuffer>* writer) const override;
    bool Deserialize(const rapidjson::Value& obj) override;
};

enum StateChangeTrigger : unsigned {
    EXECUTOR = 0,
    DEVICE_INTENTIONAL = 1,
    DEVICE_ERROR = 2
};

struct TransitionResponse : public JsonMessage
{
    StateChangeTrigger trigger; // can be EXECUTOR DEVICE_INTENTIONAL DEVICE_ERROR
    std::string state;
    std::string transitionEvent;
    bool ok;

    bool Serialize(rapidjson::Writer<rapidjson::StringBuffer>* writer) const override;
    bool Deserialize(const rapidjson::Value& obj) override;
};

} // namespace nopb
} // namespace OccLite


namespace grpc
{
template<>
class SerializationTraits<OccLite::nopb::TransitionRequest, void>
{
public:
    static Status Deserialize(ByteBuffer* byte_buffer, OccLite::nopb::TransitionRequest* dest)
    {
        bool ok = dest->JsonMessage::Deserialize(byte_buffer);
        std::cout << "slice dump:" << dest->JsonMessage::Serialize() << std::endl;
        return ok ? Status::OK : Status::CANCELLED;
    }

    static Status Serialize(const OccLite::nopb::TransitionRequest& source, ByteBuffer* buffer,
                            bool* own_buffer)
    {
        *buffer = *source.JsonMessage::SerializeToByteBuffer();
        *own_buffer = true;
        return Status::OK;
    }
};

template<>
class SerializationTraits<OccLite::nopb::TransitionResponse, void>
{
public:
    static Status Deserialize(ByteBuffer* byte_buffer, OccLite::nopb::TransitionResponse* dest)
    {
        bool ok = dest->JsonMessage::Deserialize(byte_buffer);
        std::cout << "ok: " << ok << "  slice dump:" << dest->JsonMessage::Serialize() << std::endl;
        return ok ? Status::OK : Status::CANCELLED;
    }

    static Status Serialize(const OccLite::nopb::TransitionResponse& source,
                            ByteBuffer* buffer,
                            bool* own_buffer)
    {
        *buffer = *source.JsonMessage::SerializeToByteBuffer();
        *own_buffer = true;
        return Status::OK;
    }
};

} // namespace grpc

#endif //OCC_TRANSITION_H
