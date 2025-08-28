/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021-2024 CERN and copyright holders of ALICE O².
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

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/ddsched.proto

// Package ddsched provides integration with the Data Distribution (DD) scheduler
// for managing the pool of FLPs participating in data taking operations.
package ddsched

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/monitoring"
	pb "github.com/AliceO2Group/Control/common/protos"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/integration"
	ddpb "github.com/AliceO2Group/Control/core/integration/ddsched/protos"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const (
	DDSCHED_DIAL_TIMEOUT       = 2 * time.Second
	DDSCHED_INITIALIZE_TIMEOUT = 30 * time.Second
	DDSCHED_TERMINATE_TIMEOUT  = 30 * time.Second
	TOPIC                      = topic.IntegratedService + topic.Separator + "ddsched"
)

type Plugin struct {
	ddSchedulerHost string
	ddSchedulerPort int

	ddSchedClient *RpcClient
}

func NewPlugin(endpoint string) integration.Plugin {
	u, err := url.Parse(endpoint)
	if err != nil {
		log.WithField("endpoint", endpoint).
			WithError(err).
			Error("bad service endpoint")
		return nil
	}

	portNumber, _ := strconv.Atoi(u.Port())

	return &Plugin{
		ddSchedulerHost: u.Hostname(),
		ddSchedulerPort: portNumber,
		ddSchedClient:   nil,
	}
}

func (p *Plugin) GetName() string {
	return "ddsched"
}

func (p *Plugin) GetPrettyName() string {
	return "DD (EPN DataDistribution TfScheduler)"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("ddSchedulerEndpoint")
}

func (p *Plugin) GetConnectionState() string {
	if p == nil || p.ddSchedClient == nil {
		return "UNKNOWN"
	}
	return p.ddSchedClient.conn.GetState().String()
}

func (p *Plugin) GetData(_ []any) string {
	if p == nil || p.ddSchedClient == nil {
		return ""
	}

	partitionStates := make(map[string]string)
	environmentIds := environment.ManagerInstance().Ids()

	partitionInfos := p.partitionStatesForEnvs(environmentIds)

	partitionStates = make(map[string]string)
	for envId, partitionInfo := range partitionInfos {
		state, ok := partitionInfo["state"]
		if !ok {
			continue
		}
		partitionStates[envId.String()] = state
	}

	out, err := json.Marshal(partitionStates)
	if err != nil {
		return ""
	}
	return string(out[:])
}

func (p *Plugin) GetEnvironmentsData(envIds []uid.ID) map[uid.ID]string {
	if p == nil || p.ddSchedClient == nil {
		return nil
	}

	partitionInfos := p.partitionStatesForEnvs(envIds)

	partitionInfosOut := make(map[uid.ID]string)

	for envId, partitionInfo := range partitionInfos {
		partitionInfoOut, err := json.Marshal(partitionInfo)
		if err != nil {
			continue
		}
		partitionInfosOut[envId] = string(partitionInfoOut[:])
	}

	return partitionInfosOut
}

func (p *Plugin) GetEnvironmentsShortData(envIds []uid.ID) map[uid.ID]string {
	return p.GetEnvironmentsData(envIds)
}

func (p *Plugin) partitionStatesForEnvs(envIds []uid.ID) map[uid.ID]map[string]string {
	partitionStates := make(map[uid.ID]map[string]string)

	for _, envId := range envIds {
		in := ddpb.PartitionInfo{
			PartitionId:   envId.String(),
			EnvironmentId: envId.String(),
		}
		ctx, cancel := context.WithTimeout(context.Background(), viper.GetDuration("ddschedStatusTimeout"))
		ctx = monitoring.AddEnvAndRunType(ctx, envId.String(), "none")
		state, err := p.ddSchedClient.PartitionStatus(ctx, &in, grpc.EmptyCallOption{})
		cancel()
		if err != nil {
			continue
		}
		if state == nil {
			continue
		}
		partitionState, ok := ddpb.PartitionState_name[int32(state.GetPartitionState())]
		if !ok {
			continue
		}
		partitionInfo := map[string]string{
			"state":       partitionState,
			"infoMessage": state.GetInfoMessage(),
		}
		partitionStates[envId] = partitionInfo
	}

	return partitionStates
}

