/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2024 CERN and copyright holders of ALICE O².
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

package event

import (
	"context"

	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	pb "github.com/AliceO2Group/Control/common/protos"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/proto"
)

var log = logger.New(logrus.StandardLogger(), "event")

type Writer struct {
	*kafka.Writer
}

func NewWriterWithTopic(topic topic.Topic) *Writer {
	return &Writer{
		Writer: &kafka.Writer{
			Addr:     kafka.TCP(viper.GetStringSlice("kafkaEndpoints")...),
			Topic:    string(topic),
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (w *Writer) WriteEvent(e *pb.Event) {
	data, err := proto.Marshal(e)
	if err != nil {
		log.WithField("topic", w.Topic).
			WithField("event", e).
			WithError(err).
			Error("failed to marshal event")
	}

	err = w.WriteMessages(context.Background(), kafka.Message{
		Value: data,
	})

	if err != nil {
		log.WithField("topic", w.Topic).
			WithField("event", e).
			WithField("level", infologger.IL_Support).
			Errorf("Kafka message delivery failed: %s", err.Error())
	}
}
