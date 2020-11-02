/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: Miltiadis <miltiadis.alexis@cern.ch>
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


// This a draft refactor of manager.go with goal to achieve event-driven
// architecture.

package task

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/gera"
	pb "github.com/AliceO2Group/Control/core/protos"
	"github.com/AliceO2Group/Control/core/task/taskop"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/mesos/mesos-go/api/v1/lib/extras/store"

	"github.com/AliceO2Group/Control/common/utils"

	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/k0kubun/pp"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/calls"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

const TaskMan_QUEUE = 1024


type ManagerV2 struct {
	ctx                context.Context
	fidStore           store.Singleton
	AgentCache         AgentCache
	MessageChannel     chan *TaskmanMessage

	classes            *classes
	roster             *roster

	resourceOffersDone <-chan DeploymentMap
	tasksToDeploy      chan<- Descriptors
	reviveOffersTrg    chan struct{}
	cq                 *controlcommands.CommandQueue

	tasksLaunched      int
	tasksFinished      int

	schedulerState     *schedulerState
	publicEventCh      chan<- *pb.Event
	internalEventCh    chan<- event.Event
}

func NewManagerV2(shutdown func(),
	publicEventCh chan<- *pb.Event,
	internalEventCh chan<- event.Event) (taskman *ManagerV2, err error) {
	// TODO(jdef) how to track/handle timeout errors that occur for SUBSCRIBE calls? we should
	// probably tolerate X number of subsequent subscribe failures before bailing. we'll need
	// to track the lastCallAttempted along with subsequentSubscribeTimeouts.

	// store.Singleton is a thread-safe abstraction to load and store and string,
	// provided by mesos-go.
	// We also make sure that a log message is printed with the FrameworkID.
	fidStore := store.DecorateSingleton(
		store.NewInMemorySingleton(),
		store.DoSet().AndThen(func(_ store.Setter, v string, _ error) error {
			// Store Mesos Framework ID to configuration.
			err = the.ConfSvc().NewMesosFID(v)
			if err != nil {
				log.WithField("error", err).Error("cannot write to configuration")
			}
			log.WithField("frameworkId", v).Debug("frameworkId")
			return nil
		}))

	// Set Framework ID from the configuration
	if fidValue, err := the.ConfSvc().GetMesosFID(); err == nil {
		store.SetOrPanic(fidStore)(fidValue)
	}

	taskman = &ManagerV2{
		classes:            newClasses(),
		roster:             newRoster(),
		publicEventCh:      publicEventCh,
		internalEventCh:    internalEventCh,
	}
	schedulerState, err := NewScheduler(taskman, fidStore, shutdown)
	if err != nil {
		return nil, err
	}
	taskman.schedulerState = schedulerState
	taskman.cq = taskman.schedulerState.commandqueue	// FIXME remove
	taskman.resourceOffersDone = taskman.schedulerState.resourceOffersDone
	taskman.tasksToDeploy = taskman.schedulerState.tasksToDeploy
	taskman.reviveOffersTrg = taskman.schedulerState.reviveOffersTrg

	schedulerState.setupCli()

	return
}

func (m *ManagerV2) GetState() string {
	return m.schedulerState.sm.Current()
}

func (m *ManagerV2) GetFrameworkID() string {
	return m.schedulerState.GetFrameworkID()
}


// NewTaskForMesosOffer accepts a Mesos offer and a Descriptor and returns a newly
// constructed Task.
// This function should only be called by the Mesos scheduler controller when
// matching role requests with offers (matchRoles).
func (m *ManagerV2) newTaskForMesosOffer(
	offer *mesos.Offer,
	descriptor *Descriptor,
	localBindMap channel.BindMap,
	executorId mesos.ExecutorID) (t *Task) {
	newId := uuid.NewUUID().String()
	t = &Task{
		name:         fmt.Sprintf("%s#%s", descriptor.TaskClassName, newId),
		parent:       descriptor.TaskRole,
		className:    descriptor.TaskClassName,
		hostname:     offer.Hostname,
		agentId:      offer.AgentID.Value,
		offerId:      offer.ID.Value,
		taskId:       newId,
		properties:   gera.MakeStringMap().Wrap(m.GetTaskClass(descriptor.TaskClassName).Properties),
		executorId:   executorId.Value,
		GetTaskClass: nil,
		localBindMap: nil,
		state:        STANDBY,
		status:       INACTIVE,
	}
	t.GetTaskClass = func() *Class {
		return m.GetTaskClass(t.className)
	}
	t.localBindMap = make(channel.BindMap)
	for k, v := range localBindMap {
		t.localBindMap[k] = v
	}
	return
}

