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
	pb "github.com/AliceO2Group/Control/common/protos"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("Reader", func() {
	When("converting kafka message to event", func() {
		It("unmarshals protobuf payload correctly", func() {
			e := &pb.Event{Payload: &pb.Event_CoreStartEvent{CoreStartEvent: &pb.Ev_MetaEvent_CoreStart{FrameworkId: "z"}}}
			b, err := proto.Marshal(e)
			Expect(err).To(BeNil())

			m := kafka.Message{Value: b}
			evt, err := kafkaMessageToEvent(m)
			Expect(err).To(BeNil())
			Expect(evt.GetCoreStartEvent().FrameworkId).To(Equal("z"))
		})
	})
})
