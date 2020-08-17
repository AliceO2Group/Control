/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2020 CERN and copyright holders of ALICE O².
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

//go:generate protoc --gofast_out=plugins=grpc:. protos/o2control.proto
package core

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/spf13/viper"

	"github.com/AliceO2Group/Control/common/product"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/the"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/protos"
	"github.com/looplab/fsm"
	"github.com/mesos/mesos-go/api/v1/lib/extras/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)


func NewServer(state *globalState) *grpc.Server {
	s := grpc.NewServer()
	pb.RegisterControlServer(s, &RpcServer{
		state: state,
	})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	return s
}

func (m *RpcServer) logMethod() {
	if !viper.GetBool("verbose") {
		return
	}
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
		Trace("handling RPC request")
}

// Implements interface pb.ControlServer
type RpcServer struct {
	state       *globalState
}

func (*RpcServer) TrackStatus(*pb.StatusRequest, pb.Control_TrackStatusServer) error {
	log.WithPrefix("rpcserver").
		WithField("method", "TrackStatus").
		Debug("implement me")

	return status.New(codes.Unimplemented, "not implemented").Err()
}

func (m *RpcServer) GetFrameworkInfo(context.Context, *pb.GetFrameworkInfoRequest) (*pb.GetFrameworkInfoReply, error) {
	m.logMethod()

	maj, _ := strconv.ParseInt(product.VERSION_MAJOR, 10, 32)
	min, _ := strconv.ParseInt(product.VERSION_MINOR, 10, 32)
	pat, _ := strconv.ParseInt(product.VERSION_PATCH, 10, 32)

	r := &pb.GetFrameworkInfoReply{
		FrameworkId:        m.state.taskman.GetFrameworkID(),
		EnvironmentsCount:  int32(len(m.state.environments.Ids())),
		TasksCount:         int32(m.state.taskman.TaskCount()),
		State:              m.state.taskman.GetState(),
		HostsCount:         int32(m.state.taskman.AgentCache.Count()),
		InstanceName:       viper.GetString("instanceName"),
		Version:            &pb.Version{
			Major:          int32(maj),
			Minor:          int32(min),
			Patch:          int32(pat),
			Build:          product.BUILD,
			VersionStr:     product.VERSION,
			ProductName:    product.PRETTY_SHORTNAME,
		},
	}
	return r, nil
}

func (*RpcServer) Teardown(context.Context, *pb.TeardownRequest) (*pb.TeardownReply, error) {
	log.WithPrefix("rpcserver").
		WithField("method", "Teardown").
		Debug("implement me")
	return nil, status.New(codes.Unimplemented, "not implemented").Err()
}

func (m *RpcServer) GetEnvironments(context.Context, *pb.GetEnvironmentsRequest) (*pb.GetEnvironmentsReply, error) {
	m.logMethod()

	r := &pb.GetEnvironmentsReply{
		FrameworkId: m.state.taskman.GetFrameworkID(),
		Environments: make(EnvironmentInfos, 0, 0),
	}
	for _, id := range m.state.environments.Ids() {
		env, err := m.state.environments.Environment(id)
		if err != nil {
			log.WithPrefix("rpcserver").
				WithField("error", err).
				WithField("partition", id.String()).
				Error("cannot get environment")
			continue
		}
		tasks := env.Workflow().GetTasks()
		e := &pb.EnvironmentInfo{
			Id:               env.Id().String(),
			CreatedWhen:      env.CreatedWhen().Format(time.RFC3339),
			State:            env.CurrentState(),
			Tasks:            tasksToShortTaskInfos(tasks),
			RootRole:         env.Workflow().GetName(),
			CurrentRunNumber: env.GetCurrentRunNumber(),
			Defaults:         env.GlobalDefaults.Raw(),
			Vars:             env.GlobalVars.Raw(),
			UserVars:         env.UserVars.Raw(),
		}

		r.Environments = append(r.Environments, e)
	}
	sort.Sort(EnvironmentInfos(r.Environments))

	return r, nil
}

func (m *RpcServer) NewEnvironment(cxt context.Context, request *pb.NewEnvironmentRequest) (*pb.NewEnvironmentReply, error) {
	m.logMethod()
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

	// Create new Environment instance with some roles, we get back a UUID
	id, err := m.state.environments.CreateEnvironment(request.GetWorkflowTemplate(), request.GetVars())
	if err != nil {
		return nil, status.Newf(codes.Internal, "cannot create new environment: %s", err.Error()).Err()
	}

	newEnv, err := m.state.environments.Environment(id)
	if err != nil {
		return nil, status.Newf(codes.Internal, "cannot get newly created environment: %s", err.Error()).Err()
	}

	tasks := newEnv.Workflow().GetTasks()
	ei := &pb.EnvironmentInfo{
			Id: newEnv.Id().String(),
			CreatedWhen: newEnv.CreatedWhen().Format(time.RFC3339),
			State: newEnv.CurrentState(),
			Tasks: tasksToShortTaskInfos(tasks),
			RootRole: newEnv.Workflow().GetName(),
			CurrentRunNumber: newEnv.GetCurrentRunNumber(),
		}
	r := &pb.NewEnvironmentReply{
		Environment: ei,
	}
	select {
	case m.state.PublicEvent <- pb.NewEnvironmentCreatedEvent(ei):
	default:
		log.Debug("state.PublicEvent channel is full")
	}
	return r, nil
}

