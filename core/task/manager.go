/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2019 CERN and copyright holders of ALICE O².
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
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/apricot"
	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/repos"
	"github.com/AliceO2Group/Control/core/task/sm"
	"github.com/AliceO2Group/Control/core/task/taskclass"
	"github.com/AliceO2Group/Control/core/task/taskop"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/AliceO2Group/Control/executor/executorutil"
	"github.com/mesos/mesos-go/api/v1/lib/extras/store"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/k0kubun/pp"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/calls"
	"github.com/sirupsen/logrus"
)

type KillTaskFunc func(*Task) error

const (
	TARGET_SEPARATOR_RUNE = ':'
	TARGET_SEPARATOR      = ":"
)

const TaskMan_QUEUE = 32768

type ResourceOffersOutcome struct {
	deployed     DeploymentMap
	undeployed   Descriptors
	undeployable Descriptors
}

type ResourceOffersDeploymentRequest struct {
	tasksToDeploy Descriptors
	envId         uid.ID
	outcomeCh     chan ResourceOffersOutcome
}

type Manager struct {
	deployMu sync.Mutex

	ctx            context.Context
	fidStore       store.Singleton
	AgentCache     AgentCache
	MessageChannel chan *TaskmanMessage

	classes *taskclass.Classes
	roster  *roster

	tasksToDeploy chan<- *ResourceOffersDeploymentRequest

	reviveOffersTrg  chan struct{}
	reviveOffersDone chan struct{}
	cq               *controlcommands.CommandQueue

	tasksLaunched int
	tasksFinished int

	schedulerState  *schedulerState
	internalEventCh chan<- event.Event
	ackKilledTasks  *safeAcks
	killTasksMu     sync.Mutex // to avoid races when attempting to kill the same tasks in different goroutines
}

func NewManager(shutdown func(), internalEventCh chan<- event.Event) (taskman *Manager, err error) {
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
			err = the.ConfSvc().SetRuntimeEntry("aliecs", "mesos_fid", v)
			if err != nil {
				log.WithField("error", err).Error("cannot write to configuration")
			}
			log.WithField("frameworkId", v).Debug("frameworkId")
			return nil
		}))

	// Set Framework ID from the configuration
	if fidValue, err := the.ConfSvc().GetRuntimeEntry("aliecs", "mesos_fid"); err == nil {
		store.SetOrPanic(fidStore)(fidValue)
	}

	taskman = &Manager{
		classes:         taskclass.NewClasses(),
		roster:          newRoster(),
		internalEventCh: internalEventCh,
	}
	schedState, err := NewScheduler(taskman, fidStore, shutdown)
	if err != nil {
		return nil, err
	}
	taskman.schedulerState = schedState
	taskman.cq = taskman.schedulerState.commandqueue
	taskman.tasksToDeploy = taskman.schedulerState.tasksToDeploy
	taskman.reviveOffersTrg = taskman.schedulerState.reviveOffersTrg
	taskman.reviveOffersDone = taskman.schedulerState.reviveOffersDone
	taskman.ackKilledTasks = newAcks()

	schedState.setupCli()

	return
}

// NewTaskForMesosOffer accepts a Mesos offer and a Descriptor and returns a newly
// constructed Task.
// This function should only be called by the Mesos scheduler controller when
// matching role requests with offers (matchRoles).
func (m *Manager) newTaskForMesosOffer(
	offer *mesos.Offer,
	descriptor *Descriptor,
	localBindMap channel.BindMap,
	executorId mesos.ExecutorID,
) (t *Task) {
	newId := uid.New().String()
	t = &Task{
		name:         fmt.Sprintf("%s#%s", descriptor.TaskClassName, newId),
		parent:       descriptor.TaskRole,
		className:    descriptor.TaskClassName,
		hostname:     offer.Hostname,
		agentId:      offer.AgentID.Value,
		offerId:      offer.ID.Value,
		taskId:       newId,
		properties:   gera.MakeMap[string, string]().Wrap(m.GetTaskClass(descriptor.TaskClassName).Properties),
		executorId:   executorId.Value,
		GetTaskClass: nil,
		localBindMap: nil,
		state:        sm.STANDBY,
		status:       INACTIVE,
	}
	t.GetTaskClass = func() *taskclass.Class {
		return m.GetTaskClass(t.className)
	}
	t.localBindMap = make(channel.BindMap)
	for k, v := range localBindMap {
		t.localBindMap[k] = v
	}
	return
}

