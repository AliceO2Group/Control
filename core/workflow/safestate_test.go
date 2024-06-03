package workflow

import (
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/sm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("safe state", func() {
	Describe("merging states", func() {
		When("the state we are updating belongs to a role which is a leaf in the tree", func() {
			It("should be overwritten", func() {
				cr := &callRole{
					roleBase: roleBase{state: SafeState{state: sm.CONFIGURED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				cr.state.merge(sm.RUNNING, cr)
				Expect(cr.GetState()).To(Equal(sm.RUNNING))

				tr := &taskRole{
					roleBase: roleBase{state: SafeState{state: sm.CONFIGURED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				tr.state.merge(sm.RUNNING, tr)
				Expect(tr.GetState()).To(Equal(sm.RUNNING))
			})
		})
		When("the state is healthy and is merged with ERROR state", func() {
			It("should become ERROR", func() {
				ar := &aggregatorRole{roleBase: roleBase{state: SafeState{state: sm.CONFIGURED}}}
				ar.state.merge(sm.ERROR, ar)
				Expect(ar.GetState()).To(Equal(sm.ERROR))
			})
		})
		When("the state is MIXED and is merged with ERROR state", func() {
			It("should become ERROR", func() {
				ar := &aggregatorRole{roleBase: roleBase{state: SafeState{state: sm.MIXED}}}
				ar.state.merge(sm.ERROR, ar)
				Expect(ar.GetState()).To(Equal(sm.ERROR))
			})
		})
		When("the role has at least two sub-roles, the state is ERROR and is merged with MIXED state", func() {
			It("should stay ERROR", func() {
				// because if we have two sub-roles A and B, where A is in ERROR and B enters MIXED,
				// it should not invalidate the ERROR state.
				subroles := make([]Role, 2)
				subroles[0] = &taskRole{
					roleBase: roleBase{state: SafeState{state: sm.ERROR}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				subroles[1] = &aggregatorRole{
					roleBase: roleBase{state: SafeState{state: sm.MIXED}},
				}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: sm.ERROR}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(sm.MIXED, ar)
				Expect(ar.GetState()).To(Equal(sm.ERROR))
			})
		})
		When("the role has one sub-role, the state is ERROR and is merged with MIXED state", func() {
			It("should become MIXED", func() {
				subroles := make([]Role, 1)
				subroles[0] = &aggregatorRole{roleBase: roleBase{state: SafeState{state: sm.MIXED}}}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: sm.ERROR}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(sm.MIXED, ar)
				Expect(ar.GetState()).To(Equal(sm.MIXED))
			})
		})
		When("the role has one sub-role, the state is ERROR and is merged with a healthy state", func() {
			It("should return to the healthy state", func() {
				subroles := make([]Role, 1)
				subroles[0] = &aggregatorRole{roleBase: roleBase{state: SafeState{state: sm.CONFIGURED}}}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: sm.ERROR}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(sm.CONFIGURED, ar)
				Expect(ar.GetState()).To(Equal(sm.CONFIGURED))
			})
		})
		When("the role has at least two sub-roles, state is ERROR and is merged with a healthy state", func() {
			It("should move to the corresponding healthy state", func() {
				subroles := make([]Role, 2)
				subroles[0] = &taskRole{
					roleBase: roleBase{state: SafeState{state: sm.CONFIGURED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				subroles[1] = &taskRole{
					roleBase: roleBase{state: SafeState{state: sm.CONFIGURED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: sm.ERROR}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(sm.CONFIGURED, ar)
				Expect(ar.GetState()).To(Equal(sm.CONFIGURED))
			})
		})
		When("the role has at least two sub-roles in MIXED, the state is ERROR and is merged with MIXED state", func() {
			It("should move to MIXED", func() {
				subroles := make([]Role, 2)
				subroles[0] = &taskRole{
					roleBase: roleBase{state: SafeState{state: sm.MIXED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				subroles[1] = &taskRole{
					roleBase: roleBase{state: SafeState{state: sm.MIXED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: sm.ERROR}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(sm.MIXED, ar)
				Expect(ar.GetState()).To(Equal(sm.MIXED))
			})
		})
		When("the role has at least two sub-roles, state is ERROR and is merged with a healthy state, but other sub-role stays in ERROR", func() {
			It("should stay in ERROR", func() {
				subroles := make([]Role, 2)
				subroles[0] = &taskRole{
					roleBase: roleBase{state: SafeState{state: sm.ERROR}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				subroles[1] = &taskRole{
					roleBase: roleBase{state: SafeState{state: sm.CONFIGURED}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: sm.ERROR}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(sm.STANDBY, ar)
				Expect(ar.GetState()).To(Equal(sm.ERROR))
			})
		})
		When("the role has at least two sub-roles, the state is healthy and one of critical sub-roles enters a different state than the other", func() {
			It("should enter MIXED", func() {
				subroles := make([]Role, 2)
				subroles[0] = &taskRole{
					roleBase: roleBase{state: SafeState{state: sm.RUNNING}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				subroles[1] = &taskRole{
					roleBase: roleBase{state: SafeState{state: sm.DONE}},
					Traits:   task.Traits{Trigger: "", Await: "", Timeout: "", Critical: true},
				}
				ar := &aggregatorRole{
					roleBase:   roleBase{state: SafeState{state: sm.RUNNING}},
					aggregator: aggregator{subroles},
				}
				ar.state.merge(sm.DONE, ar)
				Expect(ar.GetState()).To(Equal(sm.MIXED))
			})
		})
	})
})