func (p *Plugin) Init(_ string) error {
	if p.ddSchedClient == nil {
		cxt, cancel := context.WithCancel(context.Background())
		p.ddSchedClient = NewClient(cxt, cancel, viper.GetString("ddSchedulerEndpoint"))
		if p.ddSchedClient == nil {
			return fmt.Errorf("failed to connect to DD scheduler service on %s", viper.GetString("ddSchedulerEndpoint"))
		}
		log.Debug("DD scheduler plugin initialized")
	}
	return nil
}

func (p *Plugin) ObjectStack(_ map[string]string, _ map[string]string) (stack map[string]interface{}) {
	stack = make(map[string]interface{})
	return stack
}

func (p *Plugin) CallStack(data interface{}) (stack map[string]interface{}) {
	call, ok := data.(*callable.Call)
	if !ok {
		return
	}
	varStack := call.VarStack
	envId, ok := varStack["environment_id"]
	if !ok {
		log.Error("cannot acquire environment ID")
		return
	}

	stack = make(map[string]interface{})
	stack["PartitionInitialize"] = func() (out string) { // must formally return string even when we return nothing
		log.WithField("partition", envId).
			Debug("performing DD scheduler PartitionInitialize")

		var err error
		callFailedStr := "DDsched PartitionInitialize call failed"

		parentRoleI := call.GetParentRole()
		parentRole, ok := parentRoleI.(workflow.Role)
		if !ok {
			err = errors.New("internal error: cannot acquire parent role")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "PartitionInitialize").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		root := workflow.GetRoot(parentRole)

		stfbHostIdMap := make(map[string]string)
		stfsHostIdMap := make(map[string]string)

		workflow.LeafWalk(root, func(role workflow.Role) {
			roleVS, err := role.ConsolidatedVarStack()
			if err != nil {
				log.WithError(err).
					WithField("partition", envId).
					Error("error processing DD host_id_map")
				return
			}
			var (
				ddDiscoveryIbHostname, ddDiscoveryStfbId, ddDiscoveryStfsId       string
				ddDiscoveryIbHostnameOk, ddDiscoveryStfbIdOk, ddDiscoveryStfsIdOk bool
			)
			ddDiscoveryIbHostname, ddDiscoveryIbHostnameOk = roleVS["dd_discovery_ib_hostname"]
			ddDiscoveryStfbId, ddDiscoveryStfbIdOk = roleVS["dd_discovery_stfb_id"]
			ddDiscoveryStfsId, ddDiscoveryStfsIdOk = roleVS["dd_discovery_stfs_id"]
			if ddDiscoveryIbHostnameOk {
				if ddDiscoveryStfsIdOk {
					stfsHostIdMap[ddDiscoveryStfsId] = ddDiscoveryIbHostname
				} else if ddDiscoveryStfbIdOk {
					stfbHostIdMap[ddDiscoveryStfbId] = ddDiscoveryIbHostname
				} else {
					return
				}
			} else {
				return
			}
		})

		partitionParams := make(map[string]string)

		// FIXME: this only copies over vars prefixed with "ddsched_"
		// Figure out a better way!
		for k, v := range varStack {
			if strings.HasPrefix(k, "ddsched_") && k != "ddsched_enabled" {
				partitionParams[strings.TrimPrefix(k, "ddsched_")] = v
			}
		}

		in := ddpb.PartitionInitRequest{
			PartitionInfo: &ddpb.PartitionInfo{
				EnvironmentId: envId,
				PartitionId:   envId,
			},
			StfbHostIdMap:   stfbHostIdMap,
			StfsHostIdMap:   stfsHostIdMap,
			PartitionParams: partitionParams,
		}

		if p.ddSchedClient == nil {
			err = fmt.Errorf("DD scheduler plugin not initialized, PartitionInitialize impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "PartitionInitialize").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		if p.ddSchedClient.GetConnState() == connectivity.Shutdown {
			err = fmt.Errorf("DD scheduler client connection not available, PartitionInitialize impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "PartitionInitialize").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		var response *ddpb.PartitionResponse
		timeout := callable.AcquireTimeout(DDSCHED_INITIALIZE_TIMEOUT, varStack, "Initialize", envId)
		ctx, cancel := integration.NewContext(envId, varStack, timeout)
		defer cancel()

		payload := map[string]interface{}{
			"ddschedRequest": &in,
		}
		payloadJson, _ := json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_STARTED,
			OperationStep:       "perform DD scheduler call: PartitionInitialize",
			OperationStepStatus: pb.OpStatus_STARTED,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		response, err = p.ddSchedClient.PartitionInitialize(ctx, &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("call", "PartitionInitialize").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "perform DD scheduler call: PartitionInitialize",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               err.Error(),
			})

			return
		}
		if response.PartitionState != ddpb.PartitionState_PARTITION_CONFIGURING &&
			response.PartitionState != ddpb.PartitionState_PARTITION_CONFIGURED {
			err = fmt.Errorf("PartitionInitialize returned unexpected state %s (expected: PARTITION_CONFIGURING)", response.PartitionState.String())

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("call", "PartitionInitialize").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			payload["ddschedResponse"] = &response
			payloadJson, _ = json.Marshal(payload)
			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "perform DD scheduler call: PartitionInitialize",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               err.Error(),
			})

			return
		}

		payload["ddschedResponse"] = &response
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_ONGOING,
			OperationStep:       "perform DD scheduler call: PartitionInitialize",
			OperationStepStatus: pb.OpStatus_DONE_OK,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		payload = map[string]interface{}{
			"lastKnownState": response.PartitionState.String(),
		}
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_ONGOING,
			OperationStep:       "poll for DD partition state",
			OperationStepStatus: pb.OpStatus_STARTED,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

	PARTITION_STATE_POLLING:
		for ctx.Err() == nil {
			response, err = p.ddSchedClient.PartitionStatus(ctx, in.PartitionInfo, grpc.EmptyCallOption{})
			if err != nil {
				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("partition", envId).
					WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
					WithField("call", "PartitionStatus").
					WithField("timeout", timeout.String()).
					Error("DDsched error, will keep retying until timeout")

				// The query failed, we'll keep retrying until timeout
				time.Sleep(100 * time.Millisecond)
				continue
			}

			switch lastKnownState := response.PartitionState; lastKnownState {
			case ddpb.PartitionState_PARTITION_CONFIGURING:
				payload["lastKnownState"] = lastKnownState.String()
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "poll for DD partition state",
					OperationStepStatus: pb.OpStatus_ONGOING,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
				})

				time.Sleep(100 * time.Millisecond)
			case ddpb.PartitionState_PARTITION_CONFIGURED:
				payload["lastKnownState"] = lastKnownState.String()
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_OK,
					OperationStep:       "poll for DD partition state",
					OperationStepStatus: pb.OpStatus_DONE_OK,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
				})

				break PARTITION_STATE_POLLING
			default:
				err = fmt.Errorf("PartitionInitialize landed on unexpected state %s (expected: PARTITION_CONFIGURED)", response.PartitionState.String())

				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("partition", envId).
					WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
					WithField("call", "PartitionInitialize").
					Error("DDsched error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["lastKnownState"] = lastKnownState.String()
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_ERROR,
					OperationStep:       "poll for DD partition state",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               err.Error(),
				})

				break PARTITION_STATE_POLLING
			}

			if ctx.Err() != nil {
				err = fmt.Errorf("PartitionInitialize timeout exceeded. Latest state %s (expected: PARTITION_CONFIGURED)", response.PartitionState.String())

				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("partition", envId).
					WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
					WithField("call", "PartitionInitialize").
					WithField("timeout", timeout).
					Error("DDsched error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["lastKnownState"] = response.PartitionState.String()
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_TIMEOUT,
					OperationStep:       "poll for DD partition state",
					OperationStepStatus: pb.OpStatus_DONE_TIMEOUT,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               err.Error(),
				})

				break
			}
		}
		return
	}
	stack["PartitionTerminate"] = func() (out string) { // must formally return string even when we return nothing
		log.WithField("partition", envId).
			Debug("performing DD scheduler PartitionTerminate")

		in := ddpb.PartitionTermRequest{
			PartitionInfo: &ddpb.PartitionInfo{
				EnvironmentId: envId,
				PartitionId:   envId,
			},
		}
		var err error
		callFailedStr := "DDsched PartitionTerminate call failed"

		if p.ddSchedClient == nil {
			err = fmt.Errorf("DD scheduler plugin not initialized, PartitionTerminate impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("call", "PartitionTerminate").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		if p.ddSchedClient.GetConnState() == connectivity.Shutdown {
			err = fmt.Errorf("DD scheduler client connection not available, PartitionTerminate impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("call", "PartitionTerminate").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		var response *ddpb.PartitionResponse
		timeout := callable.AcquireTimeout(DDSCHED_TERMINATE_TIMEOUT, varStack, "Terminate", envId)
		ctx, cancel := integration.NewContext(envId, varStack, timeout)
		defer cancel()

		payload := map[string]interface{}{
			"ddschedRequest": &in,
		}
		payloadJson, _ := json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_STARTED,
			OperationStep:       "perform DD scheduler call: PartitionTerminate",
			OperationStepStatus: pb.OpStatus_STARTED,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		response, err = p.ddSchedClient.PartitionTerminate(ctx, &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("call", "PartitionTerminate").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "perform DD scheduler call: PartitionTerminate",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               err.Error(),
			})

			return
		}

		if response.PartitionState != ddpb.PartitionState_PARTITION_TERMINATING &&
			response.PartitionState != ddpb.PartitionState_PARTITION_TERMINATED {
			err = fmt.Errorf("PartitionTerminate returned unexpected state %s (expected: PARTITION_TERMINATING)", response.PartitionState.String())

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("call", "PartitionTerminate").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			payload["ddschedResponse"] = &response
			payloadJson, _ = json.Marshal(payload)
			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "perform DD scheduler call: PartitionTerminate",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               err.Error(),
			})

			return
		}

		payload["ddschedResponse"] = &response
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_ONGOING,
			OperationStep:       "perform DD scheduler call: PartitionTerminate",
			OperationStepStatus: pb.OpStatus_DONE_OK,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		payload = map[string]interface{}{
			"lastKnownState": response.PartitionState.String(),
		}
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_ONGOING,
			OperationStep:       "poll for DD partition state",
			OperationStepStatus: pb.OpStatus_STARTED,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

	PARTITION_STATE_POLLING:
		for ctx.Err() == nil {
			response, err = p.ddSchedClient.PartitionStatus(ctx, in.PartitionInfo, grpc.EmptyCallOption{})
			if err != nil {
				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("partition", envId).
					WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
					WithField("call", "PartitionStatus").
					WithField("timeout", timeout.String()).
					Error("DDsched error, will keep retying until timeout")

				// The query failed, we'll keep retrying until timeout
				time.Sleep(100 * time.Millisecond)
				continue
			}

			switch lastKnownState := response.PartitionState; lastKnownState {
			case ddpb.PartitionState_PARTITION_TERMINATING:
				payload["lastKnownState"] = lastKnownState.String()
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "poll for DD partition state",
					OperationStepStatus: pb.OpStatus_ONGOING,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
				})

				time.Sleep(100 * time.Millisecond)
			case ddpb.PartitionState_PARTITION_TERMINATED:
				payload["lastKnownState"] = lastKnownState.String()
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_OK,
					OperationStep:       "poll for DD partition state",
					OperationStepStatus: pb.OpStatus_DONE_OK,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
				})

				break PARTITION_STATE_POLLING
			default:
				err = fmt.Errorf("PartitionTerminate landed on unexpected state %s (expected: PARTITION_TERMINATED)", response.PartitionState.String())

				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("partition", envId).
					WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
					WithField("call", "PartitionTerminate").
					Error("DDsched error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["lastKnownState"] = lastKnownState.String()
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_ERROR,
					OperationStep:       "poll for DD partition state",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               err.Error(),
				})

				break PARTITION_STATE_POLLING
			}
			if ctx.Err() != nil {
				err = fmt.Errorf("PartitionTerminate timeout exceeded. Latest state %s (expected: PARTITION_TERMINATED)", response.PartitionState.String())

				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("partition", envId).
					WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
					WithField("call", "PartitionTerminate").
					WithField("timeout", timeout).
					Error("DDsched error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["lastKnownState"] = response.PartitionState.String()
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_TIMEOUT,
					OperationStep:       "poll for DD partition state",
					OperationStepStatus: pb.OpStatus_DONE_TIMEOUT,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               err.Error(),
				})

				break
			}
		}
		return
	}
	stack["EnsureTermination"] = func() (out string) {
		log.WithField("partition", envId).
			Debug("performing DD scheduler session cleanup")

		var err error
		callFailedStr := "DDsched EnsureTermination call failed"

		if p.ddSchedClient == nil {
			err = fmt.Errorf("DD scheduler plugin not initialized, EnsureTermination impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("call", "EnsureTermination").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		if p.ddSchedClient.GetConnState() == connectivity.Shutdown {
			err = fmt.Errorf("DD scheduler client connection not available, EnsureTermination impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("call", "EnsureTermination").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		var response *ddpb.PartitionResponse

		infoReq := ddpb.PartitionInfo{
			EnvironmentId: envId,
			PartitionId:   envId,
		}
		timeout := callable.AcquireTimeout(DDSCHED_TERMINATE_TIMEOUT, varStack, "Terminate", envId)
		ctx, cancel := integration.NewContext(envId, varStack, timeout)
		defer cancel()

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_STARTED,
			OperationStep:       "check DD partition status",
			OperationStepStatus: pb.OpStatus_STARTED,
			EnvironmentId:       envId,
		})

		response, err = p.ddSchedClient.PartitionStatus(ctx, &infoReq, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("call", "EnsureTermination").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "check DD partition status",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Error:               err.Error(),
			})

			return
		}

		if response == nil {
			err = errors.New("nil response")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("call", "EnsureTermination").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "check DD partition status",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Error:               err.Error(),
			})

			return
		}

		// If the partition is anything other than UNKNOWN, TERMINATING or TERMINATED,
		// we need to trigger a PartitionTerminate.
		// This usually happens when the PartitionTerminate hook call was skipped due
		// to a failed deployment or other environment error.
		if response.PartitionState == ddpb.PartitionState_PARTITION_UNKNOWN ||
			response.PartitionState == ddpb.PartitionState_PARTITION_TERMINATING ||
			response.PartitionState == ddpb.PartitionState_PARTITION_TERMINATED {
			// DDsched is in an acceptable state, so we return
			log.WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("partition", envId).
				WithField("partition_state", response.PartitionState).
				Trace("DD scheduler session cleanup not needed")

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_OK,
				OperationStep:       "check DD partition status",
				OperationStepStatus: pb.OpStatus_DONE_OK,
				EnvironmentId:       envId,
			})

			return
		}

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_ONGOING,
			OperationStep:       "check DD partition status",
			OperationStepStatus: pb.OpStatus_DONE_OK,
			EnvironmentId:       envId,
			Error:               "partition still exists, cleanup needed",
		})

		// No guarantee that the DDsched partition is in an obedient state or able
		// to perform control commands, but if it's CONFIGURED we should be able
		// call PartitionTerminate
		in := ddpb.PartitionTermRequest{
			PartitionInfo: &ddpb.PartitionInfo{
				EnvironmentId: envId,
				PartitionId:   envId,
			},
		}

		log.WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
			WithField("partition", envId).
			WithField("partition_state", response.PartitionState).
			WithField("level", infologger.IL_Support).
			Warn("DD scheduler partition still active, performing PartitionTerminate")

		payload := map[string]interface{}{
			"ddschedRequest": &in,
		}
		payloadJson, _ := json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_ONGOING,
			OperationStep:       "perform DD scheduler call: PartitionTerminate",
			OperationStepStatus: pb.OpStatus_STARTED,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		response, err = p.ddSchedClient.PartitionTerminate(ctx, &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("call", "EnsureTermination").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "perform DD scheduler call: PartitionTerminate",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               err.Error(),
			})

			return
		}
		if response.PartitionState != ddpb.PartitionState_PARTITION_TERMINATING &&
			response.PartitionState != ddpb.PartitionState_PARTITION_TERMINATED {
			err = fmt.Errorf("PartitionTerminate returned unexpected state %s (expected: PARTITION_TERMINATING)", response.PartitionState.String())

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				WithField("call", "PartitionTerminate").
				Error("DDsched error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			payload["ddschedResponse"] = &response
			payloadJson, _ = json.Marshal(payload)
			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "perform DD scheduler call: PartitionTerminate",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               err.Error(),
			})

			return
		}

		payload["ddschedResponse"] = &response
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_ONGOING,
			OperationStep:       "perform DD scheduler call: PartitionTerminate",
			OperationStepStatus: pb.OpStatus_DONE_OK,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		payload = map[string]interface{}{
			"lastKnownState": response.PartitionState.String(),
		}
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_ONGOING,
			OperationStep:       "poll for DD partition state",
			OperationStepStatus: pb.OpStatus_STARTED,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

	PARTITION_STATE_POLLING:
		for ctx.Err() == nil {
			response, err = p.ddSchedClient.PartitionStatus(ctx, in.PartitionInfo, grpc.EmptyCallOption{})
			if err != nil {
				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("partition", envId).
					WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
					WithField("call", "PartitionStatus").
					WithField("timeout", timeout.String()).
					Error("DDsched error, will keep retying until timeout")

				// The query failed, we'll keep retrying until timeout
				time.Sleep(100 * time.Millisecond)
				continue
			}

			switch lastKnownState := response.PartitionState; lastKnownState {
			case ddpb.PartitionState_PARTITION_TERMINATING:
				payload["lastKnownState"] = lastKnownState.String()
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "poll for DD partition state",
					OperationStepStatus: pb.OpStatus_ONGOING,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
				})

				time.Sleep(100 * time.Millisecond)
			case ddpb.PartitionState_PARTITION_TERMINATED:
				payload["lastKnownState"] = lastKnownState.String()
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_OK,
					OperationStep:       "poll for DD partition state",
					OperationStepStatus: pb.OpStatus_DONE_OK,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
				})

				break PARTITION_STATE_POLLING
			default:
				err = fmt.Errorf("PartitionTerminate landed on unexpected state %s (expected: PARTITION_TERMINATED)", response.PartitionState.String())

				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("partition", envId).
					WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
					WithField("call", "EnsureTermination").
					Error("DDsched error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["lastKnownState"] = lastKnownState.String()
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_ERROR,
					OperationStep:       "poll for DD partition state",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               err.Error(),
				})

				break PARTITION_STATE_POLLING
			}
			if ctx.Err() != nil {
				err = fmt.Errorf("PartitionTerminate timeout exceeded. Latest state %s (expected: PARTITION_TERMINATED)", response.PartitionState.String())

				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("partition", envId).
					WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
					WithField("call", "EnsureTermination").
					WithField("timeout", timeout).
					Error("DDsched error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["lastKnownState"] = response.PartitionState.String()
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_TIMEOUT,
					OperationStep:       "poll for DD partition state",
					OperationStepStatus: pb.OpStatus_DONE_TIMEOUT,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               err.Error(),
				})

				break
			}
		}
		return
	}
	return
}

func (p *Plugin) Destroy() error {
	return p.ddSchedClient.Close()
}