func getTaskClassList(taskClassesRequired []string) (taskClassList []*taskclass.Class, err error) {
	repoManager := the.RepoManager()
	var yamlData []byte

	taskClassList = make([]*taskclass.Class, 0)

	for _, taskClass := range taskClassesRequired {
		taskClassString := strings.Split(taskClass, "@")
		taskClassFile := taskClassString[0] + ".yaml"
		var tempRepo repos.Repo
		_, tempRepo, err = repos.NewRepo(strings.Split(taskClassFile, "tasks")[0], repoManager.GetDefaultRevision(), repoManager.GetReposPath())
		if err != nil {
			return
		}
		repo := repoManager.GetAllRepos()[tempRepo.GetIdentifier()] // get IRepo pointer from RepoManager
		if repo == nil {                                            // should never end up here
			return nil, errors.New("getTaskClassList: repo not found for " + taskClass)
		}

		taskTemplatePath := repo.GetTaskTemplatePath(taskClassFile)
		yamlData, err = os.ReadFile(taskTemplatePath)
		if err != nil {
			return nil, err
		}
		taskClassStruct := taskclass.Class{}
		err = yaml.Unmarshal(yamlData, &taskClassStruct)
		if err != nil {
			return nil, fmt.Errorf("task template load error (template=%s): %w", taskTemplatePath, err)
		}

		var taskFilename, taskPath string
		taskRev := strings.Split(taskClass, "@")
		if len(taskRev) == 2 {
			taskPath = taskRev[0]
		} else {
			taskPath = taskClass
		}
		taskInfo := strings.Split(taskPath, "/tasks/")
		if len(taskInfo) == 1 {
			taskFilename = taskInfo[0]
		} else {
			taskFilename = taskInfo[1]
		}

		if taskClassStruct.Identifier.Name != taskFilename {
			err = fmt.Errorf("the name of the task template file (%s) and the name of the task (%s) don't match", taskFilename, taskClassStruct.Identifier.Name)
			return
		}

		taskClassStruct.Identifier.RepoIdentifier = repo.GetIdentifier()
		taskClassStruct.Identifier.Hash = repo.GetHash()
		taskClassList = append(taskClassList, &taskClassStruct)
	}
	return taskClassList, nil
}

func (m *Manager) GetState() string {
	return m.schedulerState.sm.Current()
}

func (m *Manager) GetFrameworkID() string {
	return m.schedulerState.GetFrameworkID()
}

func (m *Manager) removeInactiveClasses() {
	_ = m.classes.Do(func(classMap *map[string]*taskclass.Class) error {
		keys := make([]string, 0)

		taskClassCacheTTL := viper.GetDuration("taskClassCacheTTL")

		// push keys of classes that don't appear in roster any more into a slice
		for taskClassIdentifier, class := range *classMap {
			if class == nil {
				// don't really know what to do with a valid TCI but nil class
				continue
			}
			if time.Since(class.UpdatedTimestamp) < taskClassCacheTTL {
				// class is still fresh, skip
				continue
			}
			if len(m.roster.filteredForClass(taskClassIdentifier)) == 0 {
				keys = append(keys, taskClassIdentifier)
			}
		}

		// and delete them
		for _, k := range keys {
			delete(*classMap, k)
		}

		return nil
	})

	return
}

func (m *Manager) RemoveReposClasses(repoPath string) { // Currently unused
	utils.EnsureTrailingSlash(&repoPath)

	_ = m.classes.Do(func(classMap *map[string]*taskclass.Class) error {
		for taskClassIdentifier := range *classMap {
			if strings.HasPrefix(taskClassIdentifier, repoPath) &&
				len(m.roster.filteredForClass(taskClassIdentifier)) == 0 {
				delete(*classMap, taskClassIdentifier)
			}
		}
		return nil
	})

	return
}

func (m *Manager) RefreshClasses(taskClassesRequired []string) (err error) {
	log.WithField("taskClassesRequired", len(taskClassesRequired)).
		Debug("waiting to refresh task classes")
	defer utils.TimeTrackFunction(time.Now(), log.WithField("taskClassesRequired", len(taskClassesRequired)))

	m.deployMu.Lock()
	defer m.deployMu.Unlock()

	log.WithField("taskClassesRequired", len(taskClassesRequired)).
		Debug("cleaning up inactive task classes")

	m.removeInactiveClasses()

	log.WithField("taskClassesRequired", len(taskClassesRequired)).
		Debug("loading required task classes")

	var taskClassList []*taskclass.Class
	taskClassList, err = getTaskClassList(taskClassesRequired)
	if err != nil {
		return err
	}

	for _, class := range taskClassList {
		taskClassIdentifier := class.Identifier.String()
		// If it already exists we update, otherwise we add the new class
		m.classes.UpdateClass(taskClassIdentifier, class)
	}
	return
}

