/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2026 CERN and copyright holders of ALICE O².
 * Author: Piotr Konopka <pkonopka@cern.ch>
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

package trg

import (
	"context"
	"fmt"

	trgpb "github.com/AliceO2Group/Control/core/integration/trg/protos"
	"github.com/segmentio/kafka-go"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/proto"
)

// CtpStatusReader reads CTP status messages from Kafka
type CtpStatusReader struct {
	*kafka.Reader
}

// NewCtpStatusReader creates a new reader for CTP status messages
func NewCtpStatusReader(topic string, groupID string) *CtpStatusReader {
	cfg := kafka.ReaderConfig{
		Brokers:  viper.GetStringSlice("kafkaEndpoints"),
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e7,
	}

	return &CtpStatusReader{
		Reader: kafka.NewReader(cfg),
	}
}

// Next reads the next CTP status message from Kafka
func (r *CtpStatusReader) Next(ctx context.Context) (*trgpb.Status, error) {
	if r == nil {
		return nil, fmt.Errorf("nil reader")
	}
	msg, err := r.ReadMessage(ctx)
	if err != nil {
		return nil, err
	}

	var status trgpb.Status
	if err := proto.Unmarshal(msg.Value, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CTP status message: %w", err)
	}
	return &status, nil
}
