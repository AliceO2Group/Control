/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2025 CERN and copyright holders of ALICE O².
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

package event

import (
	"context"
	"fmt"

	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	pb "github.com/AliceO2Group/Control/common/protos"
	"github.com/segmentio/kafka-go"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/proto"
)

// Reader interface provides methods to read events.
type Reader interface {
	// Next should return the next event or cancel if the context is cancelled.
	Next(ctx context.Context) (*pb.Event, error)
	// Last should return the last available event currently present on the topic (or nil if none)
	// or cancel if the context is cancelled.
	Last(ctx context.Context) (*pb.Event, error)
	Close() error
}

// DummyReader is an implementation of Reader that returns no events.
type DummyReader struct{}

func (*DummyReader) Next(context.Context) (*pb.Event, error) { return nil, nil }
func (*DummyReader) Last(context.Context) (*pb.Event, error) { return nil, nil }
func (*DummyReader) Close() error                            { return nil }

// KafkaReader reads events from Kafka and provides a blocking, cancellable API to fetch events.
type KafkaReader struct {
	*kafka.Reader
	topic   string
	brokers []string
	groupID string
}

// NewReaderWithTopic creates a KafkaReader for the provided topic and starts it.
func NewReaderWithTopic(topic topic.Topic, groupID string) *KafkaReader {
	cfg := kafka.ReaderConfig{
		Brokers:  viper.GetStringSlice("kafkaEndpoints"),
		Topic:    string(topic),
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e7,
	}

	rk := &KafkaReader{
		Reader:  kafka.NewReader(cfg),
		topic:   string(topic),
		brokers: append([]string{}, cfg.Brokers...),
		groupID: groupID,
	}
	return rk
}

// Next blocks until the next event is available or ctx is cancelled.
func (r *KafkaReader) Next(ctx context.Context) (*pb.Event, error) {
	if r == nil {
		return nil, fmt.Errorf("nil reader")
	}
	msg, err := r.ReadMessage(ctx)
	if err != nil {
		return nil, err
	}
	return kafkaMessageToEvent(msg)
}

// Last fetches the last available message on the topic (considering all partitions).
// If multiple partitions have data, the event with the greatest message timestamp is returned.
func (r *KafkaReader) Last(ctx context.Context) (*pb.Event, error) {
	if r == nil {
		return nil, fmt.Errorf("nil reader")
	}
	partitions, err := r.readPartitions()
	if err != nil {
		return nil, err
	}
	var latestEvt *pb.Event
	var latestEvtTimeNs int64
	for _, p := range partitions {
		if p.Topic != r.topic {
			continue
		}
		first, last, err := r.readFirstAndLast(p.ID)
		if err != nil {
			log.WithField(infologger.Level, infologger.IL_Devel).WithError(err).
				Warnf("failed to read offsets for %s[%d]", r.topic, p.ID)
			continue
		}
		if last <= first {
			continue
		}
		msg, err := r.readAtOffset(ctx, p.ID, last-1)
		if err != nil {
			log.WithError(err).
				WithField(infologger.Level, infologger.IL_Devel).
				Warnf("failed to read last message for %s[%d] at offset %d", r.topic, p.ID, last-1)
			continue
		}
		evt, err := kafkaMessageToEvent(msg)
		if err != nil {
			log.WithError(err).
				WithField(infologger.Level, infologger.IL_Devel).
				Warnf("failed to decode last message for %s[%d]", r.topic, p.ID)
			continue
		}
		currentEvtTimeNs := msg.Time.UnixNano()
		if latestEvt == nil || currentEvtTimeNs > latestEvtTimeNs {
			latestEvt = evt
			latestEvtTimeNs = currentEvtTimeNs
		}
	}
	return latestEvt, nil
}

// Close stops the reader.
func (r *KafkaReader) Close() error {
	if r == nil {
		return nil
	}
	return r.Reader.Close()
}

func kafkaMessageToEvent(m kafka.Message) (*pb.Event, error) {
	var evt pb.Event
	if err := proto.Unmarshal(m.Value, &evt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kafka message: %w", err)
	}
	return &evt, nil
}

func (r *KafkaReader) brokerAddr() (string, error) {
	if len(r.brokers) == 0 {
		return "", fmt.Errorf("no kafka brokers configured")
	}
	return r.brokers[0], nil
}

func (r *KafkaReader) readPartitions() ([]kafka.Partition, error) {
	addr, err := r.brokerAddr()
	if err != nil {
		return nil, err
	}
	conn, err := kafka.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return conn.ReadPartitions(r.topic)
}

func (r *KafkaReader) readFirstAndLast(partition int) (int64, int64, error) {
	addr, err := r.brokerAddr()
	if err != nil {
		return 0, 0, err
	}
	conn, err := kafka.DialLeader(context.Background(), "tcp", addr, r.topic, partition)
	if err != nil {
		return 0, 0, err
	}
	defer conn.Close()
	first, last, err := conn.ReadOffsets()
	return first, last, err
}

func (r *KafkaReader) readAtOffset(ctx context.Context, partition int, offset int64) (kafka.Message, error) {
	if offset < 0 {
		return kafka.Message{}, fmt.Errorf("invalid offset %d", offset)
	}
	kr := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   append([]string{}, r.brokers...),
		Topic:     r.topic,
		Partition: partition,
		MinBytes:  1,
		MaxBytes:  10e6,
	})
	defer kr.Close()
	if err := kr.SetOffset(offset); err != nil {
		return kafka.Message{}, err
	}
	return kr.ReadMessage(ctx)
}
