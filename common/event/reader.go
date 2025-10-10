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
	"sync"
)

// Reader interface provides methods to read events.
type Reader interface {
	Next(ctx context.Context) (*pb.Event, error)
	Close() error
}

// DummyReader is an implementation of Reader that returns no events.
type DummyReader struct{}

// Next returns the next event or nil if there are no more events.
func (*DummyReader) Next(context.Context) (*pb.Event, error) { return nil, nil }

// Close closes the DummyReader.
func (*DummyReader) Close() error { return nil }

// KafkaReader reads events from Kafka and provides a blocking, cancellable API to fetch events.
// Consumption mode is chosen at creation time:
// - latestOnly=false: consume everything (from stored offsets or beginning depending on group state)
// - latestOnly=true: seek to latest offsets on start and only receive messages produced after start
type KafkaReader struct {
	*kafka.Reader
	mu    sync.Mutex
	topic string
}

// NewReaderWithTopic creates a KafkaReader for the provided topic and starts it.
// If latestOnly is true the reader attempts to seek to the latest offsets on start so that
// only new messages (produced after creation) are consumed.
func NewReaderWithTopic(topic topic.Topic, groupID string, latestOnly bool) *KafkaReader {
	cfg := kafka.ReaderConfig{
		Brokers:  viper.GetStringSlice("kafkaEndpoints"),
		Topic:    string(topic),
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e7,
	}

	rk := &KafkaReader{
		Reader: kafka.NewReader(cfg),
		topic:  string(topic),
	}

	if latestOnly {
		// best-effort: set offset to last so we don't replay older messages
		if err := rk.SetOffset(kafka.LastOffset); err != nil {
			log.WithField(infologger.Level, infologger.IL_Devel).
				Warnf("failed to set offset to last offset: %v", err)
		}
	}

	return rk
}

// Next blocks until the next event is available or ctx is cancelled. It returns an error when the reader is closed
// (io.EOF) or the context is cancelled. The caller is responsible for providing a cancellable ctx.
func (r *KafkaReader) Next(ctx context.Context) (*pb.Event, error) {
	if r == nil {
		return nil, fmt.Errorf("nil reader")
	}

	msg, err := r.ReadMessage(ctx)
	if err != nil {
		return nil, err
	}

	event, err := kafkaMessageToEvent(msg)
	if err != nil {
		return nil, err
	}

	return event, nil
}

// Close stops the reader.
func (r *KafkaReader) Close() error {
	if r == nil {
		return nil
	}
	// Close the underlying kafka reader which will cause ReadMessage to return an error
	err := r.Reader.Close()
	if err != nil {
		log.WithField(infologger.Level, infologger.IL_Devel).
			Errorf("failed to close kafka reader: %v", err)
	}
	return err
}

func kafkaMessageToEvent(m kafka.Message) (*pb.Event, error) {
	var evt pb.Event
	if err := proto.Unmarshal(m.Value, &evt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kafka message: %w", err)
	}
	return &evt, nil
}