func (m *RpcServer) GetEnvironment(cxt context.Context, req *pb.GetEnvironmentRequest) (*pb.GetEnvironmentReply, error) {
	m.logMethod()

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

	tasks := env.Workflow().GetTasks()
	r := &pb.GetEnvironmentReply{
		Environment: &pb.EnvironmentInfo{
			Id: env.Id().String(),
			CreatedWhen: env.CreatedWhen().Format(time.RFC3339),
			State: env.CurrentState(),
			Tasks: tasksToShortTaskInfos(tasks),
			RootRole: env.Workflow().GetName(),
			CurrentRunNumber: env.GetCurrentRunNumber(),
			Defaults: env.GlobalDefaults.Raw(),
			Vars: env.GlobalVars.Raw(),
			UserVars: env.UserVars.Raw(),
		},
		Workflow: workflowToRoleTree(env.Workflow()),
	}
	return r, nil
}

func (m *RpcServer) ControlEnvironment(cxt context.Context, req *pb.ControlEnvironmentRequest) (*pb.ControlEnvironmentReply, error) {
	m.logMethod()

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

	err = env.TryTransition(trans)

	reply := &pb.ControlEnvironmentReply{
		Id: env.Id().String(),
		State: env.CurrentState(),
		CurrentRunNumber: env.GetCurrentRunNumber(),
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
	m.logMethod()

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
		err = m.state.environments.TeardownEnvironment(env.Id(), req.Force)
		if err != nil {
			return &pb.DestroyEnvironmentReply{}, status.New(codes.Internal, err.Error()).Err()
		}

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

	if req.AllowInRunningState && env.CurrentState() == "RUNNING" {
		err = env.TryTransition(environment.MakeTransition(m.state.taskman, pb.ControlEnvironmentRequest_STOP_ACTIVITY))
		if err != nil {
			return &pb.DestroyEnvironmentReply{}, status.New(codes.Internal, err.Error()).Err()
		}
	}

	canDestroy := false
	statesForDestroy := []string{"CONFIGURED", "STANDBY"}

	for _, v := range statesForDestroy {
		if env.CurrentState() == v {
			canDestroy = true
			break
		}
	}

	if !canDestroy {
		return nil, status.Newf(codes.FailedPrecondition, "cannot destroy environment in state %s", env.CurrentState()).Err()
	}

	// This might transition to STANDBY if needed, of do nothing if we're already there
	if env.CurrentState() == "CONFIGURED" {
		err = env.TryTransition(environment.MakeTransition(m.state.taskman, pb.ControlEnvironmentRequest_RESET))
		if err != nil {
			return &pb.DestroyEnvironmentReply{}, status.New(codes.Internal, err.Error()).Err()
		}
	}

	err = m.state.environments.TeardownEnvironment(env.Id(), req.Force)
	if err != nil {
		return &pb.DestroyEnvironmentReply{}, status.New(codes.Internal, err.Error()).Err()
	}

	if req.KeepTasks { // Tasks should stay running, so we're done
		return &pb.DestroyEnvironmentReply{}, nil
	}

	tasksForEnv := env.Workflow().GetTasks().GetTaskIds()
	killed, running, err := m.doCleanupTasks(tasksForEnv)
	ctr := &pb.CleanupTasksReply{KilledTasks: killed, RunningTasks: running}
	if err != nil {
		log.WithError(err).
			WithField("partition", env.Id().String()).
			Error("task cleanup error")
		return &pb.DestroyEnvironmentReply{CleanupTasksReply: ctr}, status.New(codes.Internal, err.Error()).Err()
	}
	select {
	case m.state.PublicEvent <- pb.NewEnvironmentDestroyedEvent(ctr, env.Id().String()):
	default:
		log.Debug("state.PublicEvent channel is full")
	}
	return &pb.DestroyEnvironmentReply{CleanupTasksReply: ctr}, nil
}

func (m *RpcServer) GetTasks(context.Context, *pb.GetTasksRequest) (*pb.GetTasksReply, error) {
	m.logMethod()

	tasks := m.state.taskman.GetTasks()
	r := &pb.GetTasksReply{
		Tasks: tasksToShortTaskInfos(tasks),
	}

	return r, nil
}

func (m *RpcServer) GetTask(cxt context.Context, req *pb.GetTaskRequest) (*pb.GetTaskReply, error) {
	m.logMethod()

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
			ShortInfo: taskToShortTaskInfo(task),
			ClassInfo: &pb.TaskClassInfo{
				Name: task.GetClassName(),
				ControlMode: task.GetControlMode().String(),
			},
			InboundChannels: inboundChannelsToPbChannels(inbound),
			OutboundChannels: outboundChannelsToPbChannels(outbound),
			CommandInfo: commandInfoToPbCommandInfo(commandInfo),
			TaskPath: taskPath,
			EnvId: task.GetEnvironmentId().String(),
			Properties: task.GetProperties(),
		},
	}
	return rep, nil
}

