/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2025 CERN and copyright holders of ALICE O².
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

package workflow

import (
	"github.com/AliceO2Group/Control/core/task"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

var _ = Describe("safe status merge", func() {
	viper.Set("veryVerbose", true)
	When("Task and Call roles are used", func() {
		It("status should be overwritten", func() {
			{
				cr := &callRole{
					roleBase: roleBase{status: SafeStatus{status: task.PARTIAL}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				cr.status.merge(task.ACTIVE, cr)
				Expect(cr.GetStatus()).To(Equal(task.Status(task.ACTIVE)))
			}
			{
				tr := &taskRole{
					roleBase: roleBase{status: SafeStatus{status: task.PARTIAL}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				tr.status.merge(task.INACTIVE, tr)
				Expect(tr.GetStatus()).To(Equal(task.Status(task.INACTIVE)))
			}
		})
	})

	When("Aggregate role is merged with UNDEFINED status", func() {
		It("becomes UNDEFINED", func() {
			ar := &aggregatorRole{roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}}}
			ar.status.merge(task.UNDEFINED, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.UNDEFINED)))
		})
	})

	When("Empty aggregate role is merged with any status (except UNDEFINED) status", func() {
		It("becomes UNDEFINED", func() {
			ar := &aggregatorRole{roleBase: roleBase{status: SafeStatus{status: task.INACTIVE}}}
			ar.status.merge(task.ACTIVE, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.UNDEFINED)))
		})
	})

	When("Aggregate role is merged with the same status", func() {
		It("does not change it's status", func() {
			ar := &aggregatorRole{roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}}}
			ar.status.merge(task.ACTIVE, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.ACTIVE)))
		})
	})

	When("Aggregator role has ACTIVE subroles", func() {
		It("becomes ACTIVE", func() {
			ar := &aggregatorRole{
				roleBase: roleBase{status: SafeStatus{status: task.UNDEFINED}},
				aggregator: aggregator{
					[]Role{
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
					},
				},
			}
			// SafeStatus.Merge basically ignores status argument in a lot of cases, so it does not matter what we use here
			// and in following tests
			ar.status.merge(task.INACTIVE, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.ACTIVE)))
		})
	})

	When("Aggregator role has one ACTIVE subrole", func() {
		It("becomes PARTIAL from INACTIVE", func() {
			ar := &aggregatorRole{
				roleBase: roleBase{status: SafeStatus{status: task.UNDEFINED}},
				aggregator: aggregator{
					[]Role{
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.INACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
					},
				},
			}
			ar.status.merge(task.INACTIVE, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.PARTIAL)))
		})
	})

	When("Aggregator role has one ACTIVE subrole", func() {
		It("becomes PARTIAL from PARTIAL", func() {
			ar := &aggregatorRole{
				roleBase: roleBase{status: SafeStatus{status: task.UNDEFINED}},
				aggregator: aggregator{
					[]Role{
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.PARTIAL}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
					},
				},
			}
			ar.status.merge(task.INACTIVE, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.PARTIAL)))
		})
	})

	When("Aggregator role has one ACTIVE subrole", func() {
		It("becomes UNDEPLOYABLE from UNDEPLOYABLE", func() {
			ar := &aggregatorRole{
				roleBase: roleBase{status: SafeStatus{status: task.UNDEFINED}},
				aggregator: aggregator{
					[]Role{
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.UNDEPLOYABLE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
					},
				},
			}
			ar.status.merge(task.INACTIVE, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.UNDEPLOYABLE)))
		})
	})

	When("Aggregator role has one ACTIVE subrole", func() {
		It("becomes UNDEPLOYABLE from UNDEPLOYABLE", func() {
			ar := &aggregatorRole{
				roleBase: roleBase{status: SafeStatus{status: task.UNDEFINED}},
				aggregator: aggregator{
					[]Role{
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
						&callRole{
							roleBase: roleBase{status: SafeStatus{status: task.UNDEPLOYABLE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
					},
				},
			}
			ar.status.merge(task.INACTIVE, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.UNDEPLOYABLE)))
		})
	})

	When("Aggregator role has one ACTIVE subrole", func() {
		It("becomes UNDEFINED from UNDEFINED", func() {
			ar := &aggregatorRole{
				roleBase: roleBase{status: SafeStatus{status: task.UNDEFINED}},
				aggregator: aggregator{
					[]Role{
						&callRole{
							roleBase: roleBase{status: SafeStatus{status: task.UNDEFINED}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
					},
				},
			}
			ar.status.merge(task.INACTIVE, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.UNDEFINED)))
		})
	})

	When("Aggregator role has no critical roles", func() {
		It("becomes INVARIANT regardless of subrole statuses", func() {
			ar := &aggregatorRole{
				roleBase: roleBase{status: SafeStatus{status: task.UNDEFINED}},
				aggregator: aggregator{
					[]Role{
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: false},
						},
						&callRole{
							roleBase: roleBase{status: SafeStatus{status: task.UNDEPLOYABLE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: false},
						},
					},
				},
			}
			ar.status.merge(task.INACTIVE, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.INVARIANT)))
		})
	})

	When("Aggregator role has one ACTIVE critical subrole and one INACTIVE non critical role", func() {
		It("becomes ACTIVE", func() {
			ar := &aggregatorRole{
				roleBase: roleBase{status: SafeStatus{status: task.UNDEFINED}},
				aggregator: aggregator{
					[]Role{
						&callRole{
							roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.INACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: false},
						},
					},
				},
			}
			ar.status.merge(task.INACTIVE, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.ACTIVE)))
		})
	})

	When("Aggregator role has one ACTIVE critical subrole and one INACTIVE non critical role and one ACTIVE aggregator role", func() {
		It("becomes ACTIVE", func() {
			ar := &aggregatorRole{
				roleBase: roleBase{status: SafeStatus{status: task.UNDEFINED}},
				aggregator: aggregator{
					[]Role{
						&callRole{
							roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.INACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: false},
						},
						&aggregatorRole{
							roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}},
						},
					},
				},
			}
			ar.status.merge(task.INACTIVE, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.ACTIVE)))
		})
	})

	When("Aggregator role has one ACTIVE critical subrole and one INACTIVE non critical role and one INACTIVE aggregator role", func() {
		It("becomes PARTIAL", func() {
			ar := &aggregatorRole{
				roleBase: roleBase{status: SafeStatus{status: task.UNDEFINED}},
				aggregator: aggregator{
					[]Role{
						&callRole{
							roleBase: roleBase{status: SafeStatus{status: task.ACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
						},
						&taskRole{
							roleBase: roleBase{status: SafeStatus{status: task.INACTIVE}},
							Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: false},
						},
						&aggregatorRole{
							roleBase: roleBase{status: SafeStatus{status: task.INACTIVE}},
						},
					},
				},
			}
			ar.status.merge(task.INACTIVE, ar)
			Expect(ar.GetStatus()).To(Equal(task.Status(task.PARTIAL)))
		})
	})
})
