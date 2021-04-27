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
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/AliceO2Group/Control/core/integration"
	ddpb "github.com/AliceO2Group/Control/core/integration/ddsched/protos"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const DDSCHED_DIAL_TIMEOUT = 2 * time.Second


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

func (p *Plugin) ObjectStack(data interface{}) (stack map[string]interface{}) {
	call, ok := data.(*callable.Call)
	if !ok {
		return
	}
	varStack := call.VarStack
	stack = make(map[string]interface{})
	stack["PartitionInitialize"] = func() (out string) {	// must formally return string even when we return nothing
		log.Debug("performing DD scheduler PartitionInitialize")

		parentRoleI := call.GetParentRole()
		parentRole, ok := parentRoleI.(workflow.Role)
		root := workflow.GetRoot(parentRole)

		p.stfbHostIdMap = make(map[string]string)
		p.stfsHostIdMap = make(map[string]string)

		workflow.LeafWalk(root, func(role workflow.Role) {
			roleVS, err := role.ConsolidatedVarStack()
			if err != nil {
				log.WithError(err).Error("error processing DD host_id_map")
				return
			}
			var(
				ddDiscoveryIbHostname, ddDiscoveryStfbId, ddDiscoveryStfsId string
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

		envId, ok := varStack["environment_id"]
		if !ok {
			log.Error("cannot acquire environment ID for DD scheduler PartitionInitialize")
			return
		}

		in := ddpb.PartitionInitRequest{
			PartitionInfo: &ddpb.PartitionInfo{
				EnvironmentId: envId,
				PartitionId:   envId,
			},
			StfbHostIdMap: p.stfbHostIdMap,
			StfsHostIdMap: p.stfsHostIdMap,
		}
		if p.ddSchedClient == nil {
			log.WithError(fmt.Errorf("DD scheduler plugin not initialized")).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler PartitionInitialize")
			return
		}
		if p.ddSchedClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("DD scheduler client connection not available")).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler PartitionInitialize")
			return
		}

		var (
			response *ddpb.PartitionResponse
			err error
		)
		response, err = p.ddSchedClient.PartitionInitialize(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler PartitionInitialize")
			return
		}
		if response.PartitionState != ddpb.PartitionState_PARTITION_CONFIGURING &&
			response.PartitionState != ddpb.PartitionState_PARTITION_CONFIGURED {
			log.WithError(fmt.Errorf("PartitionInitialize returned unexpected state %s (expected: PARTITION_CONFIGURING)", response.PartitionState.String())).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler PartitionInitialize")
			return
		}

		PARTITION_STATE_POLLING:
		for {
			response, err = p.ddSchedClient.PartitionStatus(context.Background(), in.PartitionInfo, grpc.EmptyCallOption{})
			switch response.PartitionState {
			case ddpb.PartitionState_PARTITION_CONFIGURING:
				time.Sleep(100 * time.Millisecond)
			case ddpb.PartitionState_PARTITION_CONFIGURED:
				break PARTITION_STATE_POLLING
			default:
				log.WithError(fmt.Errorf("PartitionInitialize landed on unexpected state %s (expected: PARTITION_CONFIGURED)", response.PartitionState.String())).
					WithField("environment_id", envId).
					WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
					Error("failed to perform DD scheduler PartitionInitialize")
				break PARTITION_STATE_POLLING
			}
		}
		return
	}
	stack["PartitionTerminate"] = func() (out string) {	// must formally return string even when we return nothing
		log.Debug("performing DD scheduler PartitionTerminate")

		envId, ok := varStack["environment_id"]
		if !ok {
			log.Error("cannot acquire environment ID for DD scheduler PartitionTerminate")
			return
		}

		in := ddpb.PartitionTermRequest{
			PartitionInfo: &ddpb.PartitionInfo{
				EnvironmentId: envId,
				PartitionId:   envId,
			},
		}
		if p.ddSchedClient == nil {
			log.WithError(fmt.Errorf("DD scheduler plugin not initialized")).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler PartitionTerminate")
			return
		}
		if p.ddSchedClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("DD scheduler client connection not available")).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler PartitionTerminate")
			return
		}

		var (
			response *ddpb.PartitionResponse
			err error
		)
		response, err = p.ddSchedClient.PartitionTerminate(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler PartitionTerminate")
		}
		if response.PartitionState != ddpb.PartitionState_PARTITION_TERMINATING &&
			response.PartitionState != ddpb.PartitionState_PARTITION_TERMINATED {
			log.WithError(fmt.Errorf("PartitionTerminate returned unexpected state %s (expected: PARTITION_TERMINATING)", response.PartitionState.String())).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler PartitionTerminate")
		}

		PARTITION_STATE_POLLING:
		for {
			response, err = p.ddSchedClient.PartitionStatus(context.Background(), in.PartitionInfo, grpc.EmptyCallOption{})
			switch response.PartitionState {
			case ddpb.PartitionState_PARTITION_TERMINATING:
				time.Sleep(100 * time.Millisecond)
			case ddpb.PartitionState_PARTITION_TERMINATED:
				break PARTITION_STATE_POLLING
			default:
				log.WithError(fmt.Errorf("PartitionTerminate landed on unexpected state %s (expected: PARTITION_TERMINATED)", response.PartitionState.String())).
					WithField("environment_id", envId).
					WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
					Error("failed to perform DD scheduler PartitionTerminate")
				break PARTITION_STATE_POLLING
			}
		}
		return
	}
	stack["EnsureTermination"] = func() (out string) {
		log.Debug("performing DD scheduler session cleanup")

		envId, ok := varStack["environment_id"]
		if !ok {
			log.Error("cannot acquire environment ID for DD scheduler session cleanup")
			return
		}

		if p.ddSchedClient == nil {
			log.WithError(fmt.Errorf("DD scheduler plugin not initialized")).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler session cleanup")
			return
		}
		if p.ddSchedClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("DD scheduler client connection not available")).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler session cleanup")
			return
		}

		var (
			response *ddpb.PartitionResponse
			err error
		)

		infoReq := ddpb.PartitionInfo{
			EnvironmentId: envId,
			PartitionId:   envId,
		}
		response, err = p.ddSchedClient.PartitionStatus(context.Background(), &infoReq, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler session cleanup")
			return
		}

		if response == nil {
			log.WithError(errors.New("nil response")).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler session cleanup")
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
				WithField("environment_id", envId).
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
			WithField("environment_id", envId).
			WithField("partition_state", response.PartitionState).
			Warn("DD scheduler partition still active, performing PartitionTerminate")

			response, err = p.ddSchedClient.PartitionTerminate(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler PartitionTerminate")
		}
		if response.PartitionState != ddpb.PartitionState_PARTITION_TERMINATING &&
			response.PartitionState != ddpb.PartitionState_PARTITION_TERMINATED {
			log.WithError(fmt.Errorf("PartitionTerminate returned unexpected state %s (expected: PARTITION_TERMINATING)", response.PartitionState.String())).
				WithField("environment_id", envId).
				WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
				Error("failed to perform DD scheduler PartitionTerminate")
		}

		PARTITION_STATE_POLLING:
		for {
			response, err = p.ddSchedClient.PartitionStatus(context.Background(), in.PartitionInfo, grpc.EmptyCallOption{})
			switch response.PartitionState {
			case ddpb.PartitionState_PARTITION_TERMINATING:
				time.Sleep(100 * time.Millisecond)
			case ddpb.PartitionState_PARTITION_TERMINATED:
				break PARTITION_STATE_POLLING
			default:
				log.WithError(fmt.Errorf("PartitionTerminate landed on unexpected state %s (expected: PARTITION_TERMINATED)", response.PartitionState.String())).
					WithField("environment_id", envId).
					WithField("endpoint", viper.GetString("ddSchedulerEndpoint")).
					Error("failed to perform DD scheduler PartitionTerminate")
				break PARTITION_STATE_POLLING
			}
		}
		return
	}
	return
}

func (p *Plugin) Destroy() error {
	return p.ddSchedClient.Close()
}

