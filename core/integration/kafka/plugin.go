/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2022 CERN and copyright holders of ALICE O².
 * Author: Piotr Konopka <piotr.jan.konopka@cern.ch>
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

//go:generate protoc --go_out=. protos/kafka.proto

package kafka

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	kafkapb "github.com/AliceO2Group/Control/core/integration/kafka/protos"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/golang/protobuf/proto"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logger.New(logrus.StandardLogger(), "kafkaclient")

type Plugin struct {
	endpoint      string
	kafkaWriter   *kafka.Writer
	envsInRunning map[string]*kafkapb.EnvInfo // env id is the key
}

func NewPlugin(endpoint string) integration.Plugin {
	_, err := url.Parse(endpoint)
	if err != nil {
		log.WithField("endpoint", endpoint).
			WithError(err).
			Error("bad service endpoint")
		return nil
	}

	return &Plugin{
		endpoint:    endpoint,
		kafkaWriter: nil,
	}
}

func (p *Plugin) GetName() string {
	return "kafka"
}

func (p *Plugin) GetPrettyName() string {
	return "Kafka FSM transition information service"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("kafkaEndpoint")
}

func (p *Plugin) GetConnectionState() string {
	if p == nil || p.kafkaWriter == nil {
		return "UNKNOWN"
	} else {
		return "READY"
	}
}

func (p *Plugin) GetData(environmentIds []uid.ID) string {
	return ""
}

func (p *Plugin) FSMEnterStateTopic(state string) string {
	return "aliecs.env_state." + state
}

func (p *Plugin) FSMLeaveStateTopic(state string) string {
	return "aliecs.env_leave_state." + state
}

func (p *Plugin) ActiveRunsListTopic() string {
	return "aliecs.env_list.RUNNING"
}

func (p *Plugin) LoggerCallback(msg string, a ...interface{}) {
	log.Infof(msg, a...)
}

func (p *Plugin) ErrorLoggerCallback(msg string, a ...interface{}) {
	log.Errorf(msg, a...)
}

func (p *Plugin) Init(_ string) error {
	const call = "Init"
	var err error

	p.kafkaWriter = &kafka.Writer{
		Addr:                   kafka.TCP(p.endpoint),
		Balancer:               &kafka.CRC32Balancer{}, // same behaviour as confluent-kafka client
		AllowAutoTopicCreation: true,
		Logger:                 kafka.LoggerFunc(p.LoggerCallback),
		ErrorLogger:            kafka.LoggerFunc(p.ErrorLoggerCallback),
		RequiredAcks:           kafka.RequireAll,
	}

	p.envsInRunning = make(map[string]*kafkapb.EnvInfo)
	log.WithField("call", call).
		Info("successfully created a kafka producer with broker '" + p.endpoint + "'")

	// Prepare and send active run list (expected to be empty during init)
	timestamp := uint64(time.Now().UnixMilli())
	activeRunsList := &kafkapb.ActiveRunsList{
		ActiveRuns: p.GetRunningEnvList(),
		Timestamp:  timestamp,
	}
	arlData, err := proto.Marshal(activeRunsList)
	if err != nil {
		log.WithField("call", call).Error("could not marshall an active runs list: ", err)
	}
	p.produceMessage(arlData, p.ActiveRunsListTopic(), "", "Init")
	return nil
}

func (p *Plugin) ObjectStack(_ map[string]string) (stack map[string]interface{}) {
	stack = make(map[string]interface{})
	return stack
}

func parseDetectors(detectorsParam string) (detectors []string, err error) {
	detectorsSlice := make([]string, 0)
	bytes := []byte(detectorsParam)
	err = json.Unmarshal(bytes, &detectorsSlice)
	if err != nil {
		log.WithError(err).
			Error("error processing the detectors list")
		return
	}
	return detectorsSlice, nil
}

func (p *Plugin) extractStateFromTrigger(trigger string) string {
	if strings.Contains(trigger, "enter_") {
		return strings.TrimPrefix(trigger, "enter_")
	} else if strings.Contains(trigger, "leave_") {
		return strings.TrimPrefix(trigger, "leave_")
	} else if trigger == "after_START_ACTIVITY" || trigger == "before_STOP_ACTIVITY" {
		return "RUNNING"
	} else {
		return "UNKNOWN"
	}
}

func (p *Plugin) newEnvStateObject(varStack map[string]string, call string) *kafkapb.EnvInfo {
	envId, ok := varStack["environment_id"]
	if !ok {
		log.WithField("call", call).Error("cannot acquire environment ID")
		return nil
	}

	trigger, ok := varStack["__call_trigger"]
	if !ok {
		log.WithField("call", call).WithField("partition", envId).
			Error("cannot acquire trigger from varStack")
		return nil
	}
	state := p.extractStateFromTrigger(trigger)

	var runNumberOpt *uint32 = nil
	var runTypeOpt *string = nil
	if state == "RUNNING" {
		// the following fields are relevant only in RUNNING state
		runNumberStr, ok := varStack["run_number"]
		if ok {
			runNumber, err := strconv.ParseUint(runNumberStr, 10, 32)
			if err == nil {
				runNumber32 := uint32(runNumber)
				runNumberOpt = &runNumber32
			}
		}

		runType, ok := varStack["run_type"]
		if ok {
			runTypeOpt = &runType
		}
	}

	enterStateTimeMsStr, ok := varStack["enter_state_time_ms"]
	if !ok {
		log.WithField("call", call).
			WithField("partition", envId).
			Error("cannot acquire enter_state_time_ms")
		return nil
	}
	enterStateTimeMs, err := strconv.ParseUint(enterStateTimeMsStr, 10, 64)
	if err != nil {
		log.WithError(err).
			WithField("call", call).
			WithField("partition", envId).
			Errorf("cannot convert enter_state_time_ms (%s) to an unsigned integer", enterStateTimeMsStr)
		return nil
	}

	detectorsStr, ok := varStack["detectors"]
	if !ok {
		log.WithField("call", call).
			WithField("partition", envId).
			Error("cannot acquire general detector list from varStack")
		return nil
	}
	detectorsSlice, err := parseDetectors(detectorsStr)
	if err != nil {
		log.WithField("call", call).
			WithField("partition", envId).
			Error("cannot parse general detector list")
		return nil
	}

	return &kafkapb.EnvInfo{
		EnvironmentId:       envId,
		RunNumber:           runNumberOpt,
		RunType:             runTypeOpt,
		State:               state,
		Detectors:           detectorsSlice,
		EnterStateTimestamp: enterStateTimeMs,
	}
}

