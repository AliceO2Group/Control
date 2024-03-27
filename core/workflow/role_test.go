package workflow

import (
	"github.com/AliceO2Group/Control/core/task"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func complexRoleTree() (root Role, leaves map[string]Role) {
	defaultState := task.RUNNING
	call1 := &callRole{
		roleBase: roleBase{Name: "call1", state: SafeState{state: defaultState}},
		Traits:   task.Traits{Critical: true}}
	task1 := &taskRole{
		roleBase: roleBase{Name: "task1", state: SafeState{state: defaultState}},
		Traits:   task.Traits{Critical: true},
	}
	agg1task_noncritical := &taskRole{
		roleBase: roleBase{Name: "agg1task_noncritical", state: SafeState{state: defaultState}},
		Traits:   task.Traits{Critical: false}}
	agg1task_critical := &taskRole{
		roleBase: roleBase{Name: "agg1task_critical", state: SafeState{state: defaultState}},
		Traits:   task.Traits{Critical: true},
	}
	agg2task_noncritical := &taskRole{
		roleBase: roleBase{Name: "agg2task_noncritical", state: SafeState{state: defaultState}},
		Traits:   task.Traits{Critical: false}}

	root = &aggregatorRole{
		roleBase{Name: "root", state: SafeState{state: defaultState}},
		aggregator{
			Roles: []Role{
				call1,
				task1,
				&includeRole{
					aggregatorRole: aggregatorRole{
						roleBase{Name: "agg1", state: SafeState{state: defaultState}},
						aggregator{
							Roles: []Role{
								agg1task_noncritical,
								agg1task_critical,
							},
						},
					},
				},
				&aggregatorRole{
					roleBase{Name: "agg2", state: SafeState{state: task.INVARIANT}},
					aggregator{
						Roles: []Role{agg2task_noncritical},
					},
				},
			},
		},
	}

	LinkChildrenToParents(root)

	// helper map for easy leaf access
	leaves = make(map[string]Role)
	leaves["call1"] = call1
	leaves["task1"] = task1
	leaves["agg1task_noncritical"] = agg1task_noncritical
	leaves["agg1task_critical"] = agg1task_critical
	leaves["agg2task_noncritical"] = agg2task_noncritical
	return
}

var _ = Describe("role", func() {
	Describe("state propagation across roles", func() {
		const defaultState = task.RUNNING
		const otherHealthyState = task.CONFIGURED

		Context("simple aggregator role", func() {
			var root Role
			When("aggregator role has one critical task and the it changes to a different state", func() {
				It("should make the parent role transition to that state", func() {
					task1 := &taskRole{
						roleBase: roleBase{Name: "task1", state: SafeState{state: defaultState}},
						Traits:   task.Traits{Critical: true},
					}
					root = &aggregatorRole{
						roleBase{Name: "agg1", state: SafeState{state: defaultState}},
						aggregator{
							Roles: []Role{task1},
						},
					}
					LinkChildrenToParents(root)
					task1.UpdateState(otherHealthyState)
					Expect(root.GetState()).To(Equal(otherHealthyState))
				})
			})
			When("aggregator role has one non-critical task and the it changes to a different state", func() {
				It("should make the parent role state stay in INVARIANT", func() {
					task1 := &taskRole{
						roleBase: roleBase{Name: "task1", state: SafeState{state: defaultState}},
						Traits:   task.Traits{Critical: false},
					}
					// the aggregator role is expected to be in INVARIANT, because there are no critical roles inside,
					// thus nothing to affect the state of the role
					root = &aggregatorRole{
						roleBase{Name: "agg1", state: SafeState{state: task.INVARIANT}},
						aggregator{
							Roles: []Role{task1},
						},
					}
					LinkChildrenToParents(root)
					task1.UpdateState(otherHealthyState)
					Expect(root.GetState()).To(Equal(task.INVARIANT))
				})
			})
			When("one of critical tasks in an aggregator role changes to a different state", func() {
				// this could be changed by the requirements described in the ticket OCTRL-846
				It("should make the parent role transition to MIXED", func() {
					task1 := &taskRole{
						roleBase: roleBase{Name: "task1", state: SafeState{state: defaultState}},
						Traits:   task.Traits{Critical: true},
					}
					task2 := &taskRole{
						roleBase: roleBase{Name: "task1", state: SafeState{state: defaultState}},
						Traits:   task.Traits{Critical: true},
					}
					root = &aggregatorRole{
						roleBase{Name: "agg1", state: SafeState{state: defaultState}},
						aggregator{
							Roles: []Role{task1, task2},
						},
					}
					LinkChildrenToParents(root)
					task1.UpdateState(otherHealthyState)
					Expect(root.GetState()).To(Equal(task.MIXED))
				})
			})
			When("a non-critical task in an aggregator role (with another critical task) changes to a different state", func() {
				It("should not influence parent's state", func() {
					task1 := &taskRole{
						roleBase: roleBase{Name: "task1", state: SafeState{state: defaultState}},
						Traits:   task.Traits{Critical: false},
					}
					task2 := &taskRole{
						roleBase: roleBase{Name: "task1", state: SafeState{state: defaultState}},
						Traits:   task.Traits{Critical: true},
					}
					root = &aggregatorRole{
						roleBase{Name: "agg1", state: SafeState{state: defaultState}},
						aggregator{
							Roles: []Role{task1, task2},
						},
					}
					LinkChildrenToParents(root)
					task1.UpdateState(otherHealthyState)
					Expect(root.GetState()).To(Equal(defaultState))
				})
			})
		})

		Context("complex role tree", func() {
			var root Role
			var leaves map[string]Role
			BeforeEach(func() {
				root, leaves = complexRoleTree()
			})
			When("one of the critical tasks moves to a different state than the rest", func() {
				It("should make the root role report MIXED", func() {
					leaves["agg1task_critical"].(*taskRole).UpdateState(otherHealthyState)
					Expect(root.GetState()).To(Equal(task.MIXED))
				})
			})
			When("one of the critical tasks moves to ERROR", func() {
				It("should make the root role report ERROR", func() {
					leaves["agg1task_critical"].(*taskRole).UpdateState(task.ERROR)
					Expect(root.GetState()).To(Equal(task.ERROR))
				})
			})
			When("one of the non-critical tasks, which is aggregated with a critical task, moves to a different healthy state than the rest", func() {
				It("should not influence the root role state", func() {
					leaves["agg1task_noncritical"].(*taskRole).UpdateState(otherHealthyState)
					Expect(root.GetState()).To(Equal(defaultState))
				})
			})
			When("one of the non-critical tasks, which is alone in an aggregator role, moves to a different healthy state than the rest", func() {
				It("should not influence the root role state", func() {
					leaves["agg2task_noncritical"].(*taskRole).UpdateState(otherHealthyState)
					Expect(root.GetState()).To(Equal(defaultState))
				})
			})
			When("one of the non-critical tasks moves to ERROR", func() {
				It("should not influence the root role state", func() {
					leaves["agg1task_noncritical"].(*taskRole).UpdateState(task.ERROR)
					Expect(root.GetState()).To(Equal(defaultState))
				})
			})
			When("all critical tasks transition to another healthy state", func() {
				It("should move the root role state to the new healthy state", func() {
					for _, leaf := range leaves {
						if typedLeaf, ok := leaf.(Updatable); ok {
							typedLeaf.updateState(otherHealthyState)
						}
					}
					Expect(root.GetState()).To(Equal(otherHealthyState))
				})
			})
		})
	})
})
