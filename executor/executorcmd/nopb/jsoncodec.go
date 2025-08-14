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

// Package nopb provides non-protobuf encoding functionality including
// JSON codec implementations for gRPC communication.
package nopb

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
)

func init() {
	encoding.RegisterCodec(&JsonCodec{})
}

// Satisfies interface grpc.encoding.Codec
type JsonCodec struct{}

func (*JsonCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (*JsonCodec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (*JsonCodec) Name() string {
	return "json"
}