func (m *ManagerV2) removeInactiveClasses() {

	classMap := m.classes.getMap()

	for taskClassIdentifier := range classMap {
		if len(m.roster.filteredForClass(taskClassIdentifier)) == 0 {
			m.classes.deleteKey(taskClassIdentifier)
		}
	}

	return
}

func (m *ManagerV2) RemoveReposClasses(repoPath string) { //Currently unused

	utils.EnsureTrailingSlash(&repoPath)

	for taskClassIdentifier := range m.classes.getMap() {
		if strings.HasPrefix(taskClassIdentifier, repoPath) &&
			len(m.roster.filteredForClass(taskClassIdentifier)) == 0 {
			m.classes.deleteKey(taskClassIdentifier)
		}
	}

	return
}

func (m *ManagerV2) RefreshClasses(taskClassesRequired []string) (err error) {

	m.removeInactiveClasses()

	var taskClassList []*Class
	taskClassList, err = getTaskClassList(taskClassesRequired)
	if err != nil {
		return err
	}

	for _, class := range taskClassList {
		taskClassIdentifier := class.Identifier.String()
		// If it already exists we update, otherwise we add the new class
		if m.classes.contains(taskClassIdentifier) {
			m.classes.updateClass(taskClassIdentifier, class)
		} else {
			m.classes.addClass(taskClassIdentifier, class)
		}
	}
	return
}

