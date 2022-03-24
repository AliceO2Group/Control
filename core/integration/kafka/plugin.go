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
	"encoding/json"
	"fmt"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	kafkapb "github.com/AliceO2Group/Control/core/integration/kafka/protos"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/url"
	"strconv"
	"time"
)

type Plugin struct {
	kafkaBroker   string
	kafkaProducer *kafka.Producer
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
		kafkaBroker:   endpoint,
		kafkaProducer: nil,
	}
}

func (p *Plugin) GetName() string {
	return "kafka"
}

func (p *Plugin) GetPrettyName() string {
	return "Kafka FSM transition information service"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("kafkaBroker")
}

func (p *Plugin) GetConnectionState() string {
	if p == nil || p.kafkaProducer == nil {
		return "UNKNOWN"
	} else {
		return "READY"
	}
}

func (p *Plugin) GetData(environmentIds []uid.ID) string {
	return ""
}

func (p *Plugin) FSMTransitionTopic(state string) string {
	return "FSM_" + state
}

func (p *Plugin) ActiveRunsListTopic() string {
	return "ACTIVE_RUNS"
}

func (p *Plugin) Init(instanceId string) error {
	var err error
	p.kafkaProducer, err = kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": p.kafkaBroker})
	if err != nil {
		return fmt.Errorf("failed to initialize a kafka producer with broker '%s'. Details: %s", p.kafkaBroker, err.Error())
	}
	p.envsInRunning = make(map[string]*kafkapb.EnvInfo)
	log.Info("Successfully created a kafka producer with broker '" + p.kafkaBroker + "'")
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

func (p *Plugin) NewEnvStateObject(varStack map[string]string, state string) *kafkapb.EnvInfo {
	envId, ok := varStack["environment_id"]
	if !ok {
		log.Error("cannot acquire environment ID")
		return nil
	}

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

	detectorsStr, ok := varStack["detectors"]
	if !ok {
		log.WithField("partition", envId).
			Error("cannot acquire general detector list from varStack")
	}
	detectorsSlice, err := parseDetectors(detectorsStr)
	if err != nil {
		log.WithField("partition", envId).
			Error("cannot parse general detector list")
		return nil
	}

	return &kafkapb.EnvInfo{
		EnvironmentId: envId,
		RunNumber:     runNumberOpt,
		RunType:       runTypeOpt,
		State:         state,
		Detectors:     detectorsSlice,
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

func (p *Plugin) ProduceMessage(message []byte, topic string, envId string) {
	log.WithField("partition", envId).
		Debug("Producing a new kafka message on topic ", topic)

	deliveryChannel := make(chan kafka.Event)
	err := p.kafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, deliveryChannel)

	if err != nil {
		log.WithField("partition", envId).
			Error("Kafka message delivery failed: ", err.Error())
	}

	e := <-deliveryChannel
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		log.WithField("partition", envId).Error("Kafka message delivery failed: ", m.TopicPartition.Error)
	} else {
		log.WithField("partition", envId).Debugf("Kafka message delivered message to topic %s [%d] at offset %v\n",
			*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
	}
}

func (p *Plugin) CreateUpdateCallback(varStack map[string]string, state string) func() string {
	return func() (out string) {
		// Retrieve and update the env info
		timestamp := uint64(time.Now().UnixNano() / 1000000)
		envInfo := p.NewEnvStateObject(varStack, state)
		p.UpdateRunningEnvList(envInfo)

		// Prepare and send new state notification
		log.WithField("partition", envInfo.EnvironmentId).
			Debug("Advertising the environment state (" + state + ") and active runs list to Kafka")
		newStateNotification := &kafkapb.NewStateNotification{
			EnvInfo:   envInfo,
			Timestamp: timestamp,
		}
		nsnData, err := proto.Marshal(newStateNotification)
		if err != nil {
			log.WithField("partition", envInfo.EnvironmentId).
				Error("Could not marshall a new state notification: ", err)
		}
		p.ProduceMessage(nsnData, p.FSMTransitionTopic(state), envInfo.EnvironmentId)

		// Prepare and send active run list
		activeRunsList := &kafkapb.ActiveRunsList{
			ActiveRuns: p.GetRunningEnvList(),
			Timestamp:  timestamp,
		}
		arlData, err := proto.Marshal(activeRunsList)
		if err != nil {
			log.WithField("partition", envInfo.EnvironmentId).
				Error("Could not marshall an active runs list: ", err)
		}
		p.ProduceMessage(arlData, p.ActiveRunsListTopic(), envInfo.EnvironmentId)
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
	stack["Standby"] = p.CreateUpdateCallback(varStack, "STANDBY")
	stack["Deployed"] = p.CreateUpdateCallback(varStack, "DEPLOYED")
	stack["Configured"] = p.CreateUpdateCallback(varStack, "CONFIGURED")
	stack["Running"] = p.CreateUpdateCallback(varStack, "RUNNING")
	stack["Done"] = p.CreateUpdateCallback(varStack, "DONE")
	stack["Error"] = p.CreateUpdateCallback(varStack, "ERROR")
	return
}

func (p *Plugin) Destroy() error {
	p.kafkaProducer.Close()
	return nil
}
