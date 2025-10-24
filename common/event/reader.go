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
	"sync"

	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	pb "github.com/AliceO2Group/Control/common/protos"
	"github.com/segmentio/kafka-go"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/proto"
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
// WARNING: If you will create more than one reader with groupID != nil and latestOnly == true
// there might inconsitencies while reading.
func NewReaderWithTopic(topic topic.Topic, groupID string, latestOnly bool) *KafkaReader {
	cfg := kafka.ReaderConfig{
		Brokers:  viper.GetStringSlice("kafkaEndpoints"),
		Topic:    string(topic),
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e7,
	}

	// Kafka-io only enables to SetOffset for whole consumer group which muset be done before consumer is created.
	// WARNING: This will might cause issues if we are creating more than one readers
	if latestOnly && groupID != "" {
		if err := resetGroupToLatest(context.Background(), viper.GetStringSlice("kafkaEndpoints"), string(topic), groupID); err != nil {
			log.WithField(infologger.Level, infologger.IL_Devel).
				Errorf("failed to set offset to last offset for whole consumer group %s: %v", groupID, err)
		}
	}

	rk := &KafkaReader{
		Reader: kafka.NewReader(cfg),
		topic:  string(topic),
	}

	if latestOnly && groupID == "" {
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

func resetGroupToLatest(ctx context.Context, brokers []string, topic, groupID string) error {
	if len(brokers) == 0 {
		return fmt.Errorf("cannot reset offset with 0 brokers")
	}

	client := kafka.Client{Addr: kafka.TCP(brokers[0])}

	meta, err := client.Metadata(ctx, &kafka.MetadataRequest{Topics: []string{topic}})
	if err != nil {
		return fmt.Errorf("metadata request: %w", err)
	}
	if len(meta.Topics) == 0 {
		return fmt.Errorf("topic %q not found", topic)
	}
	partitions := meta.Topics[0].Partitions
	if len(partitions) == 0 {
		return fmt.Errorf("topic %q has no partitions", topic)
	}

	offsetsMap, err := getLastOffsetsFromPartitions(ctx, partitions, brokers, topic)
	if err != nil {
		return err
	}

	log.Info("new group")
	consumerGroup, err := kafka.NewConsumerGroup(kafka.ConsumerGroupConfig{ID: groupID, Brokers: brokers, Topics: []string{topic}})
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	log.Info("next gen")
	gen, err := consumerGroup.Next(ctx)
	if err != nil {
		return fmt.Errorf("failed to get next Generation: %w", err)
	}
	log.Info("commiting offsets")
	err = gen.CommitOffsets(offsetsMap)
	if err != nil {
		return fmt.Errorf("failed to commit offsets: %w", err)
	}

	// req := &kafka.OffsetCommitRequest{
	// 	GroupID:      groupID,
	// 	GenerationID: -1,
	// 	MemberID:     "",
	// 	Topics:       offsetsMap,
	// }
	//
	// resp, err := client.OffsetCommit(ctx, req)
	// if err != nil {
	// 	return fmt.Errorf("offset commit failed: %w", err)
	// }
	// for t, parts := range resp.Topics {
	// 	for _, p := range parts {
	// 		if p.Error != nil {
	// 			return fmt.Errorf("commit error topic=%s partition=%d: %v", t, p.Partition, p.Error)
	// 		}
	// 	}
	// }

	log.WithField(infologger.Level, infologger.IL_Devel).Debugf("Committed group=%s offsets to latest for topic=%s", groupID, topic)
	return nil
}

// func getLastOffsetsFromPartitions(ctx context.Context, partitions []kafka.Partition, brokers []string, topic string) (map[string][]kafka.OffsetCommit, error) {
func getLastOffsetsFromPartitions(ctx context.Context, partitions []kafka.Partition, brokers []string, topic string) (map[string]map[int]int64, error) {
	// topicsMap := map[string][]kafka.OffsetCommit{}
	topicsMap := map[string]map[int]int64{}
	topicsMap[topic] = map[int]int64{}
	for _, p := range partitions {
		topicsMap[topic][int(p.ID)] = kafka.LastOffset

		// topicsMap[topic] = append(topicsMap[topic], kafka.OffsetCommit{
		// 	Partition: int(p.ID),
		// 	Offset:    kafka.LastOffset,
		// 	Metadata:  "reset",
		// })

		// partition := int(p.ID)
		//
		// conn, err := kafka.DialLeader(ctx, "tcp", brokers[0], topic, partition)
		// if err != nil {
		// 	return nil, fmt.Errorf("dial leader partition %d: %w", partition, err)
		// }
		// _ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		//
		// last, err := conn.ReadLastOffset()
		// conn.Close()
		// if err != nil {
		// 	return nil, fmt.Errorf("read last offset partition %d: %w", partition, err)
		// }
		//
		// topicsMap[topic] = append(topicsMap[topic], kafka.OffsetCommit{
		// 	Partition: partition,
		// 	Offset:    last,
		// 	Metadata:  "reset",
		// })
		//
		// log.WithField(infologger.Level, infologger.IL_Devel).Debugf("last offset (%d) from partition %d:", last, partition)
	}
	return topicsMap, nil
}
