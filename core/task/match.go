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
	"fmt"

	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/task/constraint"
	"github.com/AliceO2Group/Control/core/task/taskclass/port"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/resources"
)

type Wants struct {
	Cpu             float64
	Memory          float64
	StaticPorts     port.Ranges
	InboundChannels []channel.Inbound
}

type Limits struct {
	Cpu    float64
	Memory float64
}

func (m *Manager) GetLimitsForDescriptor(descriptor *Descriptor, envId uid.ID) (l *Limits) {
	taskClass, ok := m.classes.GetClass(descriptor.TaskClassName)
	if ok && taskClass != nil {
		l = &Limits{}
		if limits := taskClass.Limits; limits != nil {
			if limits.Cpu != nil {
				l.Cpu = *limits.Cpu
			}
			if limits.Memory != nil {
				l.Memory = *limits.Memory
			}
		}
	} else {
		log.WithPrefix("scheduler").
			WithField("partition", envId.String()).
			Warnf("missing task class %s", descriptor.TaskClassName)
	}
	return
}

// GetWantsForDescriptor matches between taskclass and taskmanager's classes
func (m *Manager) GetWantsForDescriptor(descriptor *Descriptor, envId uid.ID) (r *Wants, err error) {
	taskClass, ok := m.classes.GetClass(descriptor.TaskClassName)
	if ok && taskClass != nil {
		r = &Wants{}
		wants := taskClass.Wants
		if wants.Cpu != nil {
			r.Cpu = *wants.Cpu
		} else {
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				Warnf("missing CPU resource requirement for requested task class %s", descriptor.TaskClassName)
		}
		if wants.Memory != nil {
			r.Memory = *wants.Memory
		} else {
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				Warnf("missing memory resource requirement for requested task class %s", descriptor.TaskClassName)
		}
		if wants.Ports != nil {
			r.StaticPorts = make(port.Ranges, len(wants.Ports))
			copy(r.StaticPorts, wants.Ports)
		} else {
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				Tracef("missing ports resource requirement for requested task class %s", descriptor.TaskClassName)
		}
		r.InboundChannels = channel.MergeInbound(descriptor.RoleBind, taskClass.Bind)
	} else {
		err = fmt.Errorf("task class %s not found in manager", descriptor.TaskClassName)
	}
	return
}

type Resources mesos.Resources

func (r Resources) Satisfy(wants *Wants) bool {
	availCpu, ok := resources.CPUs(r...)
	if !ok || wants.Cpu > availCpu {
		return false
	}
	availMem, ok := resources.Memory(r...)
	if !ok || wants.Memory > float64(availMem) {
		return false
	}
	availPorts, ok := resources.Ports(r...)
	if !ok {
		return false
	}

	wantsStaticBuilder := resources.BuildRanges()
	for _, rng := range wants.StaticPorts {
		wantsStaticBuilder = wantsStaticBuilder.Span(rng.Begin, rng.End)
	}
	wantsStaticRanges := wantsStaticBuilder.Ranges.Sort().Squash()
	if wantsStaticRanges.Compare(availPorts) != -1 { // if wantsStaticRanges is NOT a subset of ports
		return false
	}

	wantsBindCount := len(wants.InboundChannels)
	// if total ports minus what we use for static ranges is LESS than the number of dynamic ports we'll need...
	if availPorts.Size()-wantsStaticRanges.Size() < uint64(wantsBindCount) {
		return false
	}

	// good job surviving til here, a winrar is you
	return true
}

func (m *Manager) BuildDescriptorConstraints(descriptors Descriptors) (cm map[*Descriptor]constraint.Constraints) {
	cm = make(map[*Descriptor]constraint.Constraints)
	for _, descriptor := range descriptors {
		taskClass, ok := m.classes.GetClass(descriptor.TaskClassName)
		if ok && taskClass != nil {
			cm[descriptor] = descriptor.RoleConstraints.MergeParent(taskClass.Constraints)
		} else {
			cm[descriptor] = descriptor.RoleConstraints
		}
	}
	return
}

/*
// BuildTasksForOffers takes in a list of Descriptors and Mesos offers, tries to find a complete
// match between them, and returns a slice of used offers, a slice of unused offers, a
// task.DeploymentMap of *Task-Deployment matches, and an error value.
// If err != nil, the other return values are still valid.
func (m *Manager) BuildTasksForOffers(descriptors Descriptors, offers []mesos.Offer) (
	offersUsed []mesos.Offer, offersLeft []mesos.Offer, tasksDeployed DeploymentMap, err error) {
	tasksDeployed = make(DeploymentMap)

	offersLeft = make([]mesos.Offer, len(offers))
	copy(offersLeft, offers)

	// for descriptor in descriptors:
	// 1) find the first match for o2roleclass and o2role
	//    NOTE: each mesos.Offer.Attributes list might have multiple entries for each name,
	//          however we assume all Attributes to be unique in the O² farm, and thus we
	//          only ever use the first occurrence.
	for _, descriptor := range descriptors {
		if index := m.indexOfOfferForDescriptor(offers, descriptor); index > -1 {
			offer := offersLeft[index]
			taskPtr := m.NewTaskForMesosOffer(&offer, &descriptor)
			tasksDeployed[taskPtr] = descriptor
			offersUsed = append(offersUsed, offer)
			// ↑ We are accepting an offer, so we must add it to the accepted list
			// ↓ and we must remove it from the offers list since it's just been claimed.
			offersLeft = append(offersLeft[:index], offersLeft[index + 1:]...)
		} else {
			msg := fmt.Sprintf("offer not found for some descriptors")
			log.WithFields(logrus.Fields{
				"role":      descriptor.TaskRole.GetPath(),
				"class":     descriptor.TaskClassName,
			}).Error(msg)
			err = errors.New(msg)
		}
	}
	return
}
*/
