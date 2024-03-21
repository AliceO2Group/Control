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

type Writer struct {
	*kafka.Writer
}

func NewWriterWithTopic(topic topic.Topic) *Writer {
	return &Writer{
		Writer: &kafka.Writer{
			Addr:                   kafka.TCP(viper.GetStringSlice("kafkaEndpoints")...),
			Topic:                  string(topic),
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
		},
	}
}

func (w *Writer) WriteEvent(e interface{}) {
	w.WriteEventWithTimestamp(e, time.Now())
}

func (w *Writer) WriteEventWithTimestamp(e interface{}, timestamp time.Time) {
	go func() {
		var (
			err          error
			wrappedEvent *pb.Event
		)

		switch e := e.(type) {
		case *pb.Ev_MetaEvent_CoreStart:
			wrappedEvent = &pb.Event{
				Timestamp: timestamp.UnixMilli(),
				Payload:   &pb.Event_CoreStartEvent{CoreStartEvent: e},
			}
		case *pb.Ev_MetaEvent_MesosHeartbeat:
			wrappedEvent = &pb.Event{
				Timestamp: timestamp.UnixMilli(),
				Payload:   &pb.Event_MesosHeartbeatEvent{MesosHeartbeatEvent: e},
			}
		case *pb.Ev_MetaEvent_FrameworkEvent:
			wrappedEvent = &pb.Event{
				Timestamp: timestamp.UnixMilli(),
				Payload:   &pb.Event_FrameworkEvent{FrameworkEvent: e},
			}
		case *pb.Ev_TaskEvent:
			wrappedEvent = &pb.Event{
				Timestamp: timestamp.UnixMilli(),
				Payload:   &pb.Event_TaskEvent{TaskEvent: e},
			}
		case *pb.Ev_RoleEvent:
			wrappedEvent = &pb.Event{
				Timestamp: timestamp.UnixMilli(),
				Payload:   &pb.Event_RoleEvent{RoleEvent: e},
			}
		case *pb.Ev_EnvironmentEvent:
			wrappedEvent = &pb.Event{
				Timestamp: timestamp.UnixMilli(),
				Payload:   &pb.Event_EnvironmentEvent{EnvironmentEvent: e},
			}
		case *pb.Ev_CallEvent:
			wrappedEvent = &pb.Event{
				Timestamp: timestamp.UnixMilli(),
				Payload:   &pb.Event_CallEvent{CallEvent: e},
			}
		case *pb.Ev_IntegratedServiceEvent:
			wrappedEvent = &pb.Event{
				Timestamp: timestamp.UnixMilli(),
				Payload:   &pb.Event_IntegratedServiceEvent{IntegratedServiceEvent: e},
			}
		}

		if wrappedEvent == nil {
			err = fmt.Errorf("unsupported event type")
		} else {
			err = w.doWriteEvent(wrappedEvent)
		}

		if err != nil {
			log.WithField("event", e).
				WithField("level", infologger.IL_Support).
				Error(err.Error())
		}
	}()
}

func (w *Writer) doWriteEvent(e *pb.Event) error {
	data, err := proto.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = w.WriteMessages(context.Background(), kafka.Message{
		Value: data,
	})

	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}
	return nil
}
