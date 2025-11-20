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
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	cmnevent "github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	pb "github.com/AliceO2Group/Control/common/protos"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/integration"
	lhcevent "github.com/AliceO2Group/Control/core/integration/lhc/event"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logger.New(logrus.StandardLogger(), "lhcclient")
var dipClientTopic topic.Topic = "dip.lhc.beam_mode"

// Plugin implements integration.Plugin and listens for LHC updates.
type Plugin struct {
	endpoint     string
	ctx          context.Context
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

	p.reader = cmnevent.NewReaderWithTopic(dipClientTopic, "o2-aliecs-core.lhc")
	if p.reader == nil {
		return errors.New("could not create a kafka reader for LHC plugin")
	}

	// Always perform a short pre-drain to consume any backlog without injecting.
	log.WithField(infologger.Level, infologger.IL_Devel).
		Info("LHC plugin: draining any initial backlog")
	p.drainBacklog(2 * time.Second)

	// If state is still empty, try reading the latest message once.
	p.mu.Lock()
	empty := p.currentState == nil || p.currentState.BeamMode == pb.BeamMode_UNKNOWN
	p.mu.Unlock()
	if empty {
		if last, err := p.reader.Last(p.ctx); err != nil {
			log.WithField(infologger.Level, infologger.IL_Support).WithError(err).Warn("failed to read last LHC state on init")
		} else if last != nil {
			if bmEvt := last.GetBeamModeEvent(); bmEvt != nil && bmEvt.GetBeamInfo() != nil {
				p.mu.Lock()
				p.currentState = bmEvt.GetBeamInfo()
				p.mu.Unlock()
			}
		} else {
			// nothing to retrieve in the topic, we move on
		}
	}

	go p.readAndInjectLhcUpdates()
	log.WithField(infologger.Level, infologger.IL_Devel).Debug("LHC plugin initialized (client started)")
	return nil
}

// drainBacklog reads messages for a limited time and only updates the plugin state, without injecting to env manager.
func (p *Plugin) drainBacklog(timeout time.Duration) {
	drainCtx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()
	for {
		msg, err := p.reader.Next(drainCtx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				break
			}
			// transient error: small sleep and continue until timeout
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if msg == nil {
			continue
		}
		if beamModeEvent := msg.GetBeamModeEvent(); beamModeEvent != nil && beamModeEvent.GetBeamInfo() != nil {
			beamInfo := beamModeEvent.GetBeamInfo()
			log.WithField(infologger.Level, infologger.IL_Devel).
				Debugf("new LHC update received while draining backlog: BeamMode=%s, FillNumber=%d, FillingScheme=%s, StableBeamsStart=%d, StableBeamsEnd=%d, BeamType=%s",
					beamInfo.GetBeamMode().String(), beamInfo.GetFillNumber(), beamInfo.GetFillingSchemeName(),
					beamInfo.GetStableBeamsStart(), beamInfo.GetStableBeamsEnd(), beamInfo.GetBeamType())

			p.mu.Lock()
			p.currentState = beamModeEvent.GetBeamInfo()
			p.mu.Unlock()
		}
	}
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
func (p *Plugin) CallStack(data interface{}) (stack map[string]interface{}) {
	call, ok := data.(*callable.Call)
	if !ok {
		return
	}

	stack = make(map[string]interface{})
	stack["UpdateFillInfo"] = func() (out string) {
		p.updateFillInfo(call)
		return
	}
	return
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

// UpdateFillInfo: propagate latest LHC fill info into the environment's global runtime vars
func (p *Plugin) updateFillInfo(call *callable.Call) (out string) {
	varStack := call.VarStack
	envId, ok := varStack["environment_id"]
	if !ok {
		err := errors.New("cannot acquire environment ID")
		log.Error(err)

		call.VarStack["__call_error_reason"] = err.Error()
		call.VarStack["__call_error"] = "LHC plugin Call Stack failed"
		return
	}

	log := log.WithFields(logrus.Fields{
		"partition": envId,
		"call":      "UpdateFillInfo",
	})

	parentRole, ok := call.GetParentRole().(callable.ParentRole)
	if !ok || parentRole == nil {
		log.WithField(infologger.Level, infologger.IL_Support).
			Error("cannot access parent role to propagate LHC fill info")
		return
	}

	if p.currentState == nil {
		log.WithField(infologger.Level, infologger.IL_Support).
			Warn("attempted to update environment with fill info, but fill info is not available in plugin")
		return
	}

	// note: the following was causing very weird behaviours, which could be attributed to memory corruption.
	//  I did not manage to understand why can't we safely clone such a proto message.
	// state := proto.Clone(p.currentState).(*pb.BeamInfo)

	p.mu.Lock()
	defer p.mu.Unlock()
	state := p.currentState

	parentRole.SetGlobalRuntimeVar("fill_info_beam_mode", state.BeamMode.String())

	// If NO_BEAM, clear all other fill info and return
	if state.BeamMode == pb.BeamMode_NO_BEAM {
		parentRole.DeleteGlobalRuntimeVar("fill_info_fill_number")
		parentRole.DeleteGlobalRuntimeVar("fill_info_filling_scheme")
		parentRole.DeleteGlobalRuntimeVar("fill_info_beam_type")
		parentRole.DeleteGlobalRuntimeVar("fill_info_stable_beams_start_ms")
		parentRole.DeleteGlobalRuntimeVar("fill_info_stable_beams_end_ms")

		log.WithField(infologger.Level, infologger.IL_Devel).
			Debug("NO_BEAM — cleared fill info vars and set beam mode only")
		return
	}

	// Otherwise, propagate latest known info
	parentRole.SetGlobalRuntimeVar("fill_info_fill_number", strconv.FormatInt(int64(state.FillNumber), 10))
	parentRole.SetGlobalRuntimeVar("fill_info_filling_scheme", state.FillingSchemeName)
	parentRole.SetGlobalRuntimeVar("fill_info_beam_type", state.BeamType)
	if state.StableBeamsStart > 0 {
		parentRole.SetGlobalRuntimeVar("fill_info_stable_beams_start_ms", strconv.FormatInt(state.StableBeamsStart, 10))
	} else {
		parentRole.DeleteGlobalRuntimeVar("fill_info_stable_beams_start_ms")
	}
	if state.StableBeamsEnd > 0 {
		parentRole.SetGlobalRuntimeVar("fill_info_stable_beams_end_ms", strconv.FormatInt(state.StableBeamsEnd, 10))
	} else {
		parentRole.DeleteGlobalRuntimeVar("fill_info_stable_beams_end_ms")
	}

	log.WithField("fillNumber", state.FillNumber).
		WithField("fillingScheme", state.FillingSchemeName).
		WithField("beamType", state.BeamType).
		WithField("beamMode", state.BeamMode).
		WithField("stableStartMs", state.StableBeamsStart).
		WithField("stableEndMs", state.StableBeamsEnd).
		WithField(infologger.Level, infologger.IL_Devel).
		Debug("updated environment fill info from latest snapshot")
	return
}