func (m *Manager) acquireTasks(envId uid.ID, taskDescriptors Descriptors) (err error) {
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
	claimableTasks := m.roster.filtered(func(task *Task) bool {
		return task.IsClaimable()
	})
	orphanTasks := m.roster.filtered(func(task *Task) bool {
		return !task.IsLocked() && !task.IsClaimable()
	})
	claimableTasksByHostname := claimableTasks.Grouped(func(task *Task) string {
		return task.hostname
	})
	orphanTasksByHostname := orphanTasks.Grouped(func(task *Task) string {
		return task.hostname
	})
	log.WithField("partition", envId.String()).
		Info("beginning of task acquisition and deployment for environment")

	for hostname, tasks := range claimableTasksByHostname {
		shortClassNames := make([]string, len(tasks))
		for i, t := range tasks {
			if tc := t.GetTaskClass(); tc != nil {
				shortClassNames[i] = tc.Identifier.Name
			} else {
				shortClassNames[i] = "unknown"
			}
		}
		log.WithField("partition", envId.String()).
			Infof("host %s has %d claimable tasks [%s]", hostname, len(tasks), strings.Join(shortClassNames, ", "))
	}

	for hostname, tasks := range orphanTasksByHostname {
		shortClassNames := make([]string, len(tasks))
		for i, t := range tasks {
			if tc := t.GetTaskClass(); tc != nil {
				shortClassNames[i] = tc.Identifier.Name
			} else {
				shortClassNames[i] = "unknown"
			}
		}
		log.WithField("partition", envId.String()).
			Warnf("host %s has %d orphan tasks (cleanup needed) [%s]", hostname, len(tasks), strings.Join(shortClassNames, ", "))
	}

	tasksToRun := make(Descriptors, 0)
	// TODO: also filter for tasks that satisfy attributes but are of wrong class,
	// to be torn down and mesosed.
	// tasksToTeardown := make([]TaskPtr, 0)
	tasksAlreadyRunning := make(DeploymentMap)

	if viper.GetBool("reuseUnlockedTasks") {
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
					if taskPtr.IsClaimable() && taskPtr.className == descriptor.TaskClassName {
						agentInfo := m.AgentCache.Get(mesos.AgentID{Value: taskPtr.agentId})
						taskClass, classFound := m.classes.GetClass(descriptor.TaskClassName)
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
						detector := ""
						parent := taskPtr.GetParent()
						if parent != nil {
							detector, ok = parent.GetUserVars().Get("detector")
							if !ok {
								detector = ""
							}
						}

						log.WithField("partition", envId.String()).
							WithField("detector", detector).
							WithField("class", descriptor.TaskClassName).
							WithField("task", taskPtr.className).
							WithField("taskHost", taskPtr.hostname).
							Warn("claiming existing unlocked task for incoming descriptor")
						break
					}
				}
			}

			if !claimed {
				// no task can be claimed for the current descriptor, we add it to the list of
				// new tasks to run
				tasksToRun = append(tasksToRun, descriptor)
			}
		}
	} else {
		// no task reuse at all, just run tasks for all descriptors
		tasksToRun = append(tasksToRun, taskDescriptors...)
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
	undeployedDescriptors := make(Descriptors, 0)
	undeployableDescriptors := make(Descriptors, 0)
	undeployedNonCriticalDescriptors := make(Descriptors, 0)
	undeployedCriticalDescriptors := make(Descriptors, 0)
	undeployableNonCriticalDescriptors := make(Descriptors, 0)
	undeployableCriticalDescriptors := make(Descriptors, 0)

	deployedTasks := make(DeploymentMap)
	if len(tasksToRun) > 0 {
		// Alright, so we have some descriptors whose requirements should be met with
		// new Tasks we're about to deploy here.
		// First we ask Mesos to revive offers and block until done, then upon receiving
		// the offers, we ask Mesos to run the required roles - if any.

		m.deployMu.Lock()

	DEPLOYMENT_ATTEMPTS_LOOP:
		for attemptCount := 0; attemptCount < MAX_ATTEMPTS_PER_DEPLOY_REQUEST; attemptCount++ {
			// We loop through the deployment attempts until we either succeed or
			// reach the maximum number of attempts. In the happy case, we should only
			// need to try once. A retry should only be necessary if the Mesos master
			// has not been able to provide the resources we need in the offers round
			// immediately after reviving.
			// We also keep track of the number of attempts made in the request object
			// so that the scheduler can decide whether to retry or not.
			// The request object is used to pass the tasks to deploy and the outcome
			// channel to the deployment routine.

			// reset variables in between retries
			deploymentSuccess = true

			outcomeCh := make(chan ResourceOffersOutcome)
			m.tasksToDeploy <- &ResourceOffersDeploymentRequest{
				tasksToDeploy: tasksToRun,
				envId:         envId,
				outcomeCh:     outcomeCh,
			} // buffered channel, does not block

			log.WithField("partition", envId).
				Debugf("scheduler has been sent request to deploy %d tasks", len(tasksToRun))

			timeReviveOffers := time.Now()
			timeDeployMu := time.Now()
			m.reviveOffersTrg <- struct{}{} // signal scheduler to revive offers
			<-m.reviveOffersDone            // we only continue when it's done
			utils.TimeTrack(timeReviveOffers, "acquireTasks: revive offers",
				log.WithField("tasksToRun", len(tasksToRun)).
					WithField("partition", envId))

			roOutcome := <-outcomeCh // blocks until a verdict from resourceOffers comes in

			utils.TimeTrack(timeDeployMu, "acquireTasks: deployment critical section",
				log.WithField("tasksToRun", len(tasksToRun)).
					WithField("partition", envId))

			deployedTasks = roOutcome.deployed
			undeployedDescriptors = roOutcome.undeployed
			undeployableDescriptors = roOutcome.undeployable

			log.WithField("tasks", deployedTasks).
				WithField("partition", envId).
				Debugf("resourceOffers is done, %d new tasks running", len(deployedTasks))

			if len(deployedTasks) != len(tasksToRun) {
				// ↑ Not all roles could be deployed. If some were critical,
				//   we cannot proceed with running this environment. Either way,
				//   we keep the roles running since they might be useful in the future.
				log.WithField("partition", envId).
					Errorf("environment deployment failure: %d tasks requested for deployment, but %d deployed", len(tasksToRun), len(deployedTasks))

				for _, desc := range undeployedDescriptors {
					if desc.TaskRole.GetTaskTraits().Critical == true {
						deploymentSuccess = false
						undeployedCriticalDescriptors = append(undeployedCriticalDescriptors, desc)
						printname := fmt.Sprintf("%s->%s", desc.TaskRole.GetPath(), desc.TaskClassName)
						log.WithField("partition", envId).
							Errorf("critical task deployment failure: %s", printname)
					} else {
						undeployedNonCriticalDescriptors = append(undeployedNonCriticalDescriptors, desc)
						printname := fmt.Sprintf("%s->%s", desc.TaskRole.GetPath(), desc.TaskClassName)
						log.WithField("partition", envId).
							Warnf("non-critical task deployment failure: %s", printname)
					}
				}

				for _, desc := range undeployableDescriptors {
					if desc.TaskRole.GetTaskTraits().Critical == true {
						deploymentSuccess = false
						undeployableCriticalDescriptors = append(undeployableCriticalDescriptors, desc)
						printname := fmt.Sprintf("%s->%s", desc.TaskRole.GetPath(), desc.TaskClassName)
						log.WithField("partition", envId).
							Errorf("critical task deployment impossible: %s", printname)
						go desc.TaskRole.UpdateStatus(UNDEPLOYABLE)
					} else {
						undeployableNonCriticalDescriptors = append(undeployableNonCriticalDescriptors, desc)
						printname := fmt.Sprintf("%s->%s", desc.TaskRole.GetPath(), desc.TaskClassName)
						log.WithField("partition", envId).
							Warnf("non-critical task deployment impossible: %s", printname)
					}
				}
			}

			if deploymentSuccess {
				// ↑ means all the required critical processes are now running,
				//   and we are ready to update the envId
				for taskPtr, descriptor := range deployedTasks {
					taskPtr.SetParent(descriptor.TaskRole)
					// Ensure everything is filled out properly
					if !taskPtr.IsLocked() {
						log.WithField("task", taskPtr.taskId).Warning("cannot lock newly deployed task")
						deploymentSuccess = false
					}
				}
				break DEPLOYMENT_ATTEMPTS_LOOP
			} else {
				log.WithField("partition", envId).
					Errorf("Attempt number %d/%d failed, but we are retrying...", attemptCount+1, MAX_ATTEMPTS_PER_DEPLOY_REQUEST)
			}
		}
	}

	m.deployMu.Unlock()

	if !deploymentSuccess {
		// While all the required roles are running, for some reason we
		// can't lock some of them, so we must roll back and keep them
		// unlocked in the roster.
		var deployedTaskIds []string
		for taskPtr := range deployedTasks {
			taskPtr.SetParent(nil)
			deployedTaskIds = append(deployedTaskIds, taskPtr.taskId)
		}

		err = TasksDeploymentError{
			tasksErrorBase:                     tasksErrorBase{taskIds: deployedTaskIds},
			failedNonCriticalDescriptors:       undeployedNonCriticalDescriptors,
			failedCriticalDescriptors:          undeployedCriticalDescriptors,
			undeployableNonCriticalDescriptors: undeployableNonCriticalDescriptors,
			undeployableCriticalDescriptors:    undeployableCriticalDescriptors,
		}
	}

	// Finally, we write to the roster. Point of no return!
	for taskPtr := range deployedTasks {
		m.roster.append(taskPtr)
	}
	if deploymentSuccess {
		for taskPtr := range deployedTasks {
			taskPtr.GetParent().SetTask(taskPtr)
		}
		for taskPtr, descriptor := range tasksAlreadyRunning {
			taskPtr.SetParent(descriptor.TaskRole)
			taskPtr.GetParent().SetTask(taskPtr)
		}
	}

	return
}

