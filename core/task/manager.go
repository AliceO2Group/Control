/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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

package task

import (
	"sync"
	"github.com/pborman/uuid"
	"github.com/AliceO2Group/Control/configuration"
	"errors"
	"fmt"
	"strings"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"gopkg.in/yaml.v2"
)

type Manager struct {
	AgentCache         AgentCache

	mu                 sync.Mutex
	classes            map[string]*TaskClass
	roster             Tasks

	cfgman             configuration.Configuration
	resourceOffersDone <-chan DeploymentMap
	tasksToDeploy      chan<- Descriptors
	reviveOffersTrg    chan struct{}
	cq                 *controlcommands.CommandQueue
}

func NewManager(cfgman configuration.Configuration,
                resourceOffersDone <-chan DeploymentMap,
                tasksToDeploy chan<- Descriptors,
                reviveOffersTrg chan struct{},
                cq *controlcommands.CommandQueue) (taskman *Manager) {
	taskman = &Manager{
		classes:            make(map[string]*TaskClass),
		roster:             make(Tasks, 0),
		cfgman:             cfgman,
		resourceOffersDone: resourceOffersDone,
		tasksToDeploy:      tasksToDeploy,
		reviveOffersTrg:    reviveOffersTrg,
		cq:                 cq,
	}
	return
}

// NewTaskForMesosOffer accepts a Mesos offer and a Descriptor and returns a newly
// constructed Task.
// This function should only be called by the Mesos scheduler controller when
// matching role requests with offers (matchRoles).
// The new role is not assigned to an environment and comes without a roleClass
// function, as those two are filled out later on by Manager.AcquireTasks.
func (m*Manager) NewTaskForMesosOffer(offer *mesos.Offer, descriptor *Descriptor, bindPorts map[string]uint64) (t *Task) {
	newId := uuid.NewUUID().String()
	t = &Task{
		name:         fmt.Sprintf("%s#%s", descriptor.TaskClassName, newId),
		parent:       descriptor.TaskRole,
		className:    descriptor.TaskClassName,
		hostname:     offer.Hostname,
		agentId:      offer.AgentID.Value,
		offerId:      offer.ID.Value,
		taskId:       newId,
		executorId:   uuid.NewUUID().String(),
		getTaskClass: nil,
		bindPorts:    nil,
	}
	t.getTaskClass = func() *TaskClass {
		return m.GetTaskClass(t.className)
	}
	t.bindPorts = make(map[string]uint64)
	for k, v := range bindPorts {
		t.bindPorts[k] = v
	}
	return
}

func (m *Manager) RefreshClasses() (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	yamlData, err := m.cfgman.GetRecursiveYaml("o2/control/tasks")
	if err != nil {
		return
	}

	taskClassesList := make([]*TaskClass, 0)
	err = yaml.Unmarshal(yamlData, &taskClassesList)
	if err != nil {
		return
	}

	for _, class := range taskClassesList {
		// If it already exists we update, otherwise we add the new class
		if _, ok := m.classes[class.Name]; ok {
			*m.classes[class.Name] = *class
		} else {
			m.classes[class.Name] = class
		}
	}
	return
}

func (m *Manager) AcquireTasks(envId uuid.Array, taskDescriptors Descriptors) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

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
					taskClass, classFound := m.classes[descriptor.TaskClassName]
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
		runningTasksForThisDescriptor := m.roster.Filtered(taskMatches)
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
			taskPtr.parent = descriptor.TaskRole
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
		var deployedTaskNames []string
		for taskPtr, _ := range deployedTasks {
			taskPtr.parent = nil
			deployedTaskNames = append(deployedTaskNames, taskPtr.name)
		}
		err = TasksDeploymentError{taskNames: deployedTaskNames}
	}

	// Finally, we write to the roster. Point of no return!
	for taskPtr, _ := range deployedTasks {
		m.roster = append(m.roster, taskPtr)
		taskPtr.parent.SetTask(taskPtr)
	}
	if deploymentSuccess {
		for taskPtr, descriptor := range tasksAlreadyRunning {
			taskPtr.parent = descriptor.TaskRole
			taskPtr.parent.SetTask(taskPtr)
		}
	}

	return
}

func (m *Manager) TeardownTasks(roleNames []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	//TODO: implement

	return nil
}

