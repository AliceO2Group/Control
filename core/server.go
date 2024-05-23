/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2022 CERN and copyright holders of ALICE O².
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

//go:generate protoc -I=./ -I=../common/ --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/o2control.proto

package core

import (
	"encoding/json"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	evpb "github.com/AliceO2Group/Control/common/protos"
	"github.com/AliceO2Group/Control/common/system"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/repos"
	"github.com/AliceO2Group/Control/core/repos/varsource"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/jinzhu/copier"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/AliceO2Group/Control/common/product"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/the"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/protos"
)

const MAX_ERROR_LENGTH = 6000 // gRPC seems to impose this limit on the status message

func NewServer(state *globalState) *grpc.Server {
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	pb.RegisterControlServer(s, &RpcServer{
		state:      state,
		envStreams: newSafeStreamsMap(),
	})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	return s
}

func (m *RpcServer) logMethod() {
	//if !viper.GetBool("verbose") {
	//	return
	//}
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return
	}
	fun := runtime.FuncForPC(pc)
	if fun == nil {
		return
	}
	log.WithPrefix("rpcserver").
		WithField("method", fun.Name()).
		WithField("level", infologger.IL_Support).
		Debug("handling RPC request")
}

func (m *RpcServer) logMethodHandled() {
	//if !viper.GetBool("verbose") {
	//	return
	//}
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return
	}
	fun := runtime.FuncForPC(pc)
	if fun == nil {
		return
	}
	log.WithPrefix("rpcserver").
		WithField("method", fun.Name()).
		WithField("level", infologger.IL_Support).
		Debug("handling RPC request DONE")
}

// Implements interface pb.ControlServer
type RpcServer struct {
	state      *globalState
	envStreams SafeStreamsMap
}

func (m *RpcServer) GetIntegratedServices(ctx context.Context, empty *pb.Empty) (*pb.ListIntegratedServicesReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	r := &pb.ListIntegratedServicesReply{Services: nil}

	services := make(map[string]*pb.IntegratedServiceInfo)

	for pluginName, _ := range integration.RegisteredPlugins() {
		s := &pb.IntegratedServiceInfo{}
		var plugin integration.Plugin
		for _, p := range integration.PluginsInstance() {
			if pluginName == p.GetName() {
				plugin = p
				break
			}
		}

		// If this plugin was loaded
		if plugin != nil {
			s = &pb.IntegratedServiceInfo{
				Name:            plugin.GetPrettyName(),
				Enabled:         false,
				Endpoint:        plugin.GetEndpoint(),
				ConnectionState: plugin.GetConnectionState(),
				Data:            plugin.GetData(nil),
			}
			enabledPlugins := viper.GetStringSlice("integrationPlugins")
			if utils.StringSliceContains(enabledPlugins, pluginName) {
				s.Enabled = true
			}
		}
		services[pluginName] = s
	}

	r.Services = services
	return r, nil
}

func (m *RpcServer) GetFrameworkInfo(context.Context, *pb.GetFrameworkInfoRequest) (*pb.GetFrameworkInfoReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	maj, _ := strconv.ParseInt(product.VERSION_MAJOR, 10, 32)
	min, _ := strconv.ParseInt(product.VERSION_MINOR, 10, 32)
	pat, _ := strconv.ParseInt(product.VERSION_PATCH, 10, 32)

	availableDetectors, activeDetectors, allDetectors, err := m.listDetectors()

	if err != nil {
		allDetectors = []string{"NIL"}
		availableDetectors = []string{}
		activeDetectors = []string{}
	}

	r := &pb.GetFrameworkInfoReply{
		FrameworkId:       m.state.taskman.GetFrameworkID(),
		EnvironmentsCount: int32(len(m.state.environments.Ids())),
		TasksCount:        int32(m.state.taskman.TaskCount()),
		State:             m.state.taskman.GetState(),
		HostsCount:        int32(m.state.taskman.AgentCache.Count()),
		InstanceName:      viper.GetString("instanceName"),
		Version: &pb.Version{
			Major:       int32(maj),
			Minor:       int32(min),
			Patch:       int32(pat),
			Build:       product.BUILD,
			VersionStr:  product.VERSION,
			ProductName: product.PRETTY_SHORTNAME,
		},
		ConfigurationEndpoint: viper.GetString("configServiceUri"),
		DetectorsInInstance:   allDetectors,
		ActiveDetectors:       activeDetectors,
		AvailableDetectors:    availableDetectors,
	}
	return r, nil
}

func (*RpcServer) Teardown(context.Context, *pb.TeardownRequest) (*pb.TeardownReply, error) {
	log.WithPrefix("rpcserver").
		WithField("method", "Teardown").
		Debug("implement me")
	return nil, status.New(codes.Unimplemented, "not implemented").Err()
}

