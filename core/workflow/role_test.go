package workflow

import (
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/sm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

func complexRoleTree() (root Role, leaves map[string]Role) {
	defaultState := sm.RUNNING
	call1 := &callRole{
		roleBase: roleBase{Name: "call1", state: SafeState{state: defaultState}},
		Traits:   task.Traits{Critical: true}}
	task1 := &taskRole{
		roleBase: roleBase{Name: "task1", state: SafeState{state: defaultState}},
		Traits:   task.Traits{Critical: true},
		Task:     &task.Task{},
	}
	agg1task_noncritical := &taskRole{
		roleBase: roleBase{Name: "agg1task_noncritical", state: SafeState{state: defaultState}},
		Traits:   task.Traits{Critical: false},
		Task:     nil}
	agg1task_critical := &taskRole{
		roleBase: roleBase{Name: "agg1task_critical", state: SafeState{state: defaultState}},
		Traits:   task.Traits{Critical: true},
		Task:     &task.Task{}}
	agg2task_noncritical := &taskRole{
		roleBase: roleBase{Name: "agg2task_noncritical", state: SafeState{state: defaultState}},
		Traits:   task.Traits{Critical: false},
		Task:     &task.Task{}}

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
					roleBase{Name: "agg2", state: SafeState{state: sm.INVARIANT}},
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
		const defaultState = sm.RUNNING
		const otherHealthyState = sm.CONFIGURED

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
						roleBase{Name: "agg1", state: SafeState{state: sm.INVARIANT}},
						aggregator{
							Roles: []Role{task1},
						},
					}
					LinkChildrenToParents(root)
					task1.UpdateState(otherHealthyState)
					Expect(root.GetState()).To(Equal(sm.INVARIANT))
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
					Expect(root.GetState()).To(Equal(sm.MIXED))
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
					Expect(root.GetState()).To(Equal(sm.MIXED))
				})
			})
			When("one of the critical tasks moves to ERROR", func() {
				It("should make the root role report ERROR", func() {
					leaves["agg1task_critical"].(*taskRole).UpdateState(sm.ERROR)
					Expect(root.GetState()).To(Equal(sm.ERROR))
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
					leaves["agg1task_noncritical"].(*taskRole).UpdateState(sm.ERROR)
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

	Describe("GetRoles", func() {
		When("getting the slice with roles of a complex tree", func() {
			var root Role
			BeforeEach(func() {
				root, _ = complexRoleTree()
			})

			It("should provide the expected role slice", func() {
				roles := root.GetRoles()
				roleExistsInSlice := func(roles []Role, roleName string) bool {
					for _, r := range roles {
						if r.GetName() == roleName {
							return true
						}
					}
					return false
				}

				// we expect that all roles within an aggregator role appear as if they were listed in the parent level
				// but aggregator in an aggregator should appear as an aggregator, while the include role is transparent.
				// FIXME: is the above really correct?
				Expect(len(roles)).To(Equal(4))
				Expect(roleExistsInSlice(roles, "call1"))
				Expect(roleExistsInSlice(roles, "task1"))
				Expect(roleExistsInSlice(roles, "agg1"))
				Expect(roleExistsInSlice(roles, "agg2"))
			})
		})
	})

	Describe("GetTasks", func() {

		When("getting the slice with roles for a complex tree", func() {
			var root Role
			BeforeEach(func() {
				root, _ = complexRoleTree()
			})
			It("should provide the expected tasks slice", func() {
				tasks := root.GetTasks()
				Expect(len(tasks)).To(Equal(3)) // there are 4 tasks in the tree, but is nil, thus 3
				for _, task := range tasks {
					Expect(task).NotTo(BeNil())
				}
				// fixme: task.Task has private fields and only task.newTaskForMesosOffer sets them at the moment,
				//  thus i am not able to verify if the correct contents are preserved
			})
		})
	})

	Describe("unmarshaling a YAML workflow template into a tree of roles", func() {
		Context("when the YAML template is a simple tree with a single task", func() {
			const yamlTemplate = `
name: root_role
description: description of the root role
defaults:
  default1: "true"
  default2: value of default2
  default3: false
vars:
  var1: "{{ default2 != 'none' }}"
  var2: "{{ default1 == 'true' && default3 == 'true' }}"
  var3: value of var3
roles:
  - name: "first_role"
    enabled: "{{ default1 == 'true' }}"
    vars:
      var3: "{{ default2 }}"
    constraints:
      - attribute: some_attribute
        value:  "{{ default2 }}"
    roles:
      - name: 'first_subrole'
        vars:
          var4: '{{default1 == "true" ? var3 : "value of var4"}}'
        task:
          load: name_of_task1
      - name: "second_subrole"
        enabled: 'false'
        roles:
          - name: "first_subsubrole"
            connect:
              - name: connection_name
                type: pull
                target: "{{ Up(2).Path }}.first_subrole:connection_name"
                rateLogging: "{{ default1 }}"
            task:
              load: name_of_task2
  - name: "second_role"
    call:
      func: testplugin.Noop()
      trigger: CONFIGURE
      timeout: 1s
      critical: false
`
			role := new(aggregatorRole)

			It("should unmarshal successfully", func() {
				err := yaml.Unmarshal([]byte(yamlTemplate), role)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should create a tree with a task role and a call role", func() {
				Expect(role.GetName()).To(Equal("root_role"))
				Expect(role.GetRoles()).To(HaveLen(2))

				Expect(role.GetRoles()[0].GetName()).To(Equal("first_role"))
				Expect(role.GetRoles()[0]).To(BeAssignableToTypeOf(&aggregatorRole{}))
				Expect(role.GetRoles()[0]).NotTo(BeAssignableToTypeOf(&callRole{}))
				Expect(role.GetRoles()[0].GetRoles()).To(HaveLen(2))

				Expect(role.GetRoles()[0].GetRoles()[0].GetName()).To(Equal("first_subrole"))
				Expect(role.GetRoles()[0].GetRoles()[0]).To(BeAssignableToTypeOf(&taskRole{}))

				Expect(role.GetRoles()[0].GetRoles()[1].GetName()).To(Equal("second_subrole"))
				Expect(role.GetRoles()[0].GetRoles()[1]).To(BeAssignableToTypeOf(&aggregatorRole{}))
				Expect(role.GetRoles()[0].GetRoles()[1].GetRoles()).To(HaveLen(1))

				Expect(role.GetRoles()[0].GetRoles()[1].GetRoles()[0].GetName()).To(Equal("first_subsubrole"))
				Expect(role.GetRoles()[0].GetRoles()[1].GetRoles()[0]).To(BeAssignableToTypeOf(&taskRole{}))

				Expect(role.GetRoles()[1].GetName()).To(Equal("second_role"))
				Expect(role.GetRoles()[1]).To(BeAssignableToTypeOf(&callRole{}))
				Expect(role.GetRoles()[1]).NotTo(BeAssignableToTypeOf(&taskRole{}))
			})
			It("should set the variables correctly", func() {
				Expect(role.GetDefaults().Raw()).To(HaveLen(3))
				Expect(role.GetRoles()[0].GetRoles()[0].GetVars().Raw()).To(HaveLen(1))
				Expect(role.GetRoles()[0].GetRoles()[1].GetVars().Raw()).To(HaveLen(0))
				Expect(role.GetRoles()[0].GetRoles()[0].ConsolidatedVarStack()).To(HaveLen(7))
				Expect(role.GetRoles()[0].GetRoles()[1].ConsolidatedVarStack()).To(HaveLen(6))
				Expect(role.GetRoles()[0].GetRoles()[0].GetVars().Raw()["var4"]).To(Equal("{{default1 == \"true\" ? var3 : \"value of var4\"}}"))
			})
		})

		Context("when the YAML template resembles readout-dataflow", func() {
			const yamlTemplate = `
name: !public readout-dataflow
description: !public "Main workflow template for ALICE data taking"
defaults:
  ###############################
  # General Configuration Panel
  ###############################
  dcs_enabled: !public
    value: "false"
    type: bool
    label: "DCS"
    description: "Enable/disable DCS SOR/EOR commands"
    widget: checkBox
    panel: General_Configuration
    index: 0
  dd_enabled: !public
    value: "true"
    type: bool
    label: "Data Distribution (FLP)"
    description: "Enable/disable Data Distribution components running on FLPs (StfBuilder and StfSender)"
    widget: checkBox
    panel: General_Configuration
    index: 1
  hosts: '["host1", "host2"]'
vars:
  auto_stop_enabled: "{{ auto_stop_timeout != 'none' }}"
  ddsched_enabled: "{{ epn_enabled == 'true' && dd_enabled == 'true' }}"
roles:
  ###########################
  # Start of CTP Readout role
  ###########################
  - name: "readout-ctp"
    enabled: "{{ ctp_readout_enabled == 'true' }}"
    vars:
      detector: "{{ctp_readout_enabled == 'true' ? inventory.DetectorForHost( ctp_readout_host ) : \"\" }}"
      readout_cfg_uri_standalone: "consul-ini://{{ consul_endpoint }}/o2/components/{{config.ResolvePath('readout/' + run_type + '/any/readout-standalone-' + ctp_readout_host)}}"
      readout_cfg_uri_stfb: "consul-ini://{{ consul_endpoint }}/o2/components/{{config.Resolve('readout', run_type, 'any', 'readout-stfb-' + ctp_readout_host)}}"
      dd_discovery_ib_hostname: "{{ ctp_readout_host }}-ib" # MUST be defined for all stfb and stfs
      # dpl_workflow is set to ctp_dpl_workflow
      dpl_workflow: "{{ util.PrefixedOverride( 'dpl_workflow', 'ctp' ) }}"
      dpl_command: "{{ util.PrefixedOverride( 'dpl_command', 'ctp' ) }}"
      stfs_shm_segment_size: "{{ ctp_stfs_shm_segment_size }}"
      it: "{{ ctp_readout_host }}"
    constraints:
      - attribute: machine_id
        value:  "{{ ctp_readout_host }}"
    roles:
      - name: "readout"
        vars:
          readout_cfg_uri: '{{dd_enabled == "true" ? readout_cfg_uri_stfb : readout_cfg_uri_standalone}}'
        task:
          load: readout-ctp
      - name: "data-distribution"
        enabled: "{{dd_enabled == 'true' && (qcdd_enabled == 'false' && minimal_dpl_enabled == 'false' && dpl_workflow == 'none' && dpl_command == 'none')}}"
        roles:
          # stfb-standalone not supported on CTP machine
          # if ctp_readout_enabled, we also assume stfb_standalone is false
          - name: "stfb"
            vars:
              dd_discovery_stfb_id: stfb-{{ ctp_readout_host }}-{{ uid.New() }} # must be defined for all stfb roles
            connect:
              - name: readout
                type: pull
                target: "{{ Up(2).Path }}.readout:readout"
                rateLogging: "{{ fmq_rate_logging }}"
            task:
              load: stfbuilder-senderoutput
  - name: host-{{ it }}
    for:
      range: "{{ hosts }}"
      var: it
    vars:
      detector: "{{ inventory.DetectorForHost( it ) }}"
      readout_cfg_uri_standalone: "consul-ini://{{ consul_endpoint }}/o2/components/{{config.ResolvePath('readout/' + run_type + '/any/readout-standalone-' + it)}}"
      readout_cfg_uri_stfb: "consul-ini://{{ consul_endpoint }}/o2/components/{{config.Resolve('readout', run_type, 'any', 'readout-stfb-' + it)}}"
      dd_discovery_ib_hostname: "{{ it }}-ib" # MUST be defined for all stfb and stfs
      # dpl_workflow is set to <detector>_dpl_workflow if such an override exists
      dpl_workflow: "{{ util.PrefixedOverride( 'dpl_workflow', strings.ToLower( inventory.DetectorForHost( it ) ) ) }}"
      dpl_command: "{{ util.PrefixedOverride( 'dpl_command', strings.ToLower( inventory.DetectorForHost( it ) ) ) }}"
    constraints:
      - attribute: machine_id
        value: "{{ it }}"
    roles:
      - name: "readout"
        vars:
          readout_cfg_uri: '{{dd_enabled == "true" ? readout_cfg_uri_stfb : readout_cfg_uri_standalone}}'
        task:
          load: readout
  - name: dcs
    enabled: "{{dcs_enabled == 'true'}}"
    defaults:
    ###############################
    # DCS Panel
    ###############################
      dcs_detectors: "{{ detectors }}"
      dcs_sor_parameters: !public
        value: "{}"
        type: string
        label: "Global SOR parameters"
        description: "additional parameters for the DCS SOR"
        widget: editBox
        panel: DCS
        index: 2
        visibleif: $$dcs_enabled === "true"
      dcs_eor_parameters: !public
        value: "{}"
        type: string
        label: "Global EOR parameters"
        description: "additional parameters for the DCS EOR"
        widget: editBox
        panel: DCS
        index: 3
        visibleif: $$dcs_enabled === "true"
    roles:
      - name: pfr
        call:
          func: dcs.PrepareForRun()
          trigger: before_CONFIGURE
          await: after_CONFIGURE
          timeout: "{{ dcs_pfr_timeout }}"
          critical: false
      - name: sor
        call:
          func: dcs.StartOfRun()
          trigger: before_START_ACTIVITY
          timeout: "{{ dcs_sor_timeout }}"
          critical: true
`
			role := new(aggregatorRole)

			It("should unmarshal successfully", func() {
				err := yaml.Unmarshal([]byte(yamlTemplate), role)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should create a complex tree correctly", func() {
				Expect(role.GetName()).To(Equal("readout-dataflow"))
				Expect(role.GetRoles()).To(HaveLen(2)) // GetRoles excludes iterator roles
				Expect(role.Roles).To(HaveLen(3))
				Expect(role.GetRoles()[0].GetName()).To(Equal("readout-ctp"))

				Expect(role.Roles[1].GetName()).To(Equal("host-{{ it }}"))
				Expect(role.Roles[1]).To(BeAssignableToTypeOf(&iteratorRole{}))

				Expect(role.GetRoles()[1]).To(BeAssignableToTypeOf(&aggregatorRole{}))
				Expect(role.GetRoles()[1].GetRoles()).To(HaveLen(2))
				Expect(role.GetRoles()[1].GetRoles()[0]).To(BeAssignableToTypeOf(&callRole{}))
			})

			It("should set the variables correctly", func() {
				Expect(role.GetDefaults().Raw()).To(HaveLen(3))
				Expect(role.GetRoles()[0].GetVars().Raw()).To(HaveLen(8))
				Expect(role.Roles[1].GetVars().Raw()).To(HaveLen(6))
				Expect(role.GetRoles()[1].GetVars().Raw()).To(HaveLen(0))

				// CTP subtree
				cvs, err := role.GetRoles()[0].GetRoles()[1].GetRoles()[0].ConsolidatedVarStack()
				Expect(err).NotTo(HaveOccurred())
				Expect(cvs).To(HaveLen(14))

				Expect(cvs["dd_enabled"]).To(Equal("true"))
				Expect(cvs["readout_cfg_uri_stfb"]).To(Equal("consul-ini://{{ consul_endpoint }}/o2/components/{{config.Resolve('readout', run_type, 'any', 'readout-stfb-' + ctp_readout_host)}}"))
				Expect(cvs["dd_discovery_ib_hostname"]).To(Equal("{{ ctp_readout_host }}-ib"))
				Expect(cvs["ddsched_enabled"]).To(Equal("{{ epn_enabled == 'true' && dd_enabled == 'true' }}"))

				// DCS subtree
				cvs, err = role.GetRoles()[1].GetRoles()[0].ConsolidatedVarStack()
				Expect(err).NotTo(HaveOccurred())
				Expect(cvs).To(HaveLen(8))

				Expect(cvs["dcs_enabled"]).To(Equal("false"))
				Expect(cvs["dcs_detectors"]).To(Equal("{{ detectors }}"))
				Expect(cvs["dcs_sor_parameters"]).To(Equal("{}"))
			})
		})
	})
})
