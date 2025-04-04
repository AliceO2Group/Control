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
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common/ecsmetrics"
	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/monitoring"
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

// Kafka writer is used to convert events from events.proto into kafka messages and to write them.
// it is built with 2 workers:
//
//	#1 is gathering kafka.Message from any goroutine which sends message into buffered channel and puts them into FifoBuffer.
//	#2 is poping any messages from FifoBuffer and sends them to Kafka
//
//	The reason for this setup over setting Async: true in kafka.Writer is the ability to have some error handling
//	of failed messages. Moreover if we used only one worker that gathers messages from channel and then sends them directly to Kafka,
//	we would block whole core if we receive lot of messages at once. So we split functionality into two workers: one is
//	putting all messages into the buffer, so if we have a lot of messages buffer just grows without blocking whole core and the
//	second does all the sending. This setup allows us to gather messages from any amount of goroutines without blocking/losing messages.
//	Another benefit is batching messages instead of writing them one by one.
type KafkaWriter struct {
	*kafka.Writer
	toBatchMessagesChan chan kafka.Message
	messageBuffer       FifoBuffer[kafka.Message]
	// NOTE: we use settable callback in order to be able to test writing of messages via KafkaWriter, without necessity of setting up cluster
	writeFunction  func([]kafka.Message)
	runningWorkers sync.WaitGroup
	batchingDoneCh chan struct{}
}

func NewWriterWithTopic(topic topic.Topic) *KafkaWriter {
	writer := &KafkaWriter{
		Writer: &kafka.Writer{
			Addr:                   kafka.TCP(viper.GetStringSlice("kafkaEndpoints")...),
			Topic:                  string(topic),
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: true,
		},
		toBatchMessagesChan: make(chan kafka.Message, 100),
		messageBuffer:       NewFifoBuffer[kafka.Message](),
		runningWorkers:      sync.WaitGroup{},
		batchingDoneCh:      make(chan struct{}, 1),
	}

	writer.writeFunction = func(messages []kafka.Message) {
		if err := writer.WriteMessages(context.Background(), messages...); err != nil {
			metric := ecsmetrics.NewMetric("kafka")
			metric.AddTag("topic", writer.Topic)
			metric.AddValue("failedsentmessages", len(messages))
			monitoring.Send(metric)
			log.Errorf("failed to write %d messages to kafka with error: %v", len(messages), err)
		}
	}

	go writer.writingLoop()
	go writer.batchingLoop()

	return writer
}

func (w *KafkaWriter) Close() {
	if w != nil {
		w.runningWorkers.Add(2)
		close(w.toBatchMessagesChan)
		w.runningWorkers.Wait()
		w.Writer.Close()
	}
}

func (w *KafkaWriter) WriteEvent(e interface{}) {
	if w != nil {
		w.WriteEventWithTimestamp(e, time.Now())
	}
}

func (w *KafkaWriter) writingLoop() {
	for {
		select {
		case <-w.batchingDoneCh:
			w.runningWorkers.Done()
			return
		default:
			messagesToSend := w.messageBuffer.PopMultiple(100)
			if len(messagesToSend) == 0 {
				continue
			}

			w.writeFunction(messagesToSend)

			metric := ecsmetrics.NewMetric("kafka")
			metric.AddTag("topic", w.Topic)
			metric.AddValue("sentmessages", len(messagesToSend))
			monitoring.Send(metric)
		}
	}
}

func (w *KafkaWriter) batchingLoop() {
	for message := range w.toBatchMessagesChan {
		w.messageBuffer.Push(message)
	}
	w.batchingDoneCh <- struct{}{}
	w.messageBuffer.ReleaseGoroutines()
	w.runningWorkers.Done()
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

// TODO: there should be written test to verify converting all of these messages
func internalEventToKafkaEvent(internalEvent interface{}, timestamp time.Time) (kafkaEvent *pb.Event, key []byte, err error) {
	kafkaEvent = &pb.Event{
		Timestamp:     timestamp.UnixMilli(),
		TimestampNano: timestamp.UnixNano(),
	}

	switch e := internalEvent.(type) {
	case *pb.Ev_MetaEvent_CoreStart:
		kafkaEvent.Payload = &pb.Event_CoreStartEvent{CoreStartEvent: e}
	case *pb.Ev_MetaEvent_MesosHeartbeat:
		kafkaEvent.Payload = &pb.Event_MesosHeartbeatEvent{MesosHeartbeatEvent: e}
	case *pb.Ev_MetaEvent_FrameworkEvent:
		kafkaEvent.Payload = &pb.Event_FrameworkEvent{FrameworkEvent: e}
	case *pb.Ev_TaskEvent:
		key = []byte(e.Taskid)
		if len(key) == 0 {
			key = nil
		}
		kafkaEvent.Payload = &pb.Event_TaskEvent{TaskEvent: e}
	case *pb.Ev_RoleEvent:
		key = extractAndConvertEnvID(e)
		kafkaEvent.Payload = &pb.Event_RoleEvent{RoleEvent: e}
	case *pb.Ev_EnvironmentEvent:
		key = extractAndConvertEnvID(e)
		kafkaEvent.Payload = &pb.Event_EnvironmentEvent{EnvironmentEvent: e}
	case *pb.Ev_CallEvent:
		key = extractAndConvertEnvID(e)
		kafkaEvent.Payload = &pb.Event_CallEvent{CallEvent: e}
	case *pb.Ev_IntegratedServiceEvent:
		key = extractAndConvertEnvID(e)
		kafkaEvent.Payload = &pb.Event_IntegratedServiceEvent{IntegratedServiceEvent: e}
	case *pb.Ev_RunEvent:
		key = extractAndConvertEnvID(e)
		kafkaEvent.Payload = &pb.Event_RunEvent{RunEvent: e}
	default:
		err = fmt.Errorf("unsupported event type")
	}

	return
}

func kafkaEventToKafkaMessage(kafkaEvent *pb.Event, key []byte) (kafka.Message, error) {
	data, err := proto.Marshal(kafkaEvent)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("failed to marshal event: %w", err)
	}

	message := kafka.Message{
		Value: data,
	}

	if key != nil {
		message.Key = key
	}

	return message, nil
}

func (w *KafkaWriter) WriteEventWithTimestamp(e interface{}, timestamp time.Time) {
	if w == nil {
		return
	}

	wrappedEvent, key, err := internalEventToKafkaEvent(e, timestamp)
	if err != nil {
		log.WithField("event", e).
			WithField("level", infologger.IL_Support).
			Errorf("Failed to convert event to kafka event: %s", err.Error())
		return
	}

	message, err := kafkaEventToKafkaMessage(wrappedEvent, key)
	if err != nil {
		log.WithField("event", e).
			WithField("level", infologger.IL_Support).
			Errorf("Failed to convert kafka event to message: %s", err.Error())
		return
	}

	w.toBatchMessagesChan <- message
}