func (m *ManagerV2) acquireTasks(envId uid.ID, taskDescriptors Descriptors) (err error) {

	/*
	Here's what's gonna happen:
	1) check if any tasks are already in Roster, whether they are already locked
	   in an environment, and whether their host has attributes that satisfy the
	   constraints
	  1a) TODO: for each of them in Roster with matching attributes but wrong class,
	      mark for teardown and mesos-deployment
	  1b) for each of them in Roster with matching attributes and class, mark for
	      takeover and reconfiguration
	3) TODO: teardown the tasks in tasksToTeardown
	4) start the tasks in tasksToRun
	5) ensure that all of them reach a CONFIGURED state
	*/

	tasksToRun := make(Descriptors, 0)
	// TODO: also filter for tasks that satisfy attributes but are of wrong class,
	// to be torn down and mesosed.
	// tasksToTeardown := make([]TaskPtr, 0)
	tasksAlreadyRunning := make(DeploymentMap)
	for _, descriptor := range taskDescriptors {
		/*
		For each descriptor we check m.AgentCache for agent attributes:
		this allows us for each idle task in roster, get agentid and plug it in cache to get the
		attributes for that host, and then we know whether
		1) a running task of the same class on that agent would qualify
		2) TODO: given enough resources obtained by freeing tasks, that agent would qualify
		 */

		// Filter function that accepts a Task if
		// a) it's !Locked
		// b) has className matching Descriptor
		// c) its Agent's Attributes satisfy the Descriptor's Constraints
		taskMatches := func(taskPtr *Task) (ok bool) {
			if taskPtr != nil {
				if !taskPtr.IsLocked() && taskPtr.className == descriptor.TaskClassName {
					agentInfo := m.AgentCache.Get(mesos.AgentID{Value: taskPtr.agentId})
					taskClass, classFound := m.classes.getClass(descriptor.TaskClassName)
					if classFound && taskClass != nil && agentInfo != nil {
						targetConstraints := descriptor.
							RoleConstraints.MergeParent(taskClass.Constraints)
						if agentInfo.Attributes.Satisfy(targetConstraints) {
							ok = true
							return
						}
					}
				}
			}
			return
		}
		runningTasksForThisDescriptor := m.roster.filtered(taskMatches)
		claimed := false
		if len(runningTasksForThisDescriptor) > 0 {
			// We have received a list of running, unlocked candidates to take over
			// the current Descriptor.
			// We now go through them, and if one of them is already claimed in
			// tasksAlreadyRunning, we go to the next one until we exhaust our options.
			for _, taskPtr := range runningTasksForThisDescriptor {
				if _, ok := tasksAlreadyRunning[taskPtr]; ok {
					continue
				} else { // task not claimed yet, we do so now
					tasksAlreadyRunning[taskPtr] = descriptor
					claimed = true
					break
				}
			}
		}

		if !claimed {
			tasksToRun = append(tasksToRun, descriptor)
		}
	}

	// TODO: fill out tasksToTeardown in the previous loop
	//if len(tasksToTeardown) > 0 {
	//	err = m.TeardownTasks(tasksToTeardown)
	//	if err != nil {
	//		return errors.New(fmt.Sprintf("cannot restart roles: %s", err.Error()))
	//	}
	//}

	// At this point, all descriptors are either
	// - matched to a TaskPtr in tasksAlreadyRunning
	// - awaiting Task deployment in tasksToRun
	deploymentSuccess := true // hopefully

	deployedTasks := make(DeploymentMap)
	if len(tasksToRun) > 0 {
		// Alright, so we have some descriptors whose requirements should be met with
		// new Tasks we're about to deploy here.
		// First we ask Mesos to revive offers and block until done, then upon receiving
		// the offers, we ask Mesos to run the required roles - if any.

		m.reviveOffersTrg <- struct{}{} // signal scheduler to revive offers
		<- m.reviveOffersTrg            // we only continue when it's done

		m.tasksToDeploy <- tasksToRun // blocks until received
		log.WithField("environmentId", envId).
			Debug("scheduler should have received request to deploy")

		// IDEA: a flps mesos-role assigned to all mesos agents on flp hosts, and then a static
		//       reservation for that mesos-role on behalf of our scheduler

		deployedTasks = <- m.resourceOffersDone
		log.WithField("tasks", deployedTasks).
			Debug("resourceOffers is done, new tasks running")

		if len(deployedTasks) != len(tasksToRun) {
			// ↑ Not all roles could be deployed. We cannot proceed
			//   with running this environment, but we keep the roles
			//   running since they might be useful in the future.
			deploymentSuccess = false
		}
	}

	if deploymentSuccess {
		// ↑ means all the required processes are now running, and we
		//   are ready to update the envId
		for taskPtr, descriptor := range deployedTasks {
			taskPtr.SetParent(descriptor.TaskRole)
			// Ensure everything is filled out properly
			if !taskPtr.IsLocked() {
				log.WithField("task", taskPtr.taskId).Warning("cannot lock newly deployed task")
				deploymentSuccess = false
			}
		}
	}
	if !deploymentSuccess {
		// While all the required roles are running, for some reason we
		// can't lock some of them, so we must roll back and keep them
		// unlocked in the roster.
		var deployedTaskIds []string
		for taskPtr, _ := range deployedTasks {
			taskPtr.SetParent(nil)
			deployedTaskIds = append(deployedTaskIds, taskPtr.taskId)
		}
		err = TasksDeploymentError{taskIds: deployedTaskIds}
	}

	// Finally, we write to the roster. Point of no return!
	for taskPtr, _ := range deployedTasks {
		m.roster.append(taskPtr)
	}
	if deploymentSuccess {
		for taskPtr, _ := range deployedTasks {
			taskPtr.GetParent().SetTask(taskPtr)
		}
		for taskPtr, descriptor := range tasksAlreadyRunning {
			taskPtr.SetParent(descriptor.TaskRole)
			taskPtr.GetParent().SetTask(taskPtr)
		}
	}

	return
}

func (m *ManagerV2) releaseTasks(envId uid.ID, tasks Tasks) error {

	taskReleaseErrors := make(map[string]error)
	taskIdsReleased := make([]string, 0)

	for _, task := range tasks {
		err := m.releaseTask(envId, task)
		if err == nil {
			taskIdsReleased = append(taskIdsReleased, task.GetTaskId())
		} else {
			switch err.(type) {
			case TaskAlreadyReleasedError:
				continue
			default:
				taskReleaseErrors[task.GetTaskId()] = err
			}
		}
	}

	m.internalEventCh <- event.NewTasksReleasedEvent(envId, taskIdsReleased, taskReleaseErrors)

	return nil
}

