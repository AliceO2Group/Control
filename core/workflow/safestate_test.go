package workflow

import (
	"github.com/AliceO2Group/Control/core/task"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("safe state", func() {
	Describe("merging states", func() {
		When("the state we are updating belongs to a role which is a leaf in the tree", func() {
			It("should be overwritten", func() {
				cr := &callRole{
					roleBase: roleBase{state: SafeState{state: task.CONFIGURED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				cr.state.merge(task.RUNNING, cr)
				Expect(cr.GetState()).To(Equal(task.RUNNING))

				tr := &taskRole{
					roleBase: roleBase{state: SafeState{state: task.CONFIGURED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				tr.state.merge(task.RUNNING, tr)
				Expect(tr.GetState()).To(Equal(task.RUNNING))
			})
		})
		When("the state is healthy and is merged with ERROR state", func() {
			It("should become ERROR", func() {
				ar := &aggregatorRole{roleBase: roleBase{state: SafeState{state: task.CONFIGURED}}}
				ar.state.merge(task.ERROR, ar)
				Expect(ar.GetState()).To(Equal(task.ERROR))
			})
		})
		When("the state is MIXED and is merged with ERROR state", func() {
			It("should become ERROR", func() {
				ar := &aggregatorRole{roleBase: roleBase{state: SafeState{state: task.MIXED}}}
				ar.state.merge(task.ERROR, ar)
				Expect(ar.GetState()).To(Equal(task.ERROR))
			})
		})
		When("the role has at least two sub-roles, the state is ERROR and is merged with MIXED state", func() {
			It("should stay ERROR", func() {
				// because if we have two sub-roles A and B, where A is in ERROR and B enters MIXED,
				// it should not invalidate the ERROR state.
				subroles := make([]Role, 2)
				subroles[0] = &taskRole{
					roleBase: roleBase{state: SafeState{state: task.ERROR}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				subroles[1] = &aggregatorRole{
					roleBase: roleBase{state: SafeState{state: task.MIXED}},
				}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: task.ERROR}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(task.MIXED, ar)
				Expect(ar.GetState()).To(Equal(task.ERROR))
			})
		})
		When("the role has one sub-role, the state is ERROR and is merged with MIXED state", func() {
			It("should become MIXED", func() {
				subroles := make([]Role, 1)
				subroles[0] = &aggregatorRole{roleBase: roleBase{state: SafeState{state: task.MIXED}}}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: task.ERROR}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(task.MIXED, ar)
				Expect(ar.GetState()).To(Equal(task.MIXED))
			})
		})
		When("the role has one sub-role, the state is ERROR and is merged with a healthy state", func() {
			It("should return to the healthy state", func() {
				subroles := make([]Role, 1)
				subroles[0] = &aggregatorRole{roleBase: roleBase{state: SafeState{state: task.CONFIGURED}}}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: task.ERROR}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(task.CONFIGURED, ar)
				Expect(ar.GetState()).To(Equal(task.CONFIGURED))
			})
		})
		When("the role has at least two sub-roles, state is ERROR and is merged with a healthy state", func() {
			It("should move to the corresponding healthy state", func() {
				subroles := make([]Role, 2)
				subroles[0] = &taskRole{
					roleBase: roleBase{state: SafeState{state: task.CONFIGURED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				subroles[1] = &taskRole{
					roleBase: roleBase{state: SafeState{state: task.CONFIGURED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: task.ERROR}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(task.CONFIGURED, ar)
				Expect(ar.GetState()).To(Equal(task.CONFIGURED))
			})
		})
		When("the role has at least two sub-roles in MIXED, the state is ERROR and is merged with MIXED state", func() {
			It("should move to MIXED", func() {
				subroles := make([]Role, 2)
				subroles[0] = &taskRole{
					roleBase: roleBase{state: SafeState{state: task.MIXED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				subroles[1] = &taskRole{
					roleBase: roleBase{state: SafeState{state: task.MIXED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: task.ERROR}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(task.MIXED, ar)
				Expect(ar.GetState()).To(Equal(task.MIXED))
			})
		})
		When("the role has at least two sub-roles, state is ERROR and is merged with a healthy state, but other sub-role stays in ERROR", func() {
			It("should stay in ERROR", func() {
				subroles := make([]Role, 2)
				subroles[0] = &taskRole{
					roleBase: roleBase{state: SafeState{state: task.ERROR}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				subroles[1] = &taskRole{
					roleBase: roleBase{state: SafeState{state: task.CONFIGURED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: task.ERROR}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(task.STANDBY, ar)
				Expect(ar.GetState()).To(Equal(task.ERROR))
			})
		})
		When("the role has at least two sub-roles, the state is healthy and one of critical sub-roles enters a different state than the other", func() {
			It("should enter MIXED", func() {
				subroles := make([]Role, 2)
				subroles[0] = &taskRole{
					roleBase: roleBase{state: SafeState{state: task.RUNNING}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				subroles[1] = &taskRole{
					roleBase: roleBase{state: SafeState{state: task.DONE}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: task.RUNNING}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(task.DONE, ar)
				Expect(ar.GetState()).To(Equal(task.MIXED))
			})
		})
	})
})