func (p *Plugin) UpdateRunningEnvList(envInfo *kafkapb.EnvInfo) {
	if envInfo.State == "RUNNING" {
		p.envsInRunning[envInfo.EnvironmentId] = envInfo
	} else {
		delete(p.envsInRunning, envInfo.EnvironmentId)
	}
}

func (p *Plugin) GetRunningEnvList() []*kafkapb.EnvInfo {
	var array []*kafkapb.EnvInfo
	for _, v := range p.envsInRunning {
		array = append(array, v)
	}
	return array
}

func (p *Plugin) produceMessage(message []byte, topic string, envId string, call string) {
	log.WithField("call", call).
		WithField("partition", envId).
		Debugf("producing a new kafka message on topic %s", topic)

	err := p.kafkaWriter.WriteMessages(context.Background(), kafka.Message{
		Topic: topic,
		Value: message,
	})

	if err != nil {
		log.WithField("call", call).
			WithField("partition", envId).
			WithField("topic", topic).
			Errorf("Kafka message delivery failed: %s", err.Error())
	}
}

func (p *Plugin) createNewStateCallback(varStack map[string]string, call string) func() string {
	return func() (out string) {
		// Retrieve and update the env info
		timestamp := uint64(time.Now().UnixMilli())
		envInfo := p.newEnvStateObject(varStack, call)
		p.UpdateRunningEnvList(envInfo)

		// Prepare and send new state notification
		log.WithField("call", call).
			WithField("partition", envInfo.EnvironmentId).
			Debug("Notifying Kafka that the environment is entering the state " + envInfo.State)
		newStateNotification := &kafkapb.NewStateNotification{
			EnvInfo:   envInfo,
			Timestamp: timestamp,
		}
		nsnData, err := proto.Marshal(newStateNotification)
		if err != nil {
			log.WithField("call", call).
				WithField("partition", envInfo.EnvironmentId).
				Error("could not marshall a new state notification: ", err)
		}
		p.produceMessage(nsnData, p.FSMEnterStateTopic(envInfo.State), envInfo.EnvironmentId, call)

		log.WithField("call", call).
			WithField("partition", envInfo.EnvironmentId).
			Debug("Notifying Kafka about the new list of environments in RUNNING")
		// Prepare and send active run list
		activeRunsList := &kafkapb.ActiveRunsList{
			ActiveRuns: p.GetRunningEnvList(),
			Timestamp:  timestamp,
		}
		arlData, err := proto.Marshal(activeRunsList)
		if err != nil {
			log.WithField("call", call).
				WithField("partition", envInfo.EnvironmentId).
				Error("could not marshall an active runs list: ", err)
		}
		p.produceMessage(arlData, p.ActiveRunsListTopic(), envInfo.EnvironmentId, call)
		return
	}
}

func (p *Plugin) createLeaveStateCallback(varStack map[string]string, call string) func() string {
	return func() (out string) {
		// Retrieve and update the env info
		timestamp := uint64(time.Now().UnixMilli())
		envInfo := p.newEnvStateObject(varStack, call)

		// Prepare and send new state notification
		log.WithField("call", call).
			WithField("partition", envInfo.EnvironmentId).
			Debug("Notifying Kafka that the environment is leaving the state " + envInfo.State)
		newStateNotification := &kafkapb.NewStateNotification{
			EnvInfo:   envInfo,
			Timestamp: timestamp,
		}
		nsnData, err := proto.Marshal(newStateNotification)
		if err != nil {
			log.WithField("call", call).
				WithField("partition", envInfo.EnvironmentId).
				Error("could not marshall a new state notification: ", err)
		}
		p.produceMessage(nsnData, p.FSMLeaveStateTopic(envInfo.State), envInfo.EnvironmentId, call)

		return
	}
}

func (p *Plugin) CallStack(data interface{}) (stack map[string]interface{}) {
	call, ok := data.(*callable.Call)
	if !ok {
		return
	}
	varStack := call.VarStack

	stack = make(map[string]interface{})
	// The first two are actually equal. We create both to allow to migrate to a more accurate name.
	stack["PublishUpdate"] = p.createNewStateCallback(varStack, "PublishUpdate")
	stack["PublishEnterStateUpdate"] = p.createNewStateCallback(varStack, "PublishEnterStateUpdate")
	stack["PublishLeaveStateUpdate"] = p.createLeaveStateCallback(varStack, "PublishLeaveStateUpdate")
	return
}

func (p *Plugin) Destroy() error {
	if err := p.kafkaWriter.Close(); err != nil {
		log.Fatal("failed to close Kafka writer:", err)
	}
	return nil
}