func (m *RpcServer) GetEnvironments(cxt context.Context, request *pb.GetEnvironmentsRequest) (*pb.GetEnvironmentsReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	r := &pb.GetEnvironmentsReply{
		FrameworkId:  m.state.taskman.GetFrameworkID(),
		Environments: make(EnvironmentInfos, 0, 0),
	}

	// Get plugin-provided environment data for all envs
	integratedServicesEnvsData := integration.PluginsInstance().GetEnvironmentsShortData(m.state.environments.Ids())

	// Get all environments
	for _, id := range m.state.environments.Ids() {
		env, err := m.state.environments.Environment(id)
		if err != nil {
			log.WithPrefix("rpcserver").
				WithField("error", err).
				WithField("partition", id.String()).
				Error("cannot get environment")
			continue
		}
		if !request.ShowAll && !env.Public {
			continue
		}
		tasks := env.Workflow().GetTasks()
		var defaults, vars, userVars map[string]string
		defaults, vars, userVars, err = env.Workflow().ConsolidatedVarMaps()
		if err != nil {
			// take raw of copy, not raw of original because the actual varstack is mutex protected
			defaults = env.GlobalDefaults.RawCopy()
			vars = env.GlobalVars.RawCopy()
			userVars = env.UserVars.RawCopy()
		}

		isEnvData, ok := integratedServicesEnvsData[id]
		if !ok || isEnvData == nil {
			isEnvData = make(map[string]string)
		}

		e := &pb.EnvironmentInfo{
			Id:                     env.Id().String(),
			CreatedWhen:            env.CreatedWhen().UnixMilli(),
			State:                  env.CurrentState(),
			RootRole:               env.Workflow().GetName(),
			Description:            env.Description,
			CurrentRunNumber:       env.GetCurrentRunNumber(),
			Defaults:               defaults,
			Vars:                   vars,
			UserVars:               userVars,
			NumberOfFlps:           int32(len(env.GetFLPs())),
			NumberOfHosts:          int32(len(env.GetAllHosts())),
			NumberOfTasks:          int32(len(tasks)),
			IntegratedServicesData: isEnvData,
			CurrentTransition:      env.CurrentTransition(),
			NumberOfActiveTasks: int32(len(tasks.Filtered(func(t *task.Task) bool {
				parent := t.GetParentRole()
				if parent == nil {
					return false
				}
				if parentRole, ok := parent.(workflow.Role); ok {
					return parentRole.GetStatus() == task.ACTIVE
				}
				return false
			}))),
			NumberOfInactiveTasks: int32(len(tasks.Filtered(func(t *task.Task) bool {
				parent := t.GetParentRole()
				if parent == nil {
					return true // if task has no parent, we treat it as inactive
				}
				if parentRole, ok := parent.(workflow.Role); ok {
					return parentRole.GetStatus() != task.ACTIVE
				}
				return false
			}))),
		}
		if request.GetShowTaskInfos() {
			e.Tasks = tasksToShortTaskInfos(tasks, m.state.taskman)
		}
		e.IncludedDetectors = env.GetActiveDetectors().StringList()

		r.Environments = append(r.Environments, e)
	}
	sort.Sort(EnvironmentInfos(r.Environments))

	return r, nil
}

func (m *RpcServer) doNewEnvironmentAsync(cxt context.Context, request *pb.NewEnvironmentRequest, id uid.ID) {
	var err error
	id, err = m.state.environments.CreateEnvironment(request.GetWorkflowTemplate(), request.GetVars(), request.GetPublic(), id, request.GetAutoTransition())
	if err != nil {
		the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
			EnvironmentId: id.String(),
			State:         "ERROR",
			Error:         "cannot create new environment",
			Message:       err.Error(),
		})
		return
	}

	newEnv, err := m.state.environments.Environment(id)
	if err != nil {
		the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
			EnvironmentId: id.String(),
			State:         "ERROR",
			Error:         "cannot get newly created environment",
			Message:       err.Error(),
		})
		return
	}

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId: id.String(),
		State:         newEnv.CurrentState(),
	})
	return
}

func (m *RpcServer) NewEnvironmentAsync(cxt context.Context, request *pb.NewEnvironmentRequest) (reply *pb.NewEnvironmentReply, err error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()

	// Create new Environment instance with some roles, we get back a UUID
	id := uid.New()

	go m.doNewEnvironmentAsync(cxt, request, id)

	ei := &pb.EnvironmentInfo{
		Id:       id.String(),
		State:    "PENDING",
		UserVars: request.GetVars(),
	}
	reply = &pb.NewEnvironmentReply{
		Environment: ei,
		Public:      request.GetPublic(),
	}
	return
}