func (m *Manager) ReleaseTasks(envId uuid.Array, tasks Tasks) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range tasks {
		err := m.releaseTask(envId, task)
		if err != nil {
			switch err.(type) {
			case TaskAlreadyReleasedError:
				continue
			default:
				return err
			}
		}
	}

	return nil
}

func (m *Manager) releaseTask(envId uuid.Array, task *Task) error {
	if task == nil {
		return TaskNotFoundError{}
	}
	if task.IsLocked() && task.GetEnvironmentId() != envId {
		return TaskLockedError{taskErrorBase: taskErrorBase{taskName: task.name}, envId: envId}
	}

	task.parent = nil

	return nil
}

func (m *Manager) ConfigureTasks(envId uuid.Array, tasks Tasks) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	notify := make(chan controlcommands.MesosCommandResponse)
	receivers, err := m.receiversForTasks(tasks)
	if err != nil {
		return err
	}

	src := "STANDBY"
	event := "CONFIGURE"
	dest := "CONFIGURED"
	args := make(controlcommands.PropertyMapsMap)

	// FIXME: fetch configuration from Consul here and put it in args

	cmd := controlcommands.NewMesosCommand_Transition(receivers, src, event, dest, args)
	m.cq.Enqueue(cmd, notify)

	response := <- notify
	close(notify)

	errText := response.Err().Error()
	if len(strings.TrimSpace(errText)) != 0 {
		return errors.New(response.Err().Error())
	}

	// FIXME: improve error handling ↑

	return nil
}

func (m *Manager) TransitionTasks(envId uuid.Array, tasks Tasks, src string, event string, dest string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	notify := make(chan controlcommands.MesosCommandResponse)
	receivers, err := m.receiversForTasks(tasks)
	if err != nil {
		return err
	}

	args := make(controlcommands.PropertyMapsMap)

	cmd := controlcommands.NewMesosCommand_Transition(receivers, src, event, dest, args)
	m.cq.Enqueue(cmd, notify)

	response := <- notify
	close(notify)

	errText := response.Err().Error()
	if len(strings.TrimSpace(errText)) != 0 {
		return errors.New(response.Err().Error())
	}

	// FIXME: improve error handling ↑

	return nil
}

func (m *Manager) GetTaskClass(name string) (b *TaskClass) {
	if m == nil {
		return
	}
	b = m.classes[name]
	return
}

func (m *Manager) TaskCount() int {
	if m == nil {
		return -1
	}
	return len(m.roster)
}

func (m *Manager) GetTasks() Tasks {
	if m == nil {
		return nil
	}
	return m.roster
}

func (m *Manager) UpdateTask(status *mesos.TaskStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskId := status.GetTaskID().Value
	taskPtr := m.roster.GetByTaskId(taskId)
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
		taskPtr.parent.UpdateStatus(ACTIVE)
	case mesos.TASK_DROPPED, mesos.TASK_LOST, mesos.TASK_KILLED, mesos.TASK_FAILED, mesos.TASK_ERROR:
		taskPtr.parent.UpdateStatus(INACTIVE)
	}
}

// FIXME: we might want to make this a method of Tasks rather than of Manager
func (m *Manager) receiversForTasks(tasks Tasks) (receivers []controlcommands.MesosCommandTarget, err error) {
	receivers = make([]controlcommands.MesosCommandTarget, 0)
	for _, task := range tasks {
		if !task.IsLocked() {
			return nil, errors.New(fmt.Sprintf("task %s is not locked, cannot send control commands", task.GetName()))
		}
		receivers = append(receivers, controlcommands.MesosCommandTarget{
			AgentId: mesos.AgentID{
				Value: task.GetAgentId(),
			},
			ExecutorId: mesos.ExecutorID{
				Value: task.GetExecutorId(),
			},
		})
	}
	return
}

func (m *Manager) propertiesForTasks(tasks Tasks) (propMapMap controlcommands.PropertyMapsMap, err error) {
	propMapMap = make(controlcommands.PropertyMapsMap)
	for _, task := range tasks {
		if !task.IsLocked() {
			return nil, errors.New(fmt.Sprintf("task %s is not locked, cannot send control commands", task.GetName()))
		}
		receiver := controlcommands.MesosCommandTarget{
			AgentId: mesos.AgentID{
				Value: task.GetAgentId(),
			},
			ExecutorId: mesos.ExecutorID{
				Value: task.GetExecutorId(),
			},
		}
		propMapMap[receiver] = task.BuildPropertyMap()
	}
	return
}