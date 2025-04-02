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
	"fmt"
	"time"

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

type Writer interface {
	WriteEvent(e interface{})
	WriteEventWithTimestamp(e interface{}, timestamp time.Time)
	Close()
}

type DummyWriter struct{}

func (*DummyWriter) WriteEvent(interface{})                         {}
func (*DummyWriter) WriteEventWithTimestamp(interface{}, time.Time) {}
func (*DummyWriter) Close()                                         {}

type KafkaWriter struct {
	*kafka.Writer
}

func NewWriterWithTopic(topic topic.Topic) *KafkaWriter {
	return &KafkaWriter{
		Writer: &kafka.Writer{
			Addr:                   kafka.TCP(viper.GetStringSlice("kafkaEndpoints")...),
			Topic:                  string(topic),
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: true,
		},
	}
}

func (w *KafkaWriter) Close() {
	if w != nil {
		w.Close()
	}
}

func (w *KafkaWriter) WriteEvent(e interface{}) {
	if w != nil {
		w.WriteEventWithTimestamp(e, time.Now())
	}
}

type HasEnvID interface {
	GetEnvironmentId() string
}

func extractAndConvertEnvID[T HasEnvID](object T) []byte {
	envID := []byte(object.GetEnvironmentId())
	if len(envID) > 0 {
		return envID
	}
	return nil
}

func (w *KafkaWriter) WriteEventWithTimestamp(e interface{}, timestamp time.Time) {
	if w == nil {
		return
	}

	var (
		err          error
		wrappedEvent *pb.Event = &pb.Event{
			Timestamp:     timestamp.UnixMilli(),
			TimestampNano: timestamp.UnixNano(),
		}
		key []byte = nil
	)

	switch e := e.(type) {
	case *pb.Ev_MetaEvent_CoreStart:
		wrappedEvent.Payload = &pb.Event_CoreStartEvent{CoreStartEvent: e}
	case *pb.Ev_MetaEvent_MesosHeartbeat:
		wrappedEvent.Payload = &pb.Event_MesosHeartbeatEvent{MesosHeartbeatEvent: e}
	case *pb.Ev_MetaEvent_FrameworkEvent:
		wrappedEvent.Payload = &pb.Event_FrameworkEvent{FrameworkEvent: e}
	case *pb.Ev_TaskEvent:
		key = []byte(e.Taskid)
		if len(key) == 0 {
			key = nil
		}
		wrappedEvent.Payload = &pb.Event_TaskEvent{TaskEvent: e}
	case *pb.Ev_RoleEvent:
		key = extractAndConvertEnvID(e)
		wrappedEvent.Payload = &pb.Event_RoleEvent{RoleEvent: e}
	case *pb.Ev_EnvironmentEvent:
		key = extractAndConvertEnvID(e)
		wrappedEvent.Payload = &pb.Event_EnvironmentEvent{EnvironmentEvent: e}
	case *pb.Ev_CallEvent:
		key = extractAndConvertEnvID(e)
		wrappedEvent = &pb.Event{
			Timestamp:     timestamp.UnixMilli(),
			TimestampNano: timestamp.UnixNano(),
			Payload:       &pb.Event_CallEvent{CallEvent: e},
		}
	case *pb.Ev_IntegratedServiceEvent:
		key = extractAndConvertEnvID(e)
		wrappedEvent.Payload = &pb.Event_IntegratedServiceEvent{IntegratedServiceEvent: e}
	case *pb.Ev_RunEvent:
		key = extractAndConvertEnvID(e)
		wrappedEvent.Payload = &pb.Event_RunEvent{RunEvent: e}
	default:
		wrappedEvent = nil
	}

	if wrappedEvent == nil {
		err = fmt.Errorf("unsupported event type")
	} else {
		err = w.doWriteEvent(key, wrappedEvent)
	}

	if err != nil {
		log.WithField("event", e).
			WithField("level", infologger.IL_Support).
			Error(err.Error())
	}
}

func (w *KafkaWriter) doWriteEvent(key []byte, e *pb.Event) error {
	if w == nil {
		return nil
	}

	data, err := proto.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	message := kafka.Message{
		Value: data,
	}

	if key != nil {
		message.Key = key
	}

	err = w.WriteMessages(context.Background(), message)
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}
	return nil
}