func (m *RpcServer) NewEnvironment(cxt context.Context, request *pb.NewEnvironmentRequest) (reply *pb.NewEnvironmentReply, err error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	// NEW_ENVIRONMENT transition
	// The following should
	// 1) Create a new value of type Environment struct
	// 2) Build the topology and ask Mesos to run all the processes
	// 3) Acquire the status of the processes to ascertain that they are indeed running and
	//    in their STANDBY state
	// 4) Execute the CONFIGURE transition on all the processes, and recheck their status to
	//    make sure they are now successfully in CONFIGURED
	// 5) Report back here with the new environment id and error code, if needed.

	// FIXME: figure out if the state.sm becomes a task.Manager sm, or no global sm at all
	//if m.state.sm.Cannot("NEW_ENVIRONMENT") {
	//	msg := fmt.Sprintf("NEW_ENVIRONMENT transition impossible, current state: %s",
	//		m.state.sm.Current())
	//	return nil, status.Error(codes.Internal, msg)
	//}
	//err := m.state.sm.Event("NEW_ENVIRONMENT") //Async until Transition call
	//defer m.state.sm.Transition()
	//
	//if _, ok := err.(fsm.NoTransitionError); !ok && err != nil {
	//	return nil, status.Newf(codes.Internal, "cannot create new environment: %s", err.Error()).Err()
	//}

	reply = &pb.NewEnvironmentReply{Public: request.Public}

	// Create new Environment instance with some roles, we get back a UUID
	id := uid.New()
	id, err = m.state.environments.CreateEnvironment(request.GetWorkflowTemplate(), request.GetVars(), request.GetPublic(), id, request.GetAutoTransition())
	if err != nil {
		st := status.Newf(codes.Internal, "cannot create new environment: %s", TruncateString(err.Error(), MAX_ERROR_LENGTH))
		ei := &pb.EnvironmentInfo{
			Id:           id.String(),
			CreatedWhen:  time.Now().UnixMilli(),
			State:        "ERROR", // not really, but close
			NumberOfFlps: 0,
		}
		st, _ = st.WithDetails(ei)
		err = st.Err()

		return
	}

	newEnv, err := m.state.environments.Environment(id)
	if err != nil {
		st := status.Newf(codes.Internal, "cannot get newly created environment: %s", TruncateString(err.Error(), MAX_ERROR_LENGTH))
		ei := &pb.EnvironmentInfo{
			Id:           id.String(),
			CreatedWhen:  time.Now().UnixMilli(),
			State:        "ERROR", // not really, but close
			NumberOfFlps: 0,
		}
		st, _ = st.WithDetails(ei)
		err = st.Err()

		return
	}

	tasks := newEnv.Workflow().GetTasks()
	var defaults, vars, userVars map[string]string
	defaults, vars, userVars, err = newEnv.Workflow().ConsolidatedVarMaps()
	if err != nil {
		defaults = newEnv.GlobalDefaults.RawCopy()
		vars = newEnv.GlobalVars.RawCopy()
		userVars = newEnv.UserVars.RawCopy()
		err = nil
	}

	integratedServicesEnvsData := integration.PluginsInstance().GetEnvironmentsData([]uid.ID{id})
	isEnvData, ok := integratedServicesEnvsData[id]
	if !ok {
		isEnvData = make(map[string]string)
	}

	ei := &pb.EnvironmentInfo{
		Id:                     newEnv.Id().String(),
		CreatedWhen:            newEnv.CreatedWhen().UnixMilli(),
		State:                  newEnv.CurrentState(),
		Tasks:                  tasksToShortTaskInfos(tasks, m.state.taskman),
		RootRole:               newEnv.Workflow().GetName(),
		Description:            newEnv.Description,
		CurrentRunNumber:       newEnv.GetCurrentRunNumber(),
		Defaults:               defaults,
		Vars:                   vars,
		UserVars:               userVars,
		NumberOfFlps:           int32(len(newEnv.GetFLPs())),
		NumberOfHosts:          int32(len(newEnv.GetAllHosts())),
		IncludedDetectors:      newEnv.GetActiveDetectors().StringList(),
		IntegratedServicesData: isEnvData,
		CurrentTransition:      newEnv.CurrentTransition(),
		NumberOfActiveTasks: int32(len(tasks.Filtered(func(t *task.Task) bool {
			parent := t.GetParentRole()
			if parent == nil {
				return false
			}
			if parentRole, ok := parent.(workflow.Role); ok {
				return parentRole.GetStatus() == task.ACTIVE
			}
			return false
		}))),
		NumberOfInactiveTasks: int32(len(tasks.Filtered(func(t *task.Task) bool {
			parent := t.GetParentRole()
			if parent == nil {
				return true // if task has no parent, we treat it as inactive
			}
			if parentRole, ok := parent.(workflow.Role); ok {
				return parentRole.GetStatus() != task.ACTIVE
			}
			return false
		}))),
	}
	reply = &pb.NewEnvironmentReply{
		Environment: ei,
		Public:      newEnv.Public,
	}
	return
}

