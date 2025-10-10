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

package lhc

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	pb "github.com/AliceO2Group/Control/common/protos"
	"io"
	"strings"
	"sync"
	"time"

	cmnevent "github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/integration"
	lhcevent "github.com/AliceO2Group/Control/core/integration/lhc/event"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logger.New(logrus.StandardLogger(), "lhcclient")
var dipClientTopic topic.Topic = "dip.lhc.beam_mode"

// Plugin implements integration.Plugin and listens for LHC updates.
type Plugin struct {
	endpoint string
	ctx      context.Context
	//cancel       context.CancelFunc
	//wg           sync.WaitGroup
	mu           sync.Mutex
	currentState *pb.BeamInfo
	reader       cmnevent.Reader
}

func NewPlugin(endpoint string) integration.Plugin {

	return &Plugin{endpoint: endpoint, mu: sync.Mutex{}, currentState: &pb.BeamInfo{BeamMode: pb.BeamMode_UNKNOWN}}
}

func (p *Plugin) Init(_ string) error {

	// use a background context for reader loop; Destroy will Close the reader
	p.ctx = context.Background()

	p.reader = cmnevent.NewReaderWithTopic(dipClientTopic, "", true)

	if p.reader == nil {
		return errors.New("could not create a kafka reader for LHC plugin")
	}
	go p.readAndInjectLhcUpdates()

	log.Debug("LHC plugin initialized (client started)")
	return nil
}

func (p *Plugin) GetName() string       { return "lhc" }
func (p *Plugin) GetPrettyName() string { return "LHC (DIP/Kafka client)" }
func (p *Plugin) GetEndpoint() string {
	return strings.Join(viper.GetStringSlice("kafkaEndpoints"), ",")
}

func (p *Plugin) GetConnectionState() string {
	if p == nil || p.reader == nil {
		return "UNKNOWN"
	}
	return "READY" // Unfortunately, kafka.Reader does not provide any GetStatus method
}

func (p *Plugin) GetData(_ []any) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.currentState == nil {
		return ""
	}

	outMap := make(map[string]interface{})
	outMap["BeamMode"] = p.currentState.BeamMode.String()
	outMap["BeamType"] = p.currentState.BeamType
	outMap["FillingSchemeName"] = p.currentState.FillingSchemeName
	outMap["FillNumber"] = p.currentState.FillNumber
	outMap["StableBeamsEnd"] = p.currentState.StableBeamsEnd
	outMap["StableBeamsStart"] = p.currentState.StableBeamsStart

	b, _ := json.Marshal(outMap)
	return string(b)
}

func (p *Plugin) GetEnvironmentsData(envIds []uid.ID) map[uid.ID]string {
	// there is nothing sensible we could provide here, LHC client is not environment-specific
	return nil
}

func (p *Plugin) GetEnvironmentsShortData(envIds []uid.ID) map[uid.ID]string {
	return p.GetEnvironmentsData(envIds)
}

func (p *Plugin) ObjectStack(_ map[string]string, _ map[string]string) (stack map[string]interface{}) {
	return make(map[string]interface{})
}
func (p *Plugin) CallStack(_ interface{}) (stack map[string]interface{}) {
	return make(map[string]interface{})
}

func (p *Plugin) Destroy() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.reader != nil {
		err := p.reader.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Plugin) readAndInjectLhcUpdates() {
	for {
		msg, err := p.reader.Next(p.ctx)
		if errors.Is(err, io.EOF) {
			log.WithField(infologger.Level, infologger.IL_Support).
				Debug("received an EOF from Kafka reader, likely cancellation was requested, breaking")
			break
		}
		if err != nil {
			log.WithField(infologger.Level, infologger.IL_Support).
				WithError(err).
				Error("error while reading from Kafka")
			// in case of errors, we throttle the loop to mitigate the risk a log spam if error persists
			time.Sleep(time.Second * 1)
			continue
		}
		if msg == nil {
			log.WithField(infologger.Level, infologger.IL_Devel).
				Warn("received an empty message with no error. it's unexpected, but continuing")
			continue
		}

		if bmEvt := msg.GetBeamModeEvent(); bmEvt != nil && bmEvt.GetBeamInfo() != nil {
			beamInfo := bmEvt.GetBeamInfo()
			log.WithField(infologger.Level, infologger.IL_Devel).
				Debugf("new LHC update received: BeamMode=%s, FillNumber=%d, FillingScheme=%s, StableBeamsStart=%d, StableBeamsEnd=%d, BeamType=%s",
					beamInfo.GetBeamMode().String(), beamInfo.GetFillNumber(), beamInfo.GetFillingSchemeName(),
					beamInfo.GetStableBeamsStart(), beamInfo.GetStableBeamsEnd(), beamInfo.GetBeamType())
			// update plugin state
			p.mu.Lock()
			p.currentState = beamInfo
			p.mu.Unlock()

			// convert to internal LHC event and notify environment manager
			go func(beamInfo *pb.BeamInfo) {
				envMan := environment.ManagerInstance()

				ev := &lhcevent.LhcStateChangeEvent{
					IntegratedServiceEventBase: cmnevent.IntegratedServiceEventBase{ServiceName: "LHC"},
					BeamInfo: lhcevent.BeamInfo{
						BeamMode:          beamInfo.GetBeamMode(),
						StableBeamsStart:  beamInfo.GetStableBeamsStart(),
						StableBeamsEnd:    beamInfo.GetStableBeamsEnd(),
						FillNumber:        beamInfo.GetFillNumber(),
						FillingSchemeName: beamInfo.GetFillingSchemeName(),
						BeamType:          beamInfo.GetBeamType(),
					},
				}
				envMan.NotifyIntegratedServiceEvent(ev)
			}(beamInfo)
		}
	}
}
