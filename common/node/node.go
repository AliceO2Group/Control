/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019 CERN and copyright holders of ALICE O².
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

package node

import (
	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/node/state"
)

type Handler func(n Node, e event.Event) error

type Path string

type Node interface {
	Run(<-chan event.Trigger, chan<- event.Event) error
	GetPath() Path
	GetName() string

	processEvent(e event.Event) error
}

type BaseNode struct {
	parent Node
	name string

	pending map[event.Trigger]struct{}
	handlers map[event.Type]Handler
	children []Node
	fromChildren <-chan event.Event
	toChildren chan<- event.Trigger

	state state.State
}

func (b *BaseNode) GetName() string {
	return b.name
}

func (b *BaseNode) GetPath() Path {
	if b.parent == nil {
		return Path(b.name)
	}
	return Path(string(b.parent.GetPath()) + "." + b.name)
}

func (b *BaseNode) processEvent(e event.Event) error {
	if handler, ok := b.handlers[e.GetType()]; ok && handler != nil {
		return handler(b, e)
	}
	return nil
}

func (b *BaseNode) Run(fromParent <-chan event.Trigger, toParent chan<- event.Event) error {

	for {
		if b.state.IsStable() {
			select {
			case e := <-fromParent:
				err := b.processEvent(e)
				if err != nil {
					toParent <- err //todo fix this with a TriggerHandlerError
				}
				b.pending[e] = struct{}{}
			}
		}
		select {
		case e := <- b.fromChildren:
			err := b.processEvent(e)
			if err != nil {
				toParent <- err
			}

			case t <- timeout:
		}
	}
}