func (m *RpcServer) GetEnvironment(cxt context.Context, req *pb.GetEnvironmentRequest) (reply *pb.GetEnvironmentReply, err error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if req == nil || len(req.Id) == 0 {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	var envId uid.ID
	envId, err = uid.FromString(req.Id)
	if err != nil {
		return nil, status.New(codes.InvalidArgument, "received bad environment id").Err()
	}

	var env *environment.Environment
	env, err = m.state.environments.Environment(envId)
	if err != nil {
		return nil, status.Newf(codes.NotFound, "environment not found: %s", err.Error()).Err()
	}

	tasks := env.Workflow().GetTasks()
	var defaults, vars, userVars map[string]string
	defaults, vars, userVars, err = env.Workflow().ConsolidatedVarMaps()
	if err != nil {
		defaults = env.GlobalDefaults.RawCopy()
		vars = env.GlobalVars.RawCopy()
		userVars = env.UserVars.RawCopy()
	}

	integratedServicesEnvsData := integration.PluginsInstance().GetEnvironmentsData([]uid.ID{envId})
	isEnvData, ok := integratedServicesEnvsData[envId]
	if !ok {
		isEnvData = make(map[string]string)
	}
	reply = &pb.GetEnvironmentReply{
		Environment: &pb.EnvironmentInfo{
			Id:                     env.Id().String(),
			CreatedWhen:            env.CreatedWhen().UnixMilli(),
			State:                  env.CurrentState(),
			Tasks:                  tasksToShortTaskInfos(tasks, m.state.taskman),
			RootRole:               env.Workflow().GetName(),
			Description:            env.Description,
			CurrentRunNumber:       env.GetCurrentRunNumber(),
			Defaults:               defaults,
			Vars:                   vars,
			UserVars:               userVars,
			NumberOfFlps:           int32(len(env.GetFLPs())),
			NumberOfHosts:          int32(len(env.GetAllHosts())),
			IntegratedServicesData: isEnvData,
			CurrentTransition:      env.CurrentTransition(),
			NumberOfActiveTasks: int32(len(tasks.Filtered(func(t *task.Task) bool {
				parent := t.GetParentRole()
				if parent == nil {
					return false
				}
				if parentRole, ok := parent.(workflow.Role); ok {
					return parentRole.GetStatus() == task.ACTIVE
				}
				return false
			}))),
			NumberOfInactiveTasks: int32(len(tasks.Filtered(func(t *task.Task) bool {
				parent := t.GetParentRole()
				if parent == nil {
					return true // if task has no parent, we treat it as inactive
				}
				if parentRole, ok := parent.(workflow.Role); ok {
					return parentRole.GetStatus() != task.ACTIVE
				}
				return false
			}))),
		},
		Public: env.Public,
	}
	if req.GetShowWorkflowTree() {
		reply.Workflow = workflowToRoleTree(env.Workflow())
	}
	reply.Environment.IncludedDetectors = env.GetActiveDetectors().StringList()

	jsonISData, err := json.Marshal(isEnvData)
	log.WithPrefix("rpcserver").
		WithField("method", "GetEnvironment").
		WithField("level", infologger.IL_Support).
		WithField("intServPayloadSize", len(jsonISData)).
		Infof("returning payload incl. integrated services data for %d services", len(isEnvData))
	return
}

func (m *RpcServer) ControlEnvironment(cxt context.Context, req *pb.ControlEnvironmentRequest) (*pb.ControlEnvironmentReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if req == nil || len(req.Id) == 0 {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	envId, err := uid.FromString(req.Id)
	if err != nil {
		return nil, status.New(codes.InvalidArgument, "received bad environment id").Err()
	}

	env, err := m.state.environments.Environment(envId)
	if err != nil {
		return nil, status.Newf(codes.NotFound, "environment not found: %s", err.Error()).Err()
	}

	trans := environment.MakeTransition(m.state.taskman, req.Type)
	if trans == nil {
		return nil, status.Newf(codes.InvalidArgument, "cannot prepare invalid transition %s", req.GetType().String()).Err()
	}

	sot := time.Now()
	err = env.TryTransition(trans)
	eot := time.Now()
	td := eot.Sub(sot)

	if err != nil {
		log.WithField("partition", env.Id()).
			WithField("level", infologger.IL_Ops).
			WithError(err).
			Errorf("transition '%s' failed, transitioning into ERROR.", req.GetType().String())
		err = env.TryTransition(environment.NewGoErrorTransition(m.state.taskman))
		if err != nil {
			log.WithField("partition", env.Id()).Warnf("could not complete requested GO_ERROR transition, forcing move to ERROR: %s", err.Error())
			env.Sm.SetState("ERROR")
		}
	}

	reply := &pb.ControlEnvironmentReply{
		Id:                 env.Id().String(),
		State:              env.CurrentState(),
		CurrentRunNumber:   env.GetCurrentRunNumber(),
		StartOfTransition:  sot.UnixMilli(),
		EndOfTransition:    eot.UnixMilli(),
		TransitionDuration: td.Milliseconds(),
	}

	if err != nil {
		return reply, status.Newf(codes.Aborted, err.Error()).Err()
	}

	return reply, nil
}

func (*RpcServer) ModifyEnvironment(context.Context, *pb.ModifyEnvironmentRequest) (*pb.ModifyEnvironmentReply, error) {
	log.WithPrefix("rpcserver").
		WithField("method", "ModifyEnvironment").
		Debug("implement me")
	return nil, status.New(codes.Unimplemented, "not implemented").Err()
}

func (m *RpcServer) DestroyEnvironment(cxt context.Context, req *pb.DestroyEnvironmentRequest) (*pb.DestroyEnvironmentReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if req == nil || len(req.Id) == 0 {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	envId, err := uid.FromString(req.Id)
	if err != nil {
		return nil, status.New(codes.InvalidArgument, "received bad environment id").Err()
	}

	env, err := m.state.environments.Environment(envId)
	if err != nil {
		return nil, status.Newf(codes.NotFound, "environment not found: %s", err.Error()).Err()
	}

	// if Force immediately disband the environment (unlocking all tasks) and run the cleanup.
	if req.Force {
		return m.doTeardownAndCleanup(env, req.Force, req.KeepTasks)
	}

	if req.AllowInRunningState && env.CurrentState() == "RUNNING" {
		err = env.TryTransition(environment.MakeTransition(m.state.taskman, pb.ControlEnvironmentRequest_STOP_ACTIVITY))
		if err != nil {
			log.WithField("partition", env.Id().String()).
				Warn("could not perform STOP transition for environment teardown, forcing")
			return m.doTeardownAndCleanup(env, true /*force*/, false /*keepTasks*/)
		}
	}

	canDestroy := false
	statesForDestroy := []string{"CONFIGURED", "DEPLOYED", "STANDBY"}

	for _, v := range statesForDestroy {
		if env.CurrentState() == v {
			canDestroy = true
			break
		}
	}

	if !canDestroy {
		log.WithField("partition", env.Id().String()).
			Warnf("cannot teardown environment in state %s, forcing", env.CurrentState())
		return m.doTeardownAndCleanup(env, true /*force*/, false /*keepTasks*/)
	}

	// This might transition to STANDBY if needed, or do nothing if we're already there
	if env.CurrentState() == "CONFIGURED" {
		err = env.TryTransition(environment.MakeTransition(m.state.taskman, pb.ControlEnvironmentRequest_RESET))
		if err != nil {
			log.Warnf("cannot teardown environment in state %s, forcing", env.CurrentState())
			return m.doTeardownAndCleanup(env, true /*force*/, false /*keepTasks*/)
		}
	}

	return m.doTeardownAndCleanup(env, req.Force, req.KeepTasks)
}

func (m *RpcServer) doTeardownAndCleanup(env *environment.Environment, force bool, keepTasks bool) (*pb.DestroyEnvironmentReply, error) {
	log.WithField("partition", env.Id().String()).
		WithField("level", infologger.IL_Ops).
		Info("DESTROY starting")

	err := m.state.environments.TeardownEnvironment(env.Id(), force)
	if err != nil {
		if !force {
			// if the teardown failed, but we haven't tried a Force teardown yet, we retry
			// a second time but with force
			return m.doTeardownAndCleanup(env, true, keepTasks)
		}
		return &pb.DestroyEnvironmentReply{}, status.New(codes.Internal, err.Error()).Err()
	}

	if keepTasks { // Tasks should stay running, so we're done
		return &pb.DestroyEnvironmentReply{}, nil
	}

	// cleanup tasks
	tasksForEnv := env.Workflow().GetTasks().GetTaskIds()
	killed, running, err := m.doCleanupTasks(tasksForEnv)
	ctr := &pb.CleanupTasksReply{KilledTasks: killed, RunningTasks: running}
	if err != nil {
		log.WithError(err).
			WithField("partition", env.Id().String()).
			Error("task cleanup error")
		return &pb.DestroyEnvironmentReply{CleanupTasksReply: ctr}, status.New(codes.Internal, err.Error()).Err()
	}
	return &pb.DestroyEnvironmentReply{CleanupTasksReply: ctr}, nil
}

func (m *RpcServer) GetActiveDetectors(_ context.Context, _ *pb.Empty) (*pb.GetActiveDetectorsReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	r := &pb.GetActiveDetectorsReply{
		Detectors: make([]string, 0),
	}
	detIds := m.state.environments.GetActiveDetectors()
	r.Detectors = detIds.StringList()

	sort.Strings(r.Detectors)
	return r, nil
}

// return string lists of available, active and all detectors
func (m *RpcServer) listDetectors() ([]string, []string, []string, error) {
	cs := the.ConfSvc()
	allDetectors, err := cs.ListDetectors(true)
	if err != nil {
		return nil, nil, nil, err
	}
	activeDetectorsMap := m.state.environments.GetActiveDetectors()
	availableDetectors := make([]string, 0)
	for _, det := range allDetectors {
		detId, err := system.IDString(det)
		if err != nil {
			continue
		}
		if _, contains := activeDetectorsMap[detId]; !contains {
			availableDetectors = append(availableDetectors, det)
		}
	}

	return availableDetectors, activeDetectorsMap.StringList(), allDetectors, nil
}

func (m *RpcServer) GetAvailableDetectors(_ context.Context, _ *pb.Empty) (*pb.GetAvailableDetectorsReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if availableDetectors, _, _, err := m.listDetectors(); err != nil {
		return nil, err
	} else {
		r := &pb.GetAvailableDetectorsReply{
			Detectors: availableDetectors,
		}
		return r, nil
	}
}

func (m *RpcServer) GetTasks(context.Context, *pb.GetTasksRequest) (*pb.GetTasksReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	tasks := m.state.taskman.GetTasks()
	r := &pb.GetTasksReply{
		Tasks: tasksToShortTaskInfos(tasks, m.state.taskman),
	}

	return r, nil
}

func (m *RpcServer) GetTask(cxt context.Context, req *pb.GetTaskRequest) (*pb.GetTaskReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	task := m.state.taskman.GetTask(req.TaskId)
	if task == nil {
		return &pb.GetTaskReply{}, status.New(codes.NotFound, "task not found").Err()
	}
	taskClass := task.GetTaskClass()
	commandInfo := task.GetTaskCommandInfo()
	var outbound []channel.Outbound
	var inbound []channel.Inbound
	taskPath := ""
	// TODO: probably not the nicest way to do this... the outbound assignments should be cached
	// in the Task
	if task.IsLocked() {
		type parentRole interface {
			CollectOutboundChannels() []channel.Outbound
			GetPath() string
			CollectInboundChannels() []channel.Inbound
		}
		parent, ok := task.GetParentRole().(parentRole)
		if ok {
			outbound = channel.MergeOutbound(parent.CollectOutboundChannels(), taskClass.Connect)
			taskPath = parent.GetPath()
			inbound = channel.MergeInbound(parent.CollectInboundChannels(), taskClass.Bind)
		}
	}
	if inbound == nil {
		inbound = make([]channel.Inbound, len(taskClass.Bind))
		copy(inbound, taskClass.Bind)
	}

	rep := &pb.GetTaskReply{
		Task: &pb.TaskInfo{
			ShortInfo: taskToShortTaskInfo(task, m.state.taskman),
			ClassInfo: &pb.TaskClassInfo{
				Name:        task.GetClassName(),
				ControlMode: task.GetControlMode().String(),
			},
			InboundChannels:  inboundChannelsToPbChannels(inbound),
			OutboundChannels: outboundChannelsToPbChannels(outbound),
			CommandInfo:      commandInfoToPbCommandInfo(commandInfo),
			TaskPath:         taskPath,
			EnvId:            task.GetEnvironmentId().String(),
			Properties:       task.GetProperties(),
		},
	}
	return rep, nil
}

func (m *RpcServer) CleanupTasks(cxt context.Context, req *pb.CleanupTasksRequest) (*pb.CleanupTasksReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	idsToKill := req.GetTaskIds()

	killed, running, err := m.doCleanupTasks(idsToKill)
	if err != nil {
		log.WithError(err).Error("task cleanup error")
		return &pb.CleanupTasksReply{KilledTasks: killed, RunningTasks: running}, status.New(codes.Internal, err.Error()).Err()
	}

	return &pb.CleanupTasksReply{KilledTasks: killed, RunningTasks: running}, nil
}

func (m *RpcServer) doCleanupTasks(taskIds []string) (killedTaskInfos []*pb.ShortTaskInfo, runningTaskInfos []*pb.ShortTaskInfo, err error) {
	var (
		killedTasks, runningTasks task.Tasks
	)
	if len(taskIds) == 0 { // by default we try to kill all, best effort
		killedTasks, runningTasks, err = m.state.taskman.Cleanup()
	} else {
		killedTasks, runningTasks, err = m.state.taskman.KillTasks(taskIds)
	}

	killedTaskInfos = tasksToShortTaskInfos(killedTasks, m.state.taskman)
	runningTaskInfos = tasksToShortTaskInfos(runningTasks, m.state.taskman)

	return
}

func (m *RpcServer) GetRoles(cxt context.Context, req *pb.GetRolesRequest) (*pb.GetRolesReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if req == nil || len(req.EnvId) == 0 {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	envId, err := uid.FromString(req.EnvId)
	if err != nil {
		return nil, status.New(codes.InvalidArgument, "received bad environment id").Err()
	}

	env, err := m.state.environments.Environment(envId)
	if err != nil {
		return nil, status.Newf(codes.NotFound, "environment not found: %s", err.Error()).Err()
	}

	resultRoles := env.QueryRoles(req.PathSpec)

	roleInfos := make([]*pb.RoleInfo, len(resultRoles))
	for i, rr := range resultRoles {
		roleInfos[i] = workflowToRoleTree(rr)
	}
	return &pb.GetRolesReply{Roles: roleInfos}, nil
}

func (m *RpcServer) GetWorkflowTemplates(cxt context.Context, req *pb.GetWorkflowTemplatesRequest) (*pb.GetWorkflowTemplatesReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if req == nil {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	workflowMap, numWorkflows, err := the.RepoManager().GetWorkflowTemplates(req.GetRepoPattern(), req.GetRevisionPattern(), req.GetAllBranches(), req.GetAllTags(), req.GetAllWorkflows())
	if err != nil {
		return nil, status.New(codes.InvalidArgument, "cannot query available workflows for "+req.GetRepoPattern()+"@"+req.GetRevisionPattern()+": "+
			err.Error()).Err()
	}

	// Behaviour: in AliECS, template resolution follows specific rules of variable priority
	// Apricot defaults are the most defaulty, so in the context of VarSpec, which entails as a minimum
	// a declaration of type default in the WFT, no Apricot default can ever override a VarSpec default,
	// so we don't even need to access Apricot defaults at all. These are only used if a variable is
	// read in the WFT but never declared.
	// Therefore the only thing from Apricot that can affect the VarSpec is a vars entry, and due to
	// priority rules, Apricot vars can only override WFT/TT defaults, because WFT/TT vars have higher
	// priority than Apricot vars. This way we ensure that the GUI receives any default values as
	// expected with the correct precedence order with respect to Apricot entries.
	// Here we make sure that for each VarSpec entry sourced from the WFT:
	//   * if declared as default
	//     * if the same declaration exists as Apricot var -> override
	//     * else no change
	//   * else no change
	vars := the.ConfSvc().GetVars()

	workflowTemplateInfos := make([]*pb.WorkflowTemplateInfo, numWorkflows)
	i := 0
	for repo, revisions := range workflowMap {
		for revision, templates := range revisions {
			for _, template := range templates {
				// First we take care of overriding any WFT VarSpec defaults with Apricot vars
				varSpecMap := make(repos.VarSpecMap)
				if template.VarInfo != nil {
					_ = copier.Copy(&varSpecMap, template.VarInfo)
				}

				for k, v := range varSpecMap {
					if v.Source == varsource.WorkflowDefaults { // if this varSpec was declared as a default
						if apricotValue, exists := vars[k]; exists { // and a corresponding Apricot var exists
							v.DefaultValue = apricotValue
							varSpecMap[k] = v
						}
					}
				}

				// Finally, we build the protobuf response
				workflowTemplateInfos[i] = &pb.WorkflowTemplateInfo{
					Repo:        string(repo),
					Revision:    string(revision),
					Template:    template.Name,
					VarSpecMap:  VarSpecMapToPbVarSpecMap(varSpecMap),
					Description: template.Description,
				}
				i++
			}
		}
	}

	return &pb.GetWorkflowTemplatesReply{WorkflowTemplates: workflowTemplateInfos}, nil
}

func (m *RpcServer) ListRepos(cxt context.Context, req *pb.ListReposRequest) (*pb.ListReposReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if req == nil {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	repoList := the.RepoManager().GetAllRepos()
	repoInfos := make([]*pb.RepoInfo, len(repoList))

	// Ensure alphabetical order of repos in output
	keys := the.RepoManager().GetOrderedRepolistKeys()

	for i, repoName := range keys {
		repo := repoList[repoName]
		var revisions []string
		if req.GetRevisions {
			revisions = repo.GetRevisions()
		} else {
			revisions = nil
		}
		repoInfos[i] = &pb.RepoInfo{Name: repoName, Default: repo.IsDefault(), DefaultRevision: repo.GetDefaultRevision(), Revisions: revisions}
	}

	return &pb.ListReposReply{Repos: repoInfos, GlobalDefaultRevision: viper.GetString("globalDefaultRevision")}, nil
}

func (m *RpcServer) AddRepo(cxt context.Context, req *pb.AddRepoRequest) (*pb.AddRepoReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if req == nil {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	newDefaultRevision, isGlobalDefault, err := the.RepoManager().AddRepo(req.Name, req.DefaultRevision)
	if err != nil {
		return nil, err
	}

	var info string
	if newDefaultRevision == req.DefaultRevision {
		info = "The default revision for this repository has been set to \"" + newDefaultRevision + "\"."
	} else if isGlobalDefault {
		info = "The default revision for this repository has been set to \"" + newDefaultRevision + "\" (global default value)."
	} else {
		info = "The default revision for this repository has been set to \"" + newDefaultRevision + "\" (fallback value)."
	}

	return &pb.AddRepoReply{NewDefaultRevision: newDefaultRevision, Info: info}, nil
}

func (m *RpcServer) RemoveRepo(cxt context.Context, req *pb.RemoveRepoRequest) (*pb.RemoveRepoReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if req == nil {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	newDefaultRepo, err := the.RepoManager().RemoveRepoByIndex(int(req.Index))

	if err != nil {
		return nil, err
	}

	return &pb.RemoveRepoReply{NewDefaultRepo: newDefaultRepo}, nil
}

func (m *RpcServer) RefreshRepos(cxt context.Context, req *pb.RefreshReposRequest) (*pb.Empty, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if req == nil {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	var err error
	if int(req.Index) == -1 {
		err = the.RepoManager().RefreshRepos()
	} else {
		err = the.RepoManager().RefreshRepoByIndex(int(req.Index))
	}
	if err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil
}

func (m *RpcServer) SetDefaultRepo(cxt context.Context, req *pb.SetDefaultRepoRequest) (*pb.Empty, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if req == nil {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	err := the.RepoManager().UpdateDefaultRepoByIndex(int(req.Index))
	if err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil
}

func (m *RpcServer) SetGlobalDefaultRevision(cxt context.Context, req *pb.SetGlobalDefaultRevisionRequest) (*pb.Empty, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if req == nil {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	err := the.RepoManager().SetGlobalDefaultRevision(req.Revision)
	if err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil
}

func (m *RpcServer) SetRepoDefaultRevision(cxt context.Context, req *pb.SetRepoDefaultRevisionRequest) (*pb.SetRepoDefaultRevisionReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	if req == nil {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	info, err := the.RepoManager().UpdateDefaultRevisionByIndex(int(req.Index), req.Revision)
	if err != nil {
		return &pb.SetRepoDefaultRevisionReply{Info: info}, nil // Info is filled with available revisions
		// err can't be set here, otherwise the response will be empty
	}

	return &pb.SetRepoDefaultRevisionReply{Info: info}, nil // Info is empty
}

func (m *RpcServer) Subscribe(req *pb.SubscribeRequest, srv pb.Control_SubscribeServer) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	for {
		ch, chOk := m.envStreams.GetChannel(req.GetId())
		if !chOk {
			continue
		}
		select {
		case event, ok := <-ch:
			if !ok {
				m.envStreams.delete(req.GetId())
				return nil
			}
			err := srv.Send(event)
			if err != nil {
				log.WithError(err).
					WithField("subscribe", req.GetId()).
					Error(err.Error())
			}
		}
	}
}

func (m *RpcServer) NewAutoEnvironment(cxt context.Context, request *pb.NewAutoEnvironmentRequest) (*pb.NewAutoEnvironmentReply, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("rpcserver"))
	m.logMethod()
	defer m.logMethodHandled()

	ch := make(chan *pb.Event)
	m.envStreams.add(request.GetId(), ch)
	sub := environment.SubscribeToStream(ch)
	id := uid.New()
	go m.state.environments.CreateAutoEnvironment(request.GetWorkflowTemplate(), request.GetVars(), id, sub)
	r := &pb.NewAutoEnvironmentReply{}
	return r, nil
}