func (m *Manager) releaseTasks(envId uid.ID, tasks Tasks) error {
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

func (m *Manager) releaseTask(envId uid.ID, task *Task) error {
	if task == nil {
		return TaskNotFoundError{}
	}
	if task.IsLocked() && task.GetEnvironmentId() != envId {
		return TaskLockedError{taskErrorBase: taskErrorBase{taskId: task.name}, envId: envId}
	}

	task.SetParent(nil)

	return nil
}

func (m *Manager) configureTasks(envId uid.ID, tasks Tasks) error {
	notify := make(chan controlcommands.MesosCommandResponse)
	receivers, err := tasks.GetMesosCommandTargets()
	if err != nil {
		return err
	}

	if tasks == nil || len(tasks) == 0 {
		return fmt.Errorf("empty task list to configure for environment %s", envId.String())
	}

	// We fetch each task's local bindMap to generate a global bindMap for the whole Tasks slice,
	// i.e. a map of the paths of registered inbound channels and their ports.
	bindMap := make(channel.BindMap)
	for _, task := range tasks {
		if task.GetParent() == nil { // Crash reported here by Roberto 6/2022
			return fmt.Errorf("task %s on %s has nil parent, this should never happen", task.GetClassName(), task.GetHostname())
		}
		taskPath := task.GetParentRolePath()
		for inbChName, endpoint := range task.GetLocalBindMap() {
			var bindMapKey string
			if strings.HasPrefix(inbChName, "::") { // global channel alias
				bindMapKey = inbChName

				// deduplication
				if existingEndpoint, existsAlready := bindMap[bindMapKey]; existsAlready {
					if channel.EndpointEquals(existingEndpoint, endpoint) {
						// means somewhere something redefines the global channel alias,
						// but the endpoint is the same so there's no problem
						continue
					} else {
						return fmt.Errorf("workflow template contains illegal redefinition of global channel alias %s", bindMapKey)
					}
				}
			} else {
				bindMapKey = taskPath + TARGET_SEPARATOR + inbChName
			}
			bindMap[bindMapKey] = endpoint.ToTargetEndpoint(task.GetHostname())
		}
	}
	log.WithFields(logrus.Fields{"bindMap": pp.Sprint(bindMap), "envId": envId.String()}).
		Debug("generated inbound bindMap for environment configuration")

	src := sm.STANDBY.String()
	event := "CONFIGURE"
	dest := sm.CONFIGURED.String()
	args := make(controlcommands.PropertyMapsMap)
	args, err = tasks.BuildPropertyMaps(bindMap)
	if err != nil {
		return err
	}
	log.WithField("map", pp.Sprint(args)).
		WithField("partition", envId.String()).
		Debug("pushing configuration to tasks")

	cmd := controlcommands.NewMesosCommand_Transition(envId, receivers, src, event, dest, args)
	cmd.ResponseTimeout = 120 * time.Second // The default timeout is 90 seconds, but we need more time for the tasks to configure
	_ = m.cq.Enqueue(cmd, notify)

	response := <-notify
	close(notify)

	if response == nil {
		return errors.New("nil response")
	}

	if response.IsMultiResponse() {
		taskCriticalErrors := make([]string, 0)
		taskNonCriticalErrors := make([]string, 0)
		i := 0
		for k, v := range response.Errors() {
			task := m.GetTask(k.TaskId.Value)
			var taskDescription string
			if task != nil {
				tci := task.GetTaskCommandInfo()
				tciValue := "unknown command"
				if tci.Value != nil {
					tciValue = *tci.Value
				}

				taskDescription = fmt.Sprintf("task '%s' on %s (id %s, name %s) failed with error: %s", tciValue, task.GetHostname(), task.GetTaskId(), task.GetName(), v.Error())

			} else {
				taskDescription = fmt.Sprintf("unknown task (id %s) failed with error: %s", k.TaskId.Value, v.Error())
			}
			if task != nil && task.GetTraits().Critical {
				taskCriticalErrors = append(taskCriticalErrors, taskDescription)
			} else if task != nil && task.parent != nil && task.parent.GetTaskTraits().Critical {
				taskCriticalErrors = append(taskCriticalErrors, taskDescription)
			} else {
				taskNonCriticalErrors = append(taskNonCriticalErrors, taskDescription)
			}
			i++
		}

		if len(taskNonCriticalErrors) > 0 {
			log.WithField("partition", envId).
				Warnf("CONFIGURE could not complete for non-critical tasks, errors: %s", strings.Join(taskNonCriticalErrors, "; "))
		}
		if len(taskCriticalErrors) > 0 {
			return fmt.Errorf("CONFIGURE could not complete for critical tasks, errors: %s", strings.Join(taskCriticalErrors, "; "))
		}
		return nil
	} else {
		respError := response.Err()
		if respError != nil {
			errText := respError.Error()
			if len(strings.TrimSpace(errText)) != 0 {
				return errors.New(response.Err().Error())
			}
			// FIXME: improve error handling ↑
		}
	}

	return nil
}

func (m *Manager) transitionTasks(envId uid.ID, tasks Tasks, src string, event string, dest string, commonArgs controlcommands.PropertyMap) error {
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

	cmd := controlcommands.NewMesosCommand_Transition(envId, receivers, src, event, dest, args)
	_ = m.cq.Enqueue(cmd, notify)

	response := <-notify
	close(notify)

	if response == nil {
		return errors.New("unknown MesosCommand error: nil response received")
	}

	if response.IsMultiResponse() {
		taskCriticalErrors := make([]string, 0)
		taskNonCriticalErrors := make([]string, 0)
		i := 0
		for k, v := range response.Errors() {
			task := m.GetTask(k.TaskId.Value)
			var taskDescription string
			if task != nil {
				tci := task.GetTaskCommandInfo()
				tciValue := "unknown command"
				if tci.Value != nil {
					tciValue = *tci.Value
				}

				taskDescription = fmt.Sprintf("task '%s' on %s (id %s) failed with error: %s", tciValue, task.GetHostname(), task.GetTaskId(), v.Error())

			} else {
				taskDescription = fmt.Sprintf("unknown task (id %s) failed with error: %s", k.TaskId.Value, v.Error())
			}
			if task != nil && task.GetTraits().Critical {
				taskCriticalErrors = append(taskCriticalErrors, taskDescription)
			} else if task != nil && task.parent != nil && task.parent.GetTaskTraits().Critical {
				taskCriticalErrors = append(taskCriticalErrors, taskDescription)
			} else {
				taskNonCriticalErrors = append(taskNonCriticalErrors, taskDescription)
			}
			i++
		}

		if len(taskNonCriticalErrors) > 0 {
			log.WithField("partition", envId).
				Warnf("%s could not complete for non-critical tasks, errors: %s", event, strings.Join(taskNonCriticalErrors, "; "))
		}
		if len(taskCriticalErrors) > 0 {
			return fmt.Errorf("%s could not complete for critical tasks, errors: %s", event, strings.Join(taskCriticalErrors, "; "))
		}
		return nil
	} else {
		respError := response.Err()
		if respError != nil {
			errText := respError.Error()
			if len(strings.TrimSpace(errText)) != 0 {
				return errors.New(response.Err().Error())
			}
			// FIXME: improve error handling ↑
		}
	}

	return nil
}

func (m *Manager) TriggerHooks(envId uid.ID, tasks Tasks) error {
	if len(tasks) == 0 {
		return nil
	}

	notify := make(chan controlcommands.MesosCommandResponse)
	receivers, err := tasks.GetMesosCommandTargets()
	if err != nil {
		return err
	}

	cmd := controlcommands.NewMesosCommand_TriggerHook(envId, receivers)
	err = m.cq.Enqueue(cmd, notify)
	if err != nil {
		return err
	}

	response := <-notify
	close(notify)

	if response == nil {
		return errors.New("unknown MesosCommand error: nil response received")
	}

	respError := response.Err()
	if respError != nil {
		errText := respError.Error()
		if len(strings.TrimSpace(errText)) != 0 {
			return errors.New(response.Err().Error())
		}
		// FIXME: improve error handling ↑
	}

	return nil
}

func (m *Manager) GetTaskClass(name string) (b *taskclass.Class) {
	if m == nil {
		return
	}
	b, _ = m.classes.GetClass(name)
	return
}

func (m *Manager) TaskCount() int {
	if m == nil {
		return -1
	}
	return len(m.roster.getTasks())
}

func (m *Manager) GetTasks() Tasks {
	if m == nil {
		return nil
	}
	return m.roster.getTasks()
}

func (m *Manager) GetTask(id string) *Task {
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

func (m *Manager) updateTaskState(taskId string, state string) {
	taskPtr := m.roster.getByTaskId(taskId)
	if taskPtr == nil {
		log.WithField("taskId", taskId).
			WithField("state", state).
			Warn("attempted state update of task not in roster")
		return
	}

	st := sm.StateFromString(state)
	taskPtr.state = st
	taskPtr.safeToStop = false
	taskPtr.SendEvent(&event.TaskEvent{Name: taskPtr.GetName(), TaskID: taskId, State: state, Hostname: taskPtr.hostname, ClassName: taskPtr.GetClassName()})
	if taskPtr.GetParent() != nil {
		taskPtr.GetParent().UpdateState(st)
	}
}

func (m *Manager) updateTaskStatus(status *mesos.TaskStatus) {
	aid := status.GetAgentID()
	host := ""
	detector := ""
	var err error

	if aid != nil {
		aci := m.AgentCache.Get(*aid)
		if aci != nil {
			host = aci.Hostname
			detector, err = apricot.Instance().GetDetectorForHost(host)
			if err != nil {
				detector = ""
			}
		}
	}

	envId := executorutil.GetEnvironmentIdFromLabelerType(status)

	taskId := status.GetTaskID().Value
	taskPtr := m.roster.getByTaskId(taskId)
	if taskPtr == nil {
		if status != nil &&
			status.GetState() != mesos.TASK_FINISHED &&
			status.GetState() != mesos.TASK_FAILED {
			log.WithField("taskId", taskId).
				WithField("mesosStatus", status.GetState().String()).
				WithField("level", infologger.IL_Devel).
				WithField("status", status.GetState().String()).
				WithField("reason", status.GetReason().String()).
				WithField("detector", detector).
				WithField("partition", envId.String()).
				Warn("attempted status update of task not in roster")
		}
		if ack, ok := m.ackKilledTasks.getValue(taskId); ok {
			ack <- struct{}{}
			// close(ack) // It can even be left open?
		}

		return
	}

	switch st := status.GetState(); st {
	case mesos.TASK_RUNNING:
		cmdInfo := taskPtr.GetTaskCommandInfo()
		var (
			cmdStr      string
			controlMode string
		)
		if cmdInfo != nil {
			cmdStr = cmdInfo.GetValue()
			controlMode = cmdInfo.ControlMode.String()
		}
		log.WithField("taskId", taskId).
			WithField("name", taskPtr.GetName()).
			WithField("level", infologger.IL_Devel).
			WithField("cmd", cmdStr).
			WithField("controlmode", controlMode).
			WithField("detector", detector).
			WithField("partition", envId.String()).
			Debug("task active (received TASK_RUNNING event from executor)")
		taskPtr.status = ACTIVE
		if taskPtr.GetParent() != nil {
			taskPtr.GetParent().UpdateStatus(ACTIVE)
		}
	case mesos.TASK_DROPPED, mesos.TASK_LOST, mesos.TASK_KILLED, mesos.TASK_FAILED, mesos.TASK_ERROR, mesos.TASK_FINISHED:

		taskPtr.status = INACTIVE
		if taskPtr.GetParent() != nil {
			taskPtr.GetParent().UpdateStatus(INACTIVE)
		}
	}
	taskPtr.SendEvent(&event.TaskEvent{Name: taskPtr.GetName(), TaskID: taskId, Status: taskPtr.status.String(), Hostname: taskPtr.hostname, ClassName: taskPtr.GetClassName()})
}

// Kill all tasks outside an environment (all unlocked tasks)
func (m *Manager) Cleanup() (killed Tasks, running Tasks, err error) {
	toKill := m.roster.filtered(func(t *Task) bool {
		return !t.IsLocked()
	})

	killed, running, err = m.doKillTasks(toKill)
	return
}

// Kill a specific list of tasks.
// If the task list includes locked tasks, TaskNotFoundError is returned.
func (m *Manager) KillTasks(taskIds []string) (killed Tasks, running Tasks, err error) {
	taskCanBeKilledFilter := func(t *Task) bool {
		if t.IsLocked() || m.ackKilledTasks.contains(t.taskId) {
			return false
		}
		for _, id := range taskIds {
			if t.taskId == id {
				return true
			}
		}
		return false
	}

	if !m.killTasksMu.TryLock() {
		log.WithField("level", infologger.IL_Support).Warnf("Scheduling killing tasks was delayed until another goroutine finishes doing so")
		m.killTasksMu.Lock()
		log.WithField("level", infologger.IL_Support).Infof("Scheduling killing tasks is resumed")
	}
	// TODO: use grouping instead of 2 passes of filtering for performance
	toKill := m.roster.filtered(taskCanBeKilledFilter)

	if len(toKill) < len(taskIds) {
		unkillable := m.roster.filtered(func(t *Task) bool { return !taskCanBeKilledFilter(t) })
		log.WithField("taskIds", strings.Join(unkillable.GetTaskIds(), ", ")).
			Debugf("some tasks cannot be physically killed (already dead or being killed in another goroutine?), will instead only be removed from roster")
	}

	for _, id := range toKill.GetTaskIds() {
		m.ackKilledTasks.addAckChannel(id)
	}

	killed, running, err = m.doKillTasks(toKill)
	m.killTasksMu.Unlock()

	for _, id := range killed.GetTaskIds() {
		ack, ok := m.ackKilledTasks.getValue(id)
		if ok {
			<-ack
			m.ackKilledTasks.deleteKey(id)
		}
	}
	return
}

func (m *Manager) doKillTask(task *Task) error {
	return m.schedulerState.killTask(context.TODO(), task.GetMesosCommandTarget())
}

func (m *Manager) doKillTasks(tasks Tasks) (killed Tasks, running Tasks, err error) {
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

	// Remove from the roster the tasks we are going to kill
	// if we couldn't kill a task add it back to the roster
	m.roster.updateTasks(m.roster.filtered(func(task *Task) bool {
		return !tasks.Contains(func(t *Task) bool {
			return t.taskId == task.taskId
		})
	}))

	for _, task := range tasks.Filtered(func(task *Task) bool { return task.status == ACTIVE }) {
		e := m.doKillTask(task)
		if e != nil {
			log.WithError(e).
				WithField("taskId", task.taskId).
				Error("could not kill task")
			err = errors.New("could not kill some tasks")
			// task should be added back to the roster
			m.roster.append(task)
		} else {
			killed = append(killed, task)
		}
	}

	running = m.roster.getTasks()

	return
}

func (m *Manager) Start(ctx context.Context) {
	m.MessageChannel = make(chan *TaskmanMessage, TaskMan_QUEUE)

	m.schedulerState.Start(ctx)

	go func() {
		for {
			select {
			case taskmanMessage, ok := <-m.MessageChannel:
				if !ok { // if the channel is closed, we bail
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

func (m *Manager) stop() {
	close(m.MessageChannel)
}

func (m *Manager) handleMessage(tm *TaskmanMessage) error {
	if m == nil {
		return errors.New("manager is nil")
	}

	messageType := tm.GetMessageType()

	switch messageType {
	case taskop.AcquireTasks:
		go func() {
			err := m.acquireTasks(tm.GetEnvironmentId(), tm.GetDescriptors())
			if err != nil {
				log.WithError(err).
					WithField("level", infologger.IL_Devel).
					WithField("partition", tm.GetEnvironmentId().String()).
					Errorf("acquireTasks failed")
			}
		}()
	case taskop.ConfigureTasks:
		go func() {
			err := m.configureTasks(tm.GetEnvironmentId(), tm.GetTasks())
			m.internalEventCh <- event.NewTasksStateChangedEvent(tm.GetEnvironmentId(), tm.GetTasks().GetTaskIds(), err)
		}()
	case taskop.TransitionTasks:
		go func() {
			err := m.transitionTasks(tm.GetEnvironmentId(), tm.GetTasks(), tm.GetSource(), tm.GetEvent(), tm.GetDestination(), tm.GetArguments())
			m.internalEventCh <- event.NewTasksStateChangedEvent(tm.GetEnvironmentId(), tm.GetTasks().GetTaskIds(), err)
		}()
	case taskop.TaskStatusMessage:
		mesosStatus := tm.status
		mesosState := mesosStatus.GetState()
		switch mesosState {
		case mesos.TASK_FINISHED:
			log.WithPrefix("taskman").
				WithFields(logrus.Fields{
					"taskId":  mesosStatus.GetTaskID().Value,
					"state":   mesosState.String(),
					"reason":  mesosStatus.GetReason().String(),
					"source":  mesosStatus.GetSource().String(),
					"message": mesosStatus.GetMessage(),
					"level":   infologger.IL_Devel,
				}).
				WithField("partition", tm.GetEnvironmentId().String()). // fixme: this is empty!
				Info("task finished")
			taskIDValue := mesosStatus.GetTaskID().Value
			m.updateTaskState(taskIDValue, "DONE")
			m.tasksFinished++
		case mesos.TASK_LOST, mesos.TASK_KILLED, mesos.TASK_FAILED, mesos.TASK_ERROR:
			log.WithPrefix("taskman").
				WithFields(logrus.Fields{
					"taskId":  mesosStatus.GetTaskID().Value,
					"state":   mesosState.String(),
					"reason":  mesosStatus.GetReason().String(),
					"source":  mesosStatus.GetSource().String(),
					"message": mesosStatus.GetMessage(),
					"level":   infologger.IL_Devel,
				}).
				WithField("partition", tm.GetEnvironmentId().String()). // fixme: this is empty!
				Error("task inactive exception")
			taskIDValue := mesosStatus.GetTaskID().Value
			t := m.GetTask(taskIDValue)
			if t != nil && t.IsLocked() {
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
	case taskop.ReleaseTasks:
		go m.releaseTasks(tm.GetEnvironmentId(), tm.GetTasks())
	}

	return nil
}

func (m *Manager) HandleExecutorFailed(e *event.ExecutorFailedEvent) map[uid.ID]struct{} {
	// returns the set of environment ids affected by the failed executor
	if len(e.ExecutorId.Value) == 0 {
		return nil
	}

	tasksForFailedExecutor := m.roster.filtered(func(t *Task) bool {
		return t.executorId == e.ExecutorId.Value
	})

	envIdsForExecutor := make(map[uid.ID]struct{})

	for _, t := range tasksForFailedExecutor {
		envIdsForExecutor[t.GetEnvironmentId()] = struct{}{}
		t.executorId = "" // causes IsLocked() to become false for sure
		thisTask := t
		go func() {
			m.updateTaskState(thisTask.taskId, "ERROR")
			thisTask.status = INACTIVE
			taskParent := thisTask.GetParent()
			if taskParent != nil {
				thisTask.GetParent().UpdateStatus(INACTIVE)
			}
		}()
	}
	return envIdsForExecutor
}

func (m *Manager) HandleAgentFailed(e *event.AgentFailedEvent) map[uid.ID]struct{} {
	// returns the set of environment ids affected by the failed executor
	if len(e.AgentId.Value) == 0 {
		return nil
	}

	tasksForFailedExecutor := m.roster.filtered(func(t *Task) bool {
		return t.agentId == e.AgentId.Value
	})

	envIdsForExecutor := make(map[uid.ID]struct{})

	for _, t := range tasksForFailedExecutor {
		envIdsForExecutor[t.GetEnvironmentId()] = struct{}{}
		t.agentId = "" // causes IsLocked() to become false for sure
		thisTask := t
		go func() {
			m.updateTaskState(thisTask.taskId, "ERROR")
			thisTask.status = INACTIVE
			if taskParent := thisTask.GetParent(); taskParent != nil {
				taskParent.UpdateStatus(INACTIVE)
			}
		}()
	}
	return envIdsForExecutor
}

// This function should only be called from the SIGINT/SIGTERM handler
func (m *Manager) EmergencyKillTasks(tasks Tasks) {
	for _, t := range tasks {
		aidStr := t.GetAgentId()
		killCall := calls.Kill(t.GetTaskId(), aidStr)
		detector := ""
		var err error

		if len(aidStr) > 0 {
			aid := &mesos.AgentID{Value: aidStr}
			aci := m.AgentCache.Get(*aid)
			if aci != nil {
				host := aci.Hostname
				detector, err = apricot.Instance().GetDetectorForHost(host)
				if err != nil {
					detector = ""
				}
			}
		}

		err = calls.CallNoData(context.TODO(), m.schedulerState.cli, killCall)
		if err != nil {
			log.WithPrefix("termination").
				WithField("detector", detector).
				WithError(err).
				Error(fmt.Sprintf("Mesos couldn't kill task %s", t.GetTaskId()))
		}
	}
}
