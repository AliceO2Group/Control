/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2025 CERN and copyright holders of ALICE O².
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
	"sync"

	pb "github.com/AliceO2Group/Control/common/protos"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("Writer", func() {
	When("event is written into writer", func() {
		It("transforms it to kafka message and sends it", func() {
			channel := make(chan struct{})
			writer := KafkaWriter{}
			writer.toBatchMessagesChan = make(chan kafka.Message, 100)
			writer.messageBuffer = NewFifoBuffer[kafka.Message]()
			writer.Writer = &kafka.Writer{}
			writer.Topic = "testtopic"
			writer.runningWorkers = sync.WaitGroup{}
			writer.batchingLoopDoneCh = make(chan struct{}, 1)

			writer.writeFunction = func(messages []kafka.Message) {
				Expect(len(messages)).To(Equal(1))
				event := &pb.Event{}
				err := proto.Unmarshal(messages[0].Value, event)
				Expect(err).To(BeNil())
				Expect(event.GetCoreStartEvent().FrameworkId).To(Equal("FrameworkId"))
				channel <- struct{}{}
			}

			go writer.writingLoop()
			go writer.batchingLoop()

			event := &pb.Ev_MetaEvent_CoreStart{FrameworkId: "FrameworkId"}

			writer.WriteEvent(event)
			<-channel
			writer.Close()
		})
	})
})
