/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
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

//go:generate protoc --go_out=. --go-grpc_out=require_unimplemented_servers=false:. protos/ddsched.proto

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

	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	ddpb "github.com/AliceO2Group/Control/core/integration/ddsched/protos"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

const (
	DDSCHED_DIAL_TIMEOUT            = 2 * time.Second
	DDSCHED_INITIALIZE_TIMEOUT      = 30 * time.Second
	DDSCHED_TERMINATE_TIMEOUT       = 30 * time.Second
	DDSCHED_DEFAULT_POLLING_TIMEOUT = 30 * time.Second
)

type Plugin struct {
	ddSchedulerHost string
	ddSchedulerPort int

	stfbHostIdMap map[string]string //map[task_id]ib_hostname
	stfsHostIdMap map[string]string

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
		stfbHostIdMap:   nil,
		stfsHostIdMap:   nil,
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

func (p *Plugin) GetData(environmentIds []uid.ID) string {
	if p == nil || p.ddSchedClient == nil {
		return ""
	}

	partitionStates := make(map[string]string)

	for _, envId := range environmentIds {
		in := ddpb.PartitionInfo{
			PartitionId:   envId.String(),
			EnvironmentId: envId.String(),
		}
		state, err := p.ddSchedClient.PartitionStatus(context.Background(), &in, grpc.EmptyCallOption{})
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
		partitionStates[envId.String()] = partitionState
	}

	out, err := json.Marshal(partitionStates)
	if err != nil {
		return ""
	}
	return string(out[:])
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

func (p *Plugin) ObjectStack(_ map[string]string) (stack map[string]interface{}) {
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

		p.stfbHostIdMap = make(map[string]string)
		p.stfsHostIdMap = make(map[string]string)

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
					p.stfsHostIdMap[ddDiscoveryStfsId] = ddDiscoveryIbHostname
				} else if ddDiscoveryStfbIdOk {
					p.stfbHostIdMap[ddDiscoveryStfbId] = ddDiscoveryIbHostname
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
			StfbHostIdMap:   p.stfbHostIdMap,
			StfsHostIdMap:   p.stfsHostIdMap,
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

		//if p.ddSchedClient.GetConnState() != connectivity.Ready {
		//	err = fmt.Errorf("DD scheduler client connection not available, PartitionInitialize impossible")
		//
		//	log.WithError(err).
		//		WithField("level", infologger.IL_Support).
		//		WithField("partition", envId).
		//		WithField("call", "PartitionInitialize").
		//		Error("DDsched error")
		//
		//	call.VarStack["__call_error_reason"] = err.Error()
		//	call.VarStack["__call_error"] = callFailedStr
		//
		//	return
		//}

		var (
			response *ddpb.PartitionResponse
		)
		timeout := callable.AcquireTimeout(DDSCHED_INITIALIZE_TIMEOUT, varStack, "Initialize", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
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

			return
		}

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

			switch response.PartitionState {
			case ddpb.PartitionState_PARTITION_CONFIGURING:
				time.Sleep(100 * time.Millisecond)
			case ddpb.PartitionState_PARTITION_CONFIGURED:
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
		//if p.ddSchedClient.GetConnState() != connectivity.Ready {
		//	err = fmt.Errorf("DD scheduler client connection not available, PartitionTerminate impossible")
		//
		//	log.WithError(err).
		//		WithField("level", infologger.IL_Support).
		//		WithField("partition", envId).
		//		WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
		//		WithField("call", "PartitionTerminate").
		//		Error("DDsched error")
		//
		//	call.VarStack["__call_error_reason"] = err.Error()
		//	call.VarStack["__call_error"] = callFailedStr
		//
		//	return
		//}

		var (
			response *ddpb.PartitionResponse
		)
		timeout := callable.AcquireTimeout(DDSCHED_TERMINATE_TIMEOUT, varStack, "Terminate", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
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

			return
		}

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

			switch response.PartitionState {
			case ddpb.PartitionState_PARTITION_TERMINATING:
				time.Sleep(100 * time.Millisecond)
			case ddpb.PartitionState_PARTITION_TERMINATED:
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
		//if p.ddSchedClient.GetConnState() != connectivity.Ready {
		//	err = fmt.Errorf("DD scheduler client connection not available, EnsureTermination impossible")
		//
		//	log.WithError(err).
		//		WithField("level", infologger.IL_Support).
		//		WithField("partition", envId).
		//		WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
		//		WithField("call", "EnsureTermination").
		//		Error("DDsched error")
		//
		//	call.VarStack["__call_error_reason"] = err.Error()
		//	call.VarStack["__call_error"] = callFailedStr
		//
		//	return
		//}

		var (
			response *ddpb.PartitionResponse
		)

		infoReq := ddpb.PartitionInfo{
			EnvironmentId: envId,
			PartitionId:   envId,
		}
		timeout := callable.AcquireTimeout(DDSCHED_TERMINATE_TIMEOUT, varStack, "Terminate", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
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
			return
		}

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
			Warn("DD scheduler partition still active, performing PartitionTerminate")

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

			return
		}

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

			switch response.PartitionState {
			case ddpb.PartitionState_PARTITION_TERMINATING:
				time.Sleep(100 * time.Millisecond)
			case ddpb.PartitionState_PARTITION_TERMINATED:
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