func (m *RpcServer) CleanupTasks(cxt context.Context, req *pb.CleanupTasksRequest) (*pb.CleanupTasksReply, error) {
	m.logMethod()
	idsToKill := req.GetTaskIds()

	killed, running, err := m.doCleanupTasks(idsToKill)
	if err != nil {
		log.WithError(err).Error("task cleanup error")
		return &pb.CleanupTasksReply{KilledTasks: killed, RunningTasks: running}, status.New(codes.Internal, err.Error()).Err()
	}

	return &pb.CleanupTasksReply{KilledTasks: killed, RunningTasks: running}, nil
}

func (m *RpcServer) doCleanupTasks(taskIds []string) (killedTaskInfos []*pb.ShortTaskInfo, runningTaskInfos []*pb.ShortTaskInfo, err error) {
	var(
		killedTasks, runningTasks task.Tasks
	)
	if len(taskIds) == 0 { // by default we try to kill all, best effort
		killedTasks, runningTasks, err = m.state.taskman.Cleanup()
	} else {
		killedTasks, runningTasks, err = m.state.taskman.KillTasks(taskIds)
	}

	killedTaskInfos = tasksToShortTaskInfos(killedTasks)
	runningTaskInfos = tasksToShortTaskInfos(runningTasks)

	return
}

func (m *RpcServer) GetRoles(cxt context.Context, req *pb.GetRolesRequest) (*pb.GetRolesReply, error) {
	m.logMethod()

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
	m.logMethod()

	if req == nil {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	workflowMap, numWorkflows, err := the.RepoManager().GetWorkflowTemplates(req.GetRepoPattern(), req.GetRevisionPattern(), req.GetAllBranches(), req.GetAllTags())
	if err != nil {
		return nil, status.New(codes.InvalidArgument, "cannot query available workflows for " + req.GetRepoPattern() + "@" + req.GetRevisionPattern() + ": " +
			err.Error()).Err()
	}

	workflowTemplateInfos := make([]*pb.WorkflowTemplateInfo, numWorkflows)
	i := 0
	for repo, revisions := range workflowMap {
		for revision, templates := range revisions {
			for _, template := range templates {
				workflowTemplateInfos[i] = &pb.WorkflowTemplateInfo{Repo: string(repo), Revision: string(revision), Template: string(template)}
				i++
			}
		}
	}

	return &pb.GetWorkflowTemplatesReply{WorkflowTemplates: workflowTemplateInfos}, nil
}

func (m *RpcServer) ListRepos(cxt context.Context, req *pb.ListReposRequest) (*pb.ListReposReply, error) {
	m.logMethod()

	if req == nil {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	repoList := the.RepoManager().GetAllRepos()
	repoInfos := make([]*pb.RepoInfo, len(repoList))

	// Ensure alphabetical order of repos in output
	keys := the.RepoManager().GetOrderedRepolistKeys()

	for i, repoName := range keys {
		repo := repoList[repoName]
		repoInfos[i] = &pb.RepoInfo{Name: repoName, Default: repo.Default, DefaultRevision: repo.DefaultRevision}
	}

	return &pb.ListReposReply{Repos: repoInfos, GlobalDefaultRevision: viper.GetString("globalDefaultRevision")}, nil
}

func (m *RpcServer) AddRepo(cxt context.Context, req *pb.AddRepoRequest) (*pb.AddRepoReply, error) {
	m.logMethod()

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
	m.logMethod()

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
	m.logMethod()

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
	m.logMethod()

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
	m.logMethod()

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
	m.logMethod()

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
	m.logMethod()
	ctx := srv.Context()
	for {
		select {
		case event := <- m.state.PublicEvent:
			m.state.RLock()
			err := srv.Send(event)
			if err != nil {
				return err
			}
			m.state.RUnlock()
		case <-ctx.Done():
            return ctx.Err()
		}
	}
}