func (m *ManagerV2) releaseTask(envId uid.ID, task *Task) error {
	if task == nil {
		return TaskNotFoundError{}
	}
	if task.IsLocked() && task.GetEnvironmentId() != envId {
		return TaskLockedError{taskErrorBase: taskErrorBase{taskId: task.name}, envId: envId}
	}

	task.SetParent(nil)

	return nil
}

func (m *ManagerV2) configureTasks(envId uid.ID, tasks Tasks) error {
	notify := make(chan controlcommands.MesosCommandResponse)
	receivers, err := tasks.GetMesosCommandTargets()
	if err != nil {
		return err
	}

	// We generate a "bindMap" i.e. a map of the paths of registered inbound channels and their ports
	bindMap := make(channel.BindMap)
	for _, task := range tasks {
		taskPath := task.GetParent().GetPath()
		for inbChName, endpoint := range task.GetLocalBindMap() {
			bindMap[taskPath + TARGET_SEPARATOR + inbChName] =
				endpoint.ToTargetEndpoint(task.GetHostname())
		}
	}
	log.WithFields(logrus.Fields{"bindMap": pp.Sprint(bindMap), "envId": envId.String()}).
		Debug("generated inbound bindMap for environment configuration")

	src := STANDBY.String()
	event := "CONFIGURE"
	dest := CONFIGURED.String()
	args := make(controlcommands.PropertyMapsMap)
	args, err = tasks.BuildPropertyMaps(bindMap)
	if err != nil {
		return err
	}
	log.WithField("map", pp.Sprint(args)).Debug("pushing configuration to tasks")

	cmd := controlcommands.NewMesosCommand_Transition(receivers, src, event, dest, args)
	m.cq.Enqueue(cmd, notify)

	response := <- notify
	close(notify)

	if response == nil {
		return errors.New("nil response")
	}

	errText := response.Err().Error()
	if len(strings.TrimSpace(errText)) != 0 {
		return errors.New(response.Err().Error())
	}

	// FIXME: improve error handling ↑

	return nil
}

func (m *ManagerV2) transitionTasks(tasks Tasks, src string, event string, dest string, commonArgs controlcommands.PropertyMap) error {
	notify := make(chan controlcommands.MesosCommandResponse)
	receivers, err := tasks.GetMesosCommandTargets()

	if err != nil {
		return err
	}

	args := make(controlcommands.PropertyMapsMap)

	// If we're pushing some arg values to all targets...
	if len(commonArgs) > 0 {
		for _, rec := range receivers {
			args[rec] = make(controlcommands.PropertyMap)
			for k, v := range commonArgs {
				args[rec][k] = v
			}
		}
	}

	cmd := controlcommands.NewMesosCommand_Transition(receivers, src, event, dest, args)
	m.cq.Enqueue(cmd, notify)

	response := <- notify
	close(notify)

	if response == nil {
		return errors.New("unknown MesosCommand error: nil response received")
	}

	errText := response.Err().Error()
	if len(strings.TrimSpace(errText)) != 0 {
		return errors.New(response.Err().Error())
	}

	// FIXME: improve error handling ↑

	return nil
}

func (m *ManagerV2) TriggerHooks(tasks Tasks) error {
	if len(tasks) == 0 {
		return nil
	}

	notify := make(chan controlcommands.MesosCommandResponse)
	receivers, err := tasks.GetMesosCommandTargets()

	if err != nil {
		return err
	}

	cmd := controlcommands.NewMesosCommand_TriggerHook(receivers)
	err = m.cq.Enqueue(cmd, notify)
	if err != nil {
		return err
	}

	response := <- notify
	close(notify)

	if response == nil {
		return errors.New("unknown MesosCommand error: nil response received")
	}

	errText := response.Err().Error()
	if len(strings.TrimSpace(errText)) != 0 {
		return errors.New(response.Err().Error())
	}

	// FIXME: improve error handling ↑

	return nil
}

