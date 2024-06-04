/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2024 CERN and copyright holders of ALICE O².
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

package fairmq

import (
	"testing"

	"github.com/AliceO2Group/Control/core/task/sm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("fairmq", func() {
	Describe("ODC to AliECS state mapping", func() {
		var (
			ecsState sm.State
		)
		When("transitioning ODC partition in sync with ECS environment", func() {
			BeforeEach(func() {
				ecsState = sm.UNKNOWN
			})
			It("should correctly report ECS states for an ordinary environment lifecycle", func() {
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZING DEVICE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZED", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("BINDING", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("BOUND", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("CONNECTING", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("DEVICE READY", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZING TASK", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("READY", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState = ToEcsState("RUNNING", ecsState)
				Expect(ecsState).To(Equal(sm.RUNNING))
				ecsState = ToEcsState("READY", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState = ToEcsState("RUNNING", ecsState) // start-stop-start
				Expect(ecsState).To(Equal(sm.RUNNING))
				ecsState = ToEcsState("READY", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState = ToEcsState("RESETTING TASK", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState = ToEcsState("DEVICE READY", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState := ToEcsState("RESETTING DEVICE", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("EXITING", ecsState)
				Expect(ecsState).To(Equal(sm.DONE))
			})
			It("should correctly report ECS states for an environment lifecycle that goes to ERROR during CONNECTING", func() {
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZING DEVICE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZED", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("BINDING", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("BOUND", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("CONNECTING", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("ERROR", ecsState)
				Expect(ecsState).To(Equal(sm.ERROR))
				ecsState = ToEcsState("EXITING", ecsState)
				Expect(ecsState).To(Equal(sm.DONE))
			})
			It("should correctly report ECS states for an environment lifecycle that goes to ERROR during READY", func() {
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZING DEVICE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZED", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("BINDING", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("BOUND", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("CONNECTING", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("DEVICE READY", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZING TASK", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("READY", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState = ToEcsState("ERROR", ecsState)
				Expect(ecsState).To(Equal(sm.ERROR))
				ecsState = ToEcsState("EXITING", ecsState)
				Expect(ecsState).To(Equal(sm.DONE))
			})
			It("should correctly report ECS states for an environment lifecycle that goes to ERROR during RUNNING", func() {
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZING DEVICE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZED", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("BINDING", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("BOUND", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("CONNECTING", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("DEVICE READY", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZING TASK", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("READY", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState = ToEcsState("RUNNING", ecsState)
				Expect(ecsState).To(Equal(sm.RUNNING))
				ecsState = ToEcsState("ERROR", ecsState)
				Expect(ecsState).To(Equal(sm.ERROR))
				ecsState = ToEcsState("EXITING", ecsState)
				Expect(ecsState).To(Equal(sm.DONE))
			})
			It("should correctly report ECS states for an environment lifecycle that unexpectedly resets at INITIALIZED", func() {
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZING DEVICE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZED", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState := ToEcsState("RESETTING DEVICE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("EXITING", ecsState)
				Expect(ecsState).To(Equal(sm.DONE))
			})
			It("should correctly report ECS states for an environment lifecycle that unexpectedly resets at DEVICE READY", func() {
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZING DEVICE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZED", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("BINDING", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("BOUND", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("CONNECTING", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("DEVICE READY", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState := ToEcsState("RESETTING DEVICE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("EXITING", ecsState)
				Expect(ecsState).To(Equal(sm.DONE))
			})
			It("should correctly report ECS states for an environment lifecycle that resets at READY without going to RUNNING", func() {
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZING DEVICE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZED", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("BINDING", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("BOUND", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("CONNECTING", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("DEVICE READY", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZING TASK", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("READY", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState = ToEcsState("RESETTING TASK", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState = ToEcsState("DEVICE READY", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState := ToEcsState("RESETTING DEVICE", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("EXITING", ecsState)
				Expect(ecsState).To(Equal(sm.DONE))
			})
			It("should correctly report ECS states for an environment lifecycle that sends invalid states", func() {
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZING DEVICE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("OK", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("INITIALIZED", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("", ecsState)
				Expect(ecsState).To(Equal(sm.UNKNOWN))
				ecsState = ToEcsState("READY", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
				ecsState = ToEcsState("ROLLING", ecsState)
				Expect(ecsState).To(Equal(sm.UNKNOWN))
				ecsState = ToEcsState("OK", ecsState)
				Expect(ecsState).To(Equal(sm.UNKNOWN))
				ecsState = ToEcsState("IDLE", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("MIXED", ecsState)
				Expect(ecsState).To(Equal(sm.STANDBY))
				ecsState = ToEcsState("READY", ecsState)
				Expect(ecsState).To(Equal(sm.CONFIGURED))
			})

		})
	})
})

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ODC state mapping test suite")
}
