/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2025 CERN and copyright holders of ALICE O².
 * Author: Michal Tichak <michal.tichak@cern.ch>
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

package executable

import (
	"testing"
	"time"

	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("kubectl task test", func() {
	task := KubectlTask{}
	task.Tci = &common.TaskCommandInfo{}
	task.Tci.Arguments = []string{"exampletask.yaml"}
	task.configYaml = "exampletask.yaml"
	task.ti = &mesos.TaskInfo{Name: "exampletask"}
	When("starting and stoping the task", func() {
		It("should start and stop accordingly", func() {
			err := task.Launch()
			Expect(err).NotTo(HaveOccurred())

			// just so we can see monitoring tools show something
			time.Sleep(2 * time.Second)

			err = task.Kill()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	When("transitioning to running", func() {
		It("should transition", func() {
			task.configYaml = "exampletask.yaml"
			transitionMsg := &executorcmd.ExecutorCommand_Transition{}
			transitionMsg.Destination = "running"
			result := task.Transition(transitionMsg)
			Expect(result.ErrorString).To(BeEmpty())
		})
	})

	// this test expects there is a task with a status on cluster.
	When("geting current status", func() {
		It("should get proper status", func() {
			status, err := task.getTaskStatus()
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal("standby"))
		})
	})
})

func TestKubectlTask(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "KubectlTask suite")
}