func (m *ManagerV2) GetTaskClass(name string) (b *Class) {
	if m == nil {
		return
	}
	b, _ = m.classes.getClass(name)
	return
}

func (m *ManagerV2) TaskCount() int {
	if m == nil {
		return -1
	}
	return len(m.roster.getTasks())
}

func (m *ManagerV2) GetTasks() Tasks {
	if m == nil {
		return nil
	}
	return m.roster.getTasks()
}

func (m *ManagerV2) GetTask(id string) *Task {
	if m == nil {
		return nil
	}
	for _, t := range m.roster.getTasks() {
		if t.taskId == id {
			return t
		}
	}
	return nil
}

func (m *ManagerV2) updateTaskState(taskId string, state string) {

	taskPtr := m.roster.getByTaskId(taskId)
	if taskPtr == nil {
		log.WithField("taskId", taskId).
			Warn("attempted state update of task not in roster")
		return
	}

	st := StateFromString(state)
	taskPtr.state = st
	taskPtr.safeToStop = false
	if taskPtr.GetParent() != nil {
		taskPtr.GetParent().UpdateState(st)
	}
}

func (m *ManagerV2) updateTaskStatus(status *mesos.TaskStatus) {

	taskId := status.GetTaskID().Value
	taskPtr := m.roster.getByTaskId(taskId)
	if taskPtr == nil {
		log.WithField("taskId", taskId).
			Warn("attempted status update of task not in roster")
		return
	}

	switch st := status.GetState(); st {
	case mesos.TASK_RUNNING:
		log.WithField("taskId", taskId).
			WithField("name", taskPtr.GetName()).
			Debug("task running")
		taskPtr.status = ACTIVE
		if taskPtr.GetParent() != nil {
			taskPtr.GetParent().UpdateStatus(ACTIVE)
		}
	case mesos.TASK_DROPPED, mesos.TASK_LOST, mesos.TASK_KILLED, mesos.TASK_FAILED, mesos.TASK_ERROR:
		taskPtr.status = INACTIVE
		if taskPtr.GetParent() != nil {
			taskPtr.GetParent().UpdateStatus(INACTIVE)
		}
	}
}

// Kill all tasks outside an environment (all unlocked tasks)
func (m *ManagerV2) Cleanup() (killed Tasks, running Tasks, err error) {

	toKill := m.roster.filtered(func(t *Task) bool {
		return !t.IsLocked()
	})

	killed, running, err = m.doKillTasks(toKill)
	return
}

// Kill a specific list of tasks.
// If the task list includes locked tasks, TaskNotFoundError is returned.
func (m *ManagerV2) KillTasks(taskIds []string) (killed Tasks, running Tasks, err error) {

	toKill := m.roster.filtered(func(t *Task) bool {
		if t.IsLocked() {
			return false
		}
		for _, id := range taskIds {
			if t.taskId == id {
				return true
			}
		}
		return false
	})

	if len(toKill) < len(taskIds) {
		err = TaskNotFoundError{}
		return
	}

	killed, running, err = m.doKillTasks(toKill)
	return
}

func (m *ManagerV2) doKillTask(task *Task) error {
	return m.schedulerState.killTask(context.TODO(), task.GetMesosCommandTarget())
}


func (m *ManagerV2) doKillTasks(tasks Tasks) (killed Tasks, running Tasks, err error) {
	// We assume all tasks are unlocked

	// Build slice of tasks with status !ACTIVE
	inactiveTasks := tasks.Filtered(func(task *Task) bool {
		return task.status != ACTIVE
	})
	// Remove from the roster the tasks which are also in the inactiveTasks list to delete
	m.roster.updateTasks(m.roster.filtered(func(task *Task) bool {
		return !inactiveTasks.Contains(func(t *Task) bool {
			return t.taskId == task.taskId
		})
	}))

	for _, task := range tasks.Filtered(func(task *Task) bool { return task.status == ACTIVE }) {
		e := m.doKillTask(task)
		if e != nil {
			log.WithError(e).WithField("taskId", task.taskId).Error("could not kill task")
			err = errors.New("could not kill some tasks")
		} else {
			killed = append(killed, task)
		}
	}

	// Remove from the roster the tasks we've just killed
	m.roster.updateTasks(m.roster.filtered(func(task *Task) bool {
		return !killed.Contains(func(t *Task) bool {
			return t.taskId == task.taskId
		})
	}))
	running = m.roster.getTasks()

	return
}

func (m *ManagerV2) Start(ctx context.Context) {
	m.MessageChannel = make(chan *TaskmanMessage, TaskMan_QUEUE)

	m.schedulerState.Start(ctx)

	go func() {
		for {
			select {
			case taskmanMessage, ok := <-m.MessageChannel:
				if !ok {  // if the channel is closed, we bail
					return
				}
				err := m.handleMessage(taskmanMessage)
				if err != nil {
					log.Debug(err)
				}
			}
		}
	}()
}

func (m *ManagerV2) stop() {
	close(m.MessageChannel)
}

func (m *ManagerV2) handleMessage(tm *TaskmanMessage) (error) {
	if m == nil {
		return errors.New("manager is nil")
	}

	messageType := tm.GetMessageType()

	switch messageType{
	case taskop.AcquireTasks:
		go m.acquireTasks(tm.GetEnvironmentId(), tm.GetDescriptors())
	case taskop.ConfigureTasks:
		go m.configureTasks(tm.GetEnvironmentId(),tm.GetTasks())
	case taskop.TransitionTasks:
		go m.transitionTasks(tm.GetTasks(),tm.GetSource(),tm.GetEvent(),tm.GetDestination(),tm.GetArguments())
	case taskop.TaskStatusMessage:
		mesosStatus := tm.status
		mesosState := mesosStatus.GetState()
		switch mesosState {
		case mesos.TASK_FINISHED:
			m.tasksFinished++
		case mesos.TASK_LOST, mesos.TASK_KILLED, mesos.TASK_FAILED, mesos.TASK_ERROR:
			log.WithPrefix("taskman").
				WithFields(logrus.Fields{
					"taskId": mesosStatus.GetTaskID().Value,
					"state": mesosState.String(),
					"reason": mesosStatus.GetReason().String(),
					"source": mesosStatus.GetSource().String(),
					"message": mesosStatus.GetMessage(),
				}).
				Info("task inactive exception")
			taskIDValue := mesosStatus.GetTaskID().Value
			t := m.GetTask(taskIDValue)
			if  t != nil && t.IsLocked() {
				go m.updateTaskState(taskIDValue, "ERROR")
			}
		}

		// This will check if the task update is from a reconciliation, as well as whether the task
		// is in a state in which a mesos Kill call is possible.
		// Reconcilation tasks are not part of the taskman.roster
		if mesosStatus.GetReason().String() == "REASON_RECONCILIATION" &&
			(mesosState == mesos.TASK_STAGING ||
				mesosState == mesos.TASK_STARTING ||
				mesosState == mesos.TASK_RUNNING ||
				mesosState == mesos.TASK_KILLING ||
				mesosState == mesos.TASK_UNKNOWN) {
			killCall := calls.Kill(mesosStatus.TaskID.GetValue(), mesosStatus.AgentID.GetValue())
			calls.CallNoData(context.TODO(), m.schedulerState.cli, killCall)
		} else {
			// Enqueue task state update
			go m.updateTaskStatus(&mesosStatus)
		}
	case taskop.TaskStateMessage:
		go m.updateTaskState(tm.taskId, tm.state)
	// case event.KillTasks:
		// m.killTasks(tm.GetTaskIds())
	case taskop.ReleaseTasks:
		go m.releaseTasks(tm.GetEnvironmentId(), tm.GetTasks())
	}


	return nil
}

// This function should only be called from the SIGINT/SIGTERM handler
func (m *ManagerV2) EmergencyKillTasks(tasks Tasks) {
	for _, t := range tasks {
		killCall := calls.Kill(t.GetTaskId(), t.GetAgentId())
		err := calls.CallNoData(context.TODO(), m.schedulerState.cli, killCall)
		if err != nil {
			log.WithPrefix("termination").WithError(err).Error(fmt.Sprintf("Mesos couldn't kill task %s",t.GetTaskId()))
		}
	}
}