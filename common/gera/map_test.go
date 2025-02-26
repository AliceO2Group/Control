/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020-2024 CERN and copyright holders of ALICE O².
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

package gera

import (
	"maps"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("hierarchical key-value store", func() {
	const (
		testPayloadDefaultsYAML1 = `
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
pdp_workflow_parameters: !public
    value: "QC,CALIB,GPU,CTF,EVENT_DISPLAY"
    type: string
    label: "Workflow parameters"
    description: "Comma-separated list of workflow parameters. Valid parameters are: QC,CALIB,GPU,CTF,EVENT_DISPLAY."
    widget: editBox
    panel: "EPNs_Workflows"
    index: 603
    rows: 10
user: flp
extra_env_vars: ""
`
		testPayloadDefaultsYAML2 = `
detector: ""
dpl_workflow: "{{ util.PrefixedOverride( 'dpl_workflow', 'ctp' ) }}"
dpl_command: "{{ util.PrefixedOverride( 'dpl_command', 'ctp' ) }}"
stfs_shm_segment_size: "{{ ctp_stfs_shm_segment_size }}"
it: "{{ ctp_readout_host }}"
user: "epn"
`
		testPayloadVarsYAML1 = `
auto_stop_enabled: "{{ auto_stop_timeout != 'none' }}"
ddsched_enabled: "{{ epn_enabled == 'true' && dd_enabled == 'true' }}"
odc_enabled: "{{ epn_enabled }}"
odc_topology_fullname: '{{ epn_enabled == "true" ? odc.GenerateEPNTopologyFullname() : "" }}'
`
		testPayloadVarsYAML2 = `
detector: "{{ctp_readout_enabled == 'true' ? inventory.DetectorForHost( ctp_readout_host ) : \"\" }}"
# dpl_workflow is set to ctp_dpl_workflow
dpl_workflow: "12345"
dpl_command: ""
stfs_shm_segment_size: "{{ ctp_stfs_shm_segment_size }}"
`
		testPayloadUserVarsYAML1 = `
ccdb_enabled: "true"
ccdb_host: ""
dd_enabled: "false"
pdp_workflow_parameters: ""
`
	)

	var (
		stringMap *WrapMap[string, string]
		err       error
	)

	customUnmarshalYAML := func(w Map[string, string], unmarshal func(interface{}) error) error {
		nodes := make(map[string]yaml.Node)
		err := unmarshal(&nodes)
		if err == nil {
			m := make(map[string]string)
			for k, v := range nodes {
				if v.Kind == yaml.ScalarNode {
					m[k] = v.Value
				} else if v.Kind == yaml.MappingNode && v.Tag == "!public" {
					type auxType struct {
						Value string
					}
					var aux auxType
					err = v.Decode(&aux)
					if err != nil {
						continue
					}
					m[k] = aux.Value
				}
			}

			wPtr := w.(*WrapMap[string, string])
			*wPtr = WrapMap[string, string]{
				theMap: m,
				parent: nil,
			}
		} else {
			wPtr := w.(*WrapMap[string, string])
			*wPtr = WrapMap[string, string]{
				theMap: make(map[string]string),
				parent: nil,
			}
		}
		return err
	}

	stringMap = MakeMap[string, string]()

	When("unmarshaling a YAML document into an empty Map", func() {
		BeforeEach(func() {
			stringMap = MakeMap[string, string]()
			Expect(stringMap).NotTo(BeNil())
		})
		It("should unmarshal correctly with the default unmarshaler if the YAML document is a flat key-value map", func() {
			err = yaml.Unmarshal([]byte(testPayloadDefaultsYAML2), stringMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(stringMap).NotTo(BeNil())
			value, ok := stringMap.Get("odc_enabled")
			Expect(value).To(BeEmpty())
			Expect(ok).NotTo(BeTrue())
			value, ok = stringMap.Get("detector")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(""))
			value, ok = stringMap.Get("dpl_workflow")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("{{ util.PrefixedOverride( 'dpl_workflow', 'ctp' ) }}"))
		})
		It("should fail to unmarshal with the default unmarshaler if the YAML document is a tree with YAML tags", func() {
			err = yaml.Unmarshal([]byte(testPayloadDefaultsYAML1), stringMap)
			Expect(err).To(HaveOccurred())
		})
		It("should unmarshal correctly with a custom unmarshaler if the YAML document is a flat key-value map", func() {
			stringMap = stringMap.WithUnmarshalYAML(customUnmarshalYAML)
			err = yaml.Unmarshal([]byte(testPayloadDefaultsYAML2), stringMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(stringMap).NotTo(BeNil())
			value, ok := stringMap.Get("odc_enabled")
			Expect(value).To(BeEmpty())
			Expect(ok).NotTo(BeTrue())
			value, ok = stringMap.Get("detector")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(""))
			value, ok = stringMap.Get("dpl_workflow")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("{{ util.PrefixedOverride( 'dpl_workflow', 'ctp' ) }}"))
		})
		It("should unmarshal correctly with a custom unmarshaler if the YAML document is a tree with YAML tags", func() {
			stringMap = stringMap.WithUnmarshalYAML(customUnmarshalYAML)
			err = yaml.Unmarshal([]byte(testPayloadDefaultsYAML1), stringMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(stringMap).NotTo(BeNil())
			value, ok := stringMap.Get("odc_enabled")
			Expect(value).To(BeEmpty())
			Expect(ok).NotTo(BeTrue())
			value, ok = stringMap.Get("dcs_enabled")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("false"))
			value, ok = stringMap.Get("dd_enabled")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("true"))
		})
	})
	When("marshaling into a YAML document from a Map", func() {
		BeforeEach(func() {
			stringMap = MakeMap[string, string]()
			Expect(stringMap).NotTo(BeNil())
			err = yaml.Unmarshal([]byte(testPayloadDefaultsYAML2), stringMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(stringMap).NotTo(BeNil())
		})

		It("should marshal correctly", func() {
			marshaledYAML, err := yaml.Marshal(stringMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(marshaledYAML).NotTo(BeEmpty())

			reUnmarshaledMap := make(map[string]string)
			err = yaml.Unmarshal(marshaledYAML, &reUnmarshaledMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(reUnmarshaledMap).NotTo(BeEmpty())

			unmarshaledTestPayloadDefaultsYAML2 := make(map[string]string)
			err = yaml.Unmarshal([]byte(testPayloadDefaultsYAML2), &unmarshaledTestPayloadDefaultsYAML2)
			Expect(err).NotTo(HaveOccurred())
			Expect(unmarshaledTestPayloadDefaultsYAML2).NotTo(BeEmpty())
			Expect(maps.Equal(reUnmarshaledMap, unmarshaledTestPayloadDefaultsYAML2)).To(BeTrue())
		})
	})
	When("accessing the structure's underlying map", func() {
		BeforeEach(func() {
			stringMap = MakeMap[string, string]()
			Expect(stringMap).NotTo(BeNil())
			err = yaml.Unmarshal([]byte(testPayloadDefaultsYAML2), stringMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(stringMap).NotTo(BeNil())
		})

		It("should correctly return a reference to the raw map", func() {
			rawMap := stringMap.Raw()
			Expect(rawMap).NotTo(BeNil())
			Expect(len(rawMap)).To(Equal(6))
			value, ok := rawMap["odc_enabled"]
			Expect(ok).NotTo(BeTrue())
			value, ok = rawMap["detector"]
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(""))
			value, ok = rawMap["dpl_workflow"]
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("{{ util.PrefixedOverride( 'dpl_workflow', 'ctp' ) }}"))
			value2, ok := stringMap.Get("dpl_workflow")
			Expect(ok).To(BeTrue())
			Expect(value2).To(Equal(value))
			rawMap["detector"] = "test"
			value, ok = stringMap.Get("detector")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("test"))
		})

		It("should correctly return a copy of the whole structure", func() {
			copy := stringMap.Copy()
			Expect(copy).NotTo(BeNil())
			Expect(copy == stringMap).NotTo(BeTrue())
			Expect(copy).To(Equal(stringMap))
			Expect(copy.Raw()).To(Equal(stringMap.Raw()))
			Expect(copy.Len()).To(Equal(stringMap.Len()))
			copy.Set("detector", "test")
			Expect(copy).NotTo(Equal(stringMap))
			Expect(copy.Raw()).NotTo(Equal(stringMap.Raw()))
			Expect(copy.Len()).To(Equal(stringMap.Len()))
			copy.Set("detector2", "test2")
			Expect(copy.Len()).To(Equal(stringMap.Len() + 1))
		})

		It("should correctly return a copy of the underlying map", func() {
			rawCopy := stringMap.RawCopy()
			Expect(rawCopy).NotTo(BeNil())
			Expect(len(rawCopy)).To(Equal(6))
			rawCopy["detector"] = "test"
			value, ok := stringMap.Get("detector")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(""))
		})
	})
	When("wrapping and unwrapping maps", func() {
		defaults1 := MakeMap[string, string]()
		Expect(defaults1).NotTo(BeNil())

		defaults1 = defaults1.WithUnmarshalYAML(customUnmarshalYAML)
		err = yaml.Unmarshal([]byte(testPayloadDefaultsYAML1), defaults1)
		Expect(err).NotTo(HaveOccurred())
		Expect(defaults1).NotTo(BeNil())

		defaults2 := MakeMap[string, string]()
		Expect(defaults2).NotTo(BeNil())

		err = yaml.Unmarshal([]byte(testPayloadDefaultsYAML2), defaults2)
		Expect(err).NotTo(HaveOccurred())
		Expect(defaults2).NotTo(BeNil())

		vars1 := MakeMap[string, string]()
		Expect(vars1).NotTo(BeNil())

		err = yaml.Unmarshal([]byte(testPayloadVarsYAML1), vars1)
		Expect(err).NotTo(HaveOccurred())
		Expect(vars1).NotTo(BeNil())

		vars2 := MakeMap[string, string]()
		Expect(vars2).NotTo(BeNil())

		err = yaml.Unmarshal([]byte(testPayloadVarsYAML2), vars2)
		Expect(err).NotTo(HaveOccurred())
		Expect(vars2).NotTo(BeNil())

		userVars1 := MakeMap[string, string]()
		Expect(userVars1).NotTo(BeNil())

		err = yaml.Unmarshal([]byte(testPayloadUserVarsYAML1), userVars1)
		Expect(err).NotTo(HaveOccurred())
		Expect(userVars1).NotTo(BeNil())

		var wrappedDefaults, wrappedVars, wrappedUserVars, wrappedAll Map[string, string]

		It("should wrap correctly", func() {
			wrappedDefaults = defaults2.Wrap(defaults1)
			Expect(wrappedDefaults).NotTo(BeNil())
			Expect(wrappedDefaults).To(Equal(defaults2))

			wrappedVars = vars2.Wrap(vars1)
			Expect(wrappedVars).NotTo(BeNil())
			Expect(wrappedVars).To(Equal(vars2))

			wrappedUserVars = userVars1
		})

		It("should unwrap correctly", func() {
			unwrapped := vars2.Unwrap()
			Expect(unwrapped).NotTo(BeNil())
			Expect(unwrapped).To(Equal(vars1))

			wrappedVars = vars2.Wrap(vars1)
			Expect(wrappedVars).NotTo(BeNil())
		})

		var flattenedDefaults, flattenedVars, flattenedUserVars, flattenedAll map[string]string

		It("should flatten correctly", func() {
			flattenedDefaults, err = wrappedDefaults.Flattened()
			Expect(err).NotTo(HaveOccurred())
			Expect(flattenedDefaults).To(Equal(map[string]string{
				"detector":                "",
				"dpl_workflow":            "{{ util.PrefixedOverride( 'dpl_workflow', 'ctp' ) }}",
				"dpl_command":             "{{ util.PrefixedOverride( 'dpl_command', 'ctp' ) }}",
				"stfs_shm_segment_size":   "{{ ctp_stfs_shm_segment_size }}",
				"it":                      "{{ ctp_readout_host }}",
				"dcs_enabled":             "false",
				"dd_enabled":              "true",
				"pdp_workflow_parameters": "QC,CALIB,GPU,CTF,EVENT_DISPLAY",
				"user":                    "epn",
				"extra_env_vars":          "",
			}))

			flattenedVars, err = wrappedVars.Flattened()
			Expect(err).NotTo(HaveOccurred())
			Expect(flattenedVars).To(Equal(map[string]string{
				"detector":              "{{ctp_readout_enabled == 'true' ? inventory.DetectorForHost( ctp_readout_host ) : \"\" }}",
				"dpl_workflow":          "12345",
				"dpl_command":           "",
				"stfs_shm_segment_size": "{{ ctp_stfs_shm_segment_size }}",
				"auto_stop_enabled":     "{{ auto_stop_timeout != 'none' }}",
				"ddsched_enabled":       "{{ epn_enabled == 'true' && dd_enabled == 'true' }}",
				"odc_enabled":           "{{ epn_enabled }}",
				"odc_topology_fullname": "{{ epn_enabled == \"true\" ? odc.GenerateEPNTopologyFullname() : \"\" }}",
			}))

			flattenedUserVars, err = wrappedUserVars.Flattened()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should wrap correctly from previously flattened maps", func() {
			wrappedAll = MakeMapWithMap(flattenedUserVars).Wrap(MakeMapWithMap(flattenedVars).Wrap(MakeMapWithMap(flattenedDefaults)))
			Expect(wrappedAll).NotTo(BeNil())
			flattenedAll, err = wrappedAll.Flattened()
			Expect(err).NotTo(HaveOccurred())

			Expect(flattenedAll).To(Equal(map[string]string{
				"detector":                "{{ctp_readout_enabled == 'true' ? inventory.DetectorForHost( ctp_readout_host ) : \"\" }}",
				"dpl_workflow":            "12345",
				"dpl_command":             "",
				"stfs_shm_segment_size":   "{{ ctp_stfs_shm_segment_size }}",
				"it":                      "{{ ctp_readout_host }}",
				"auto_stop_enabled":       "{{ auto_stop_timeout != 'none' }}",
				"ddsched_enabled":         "{{ epn_enabled == 'true' && dd_enabled == 'true' }}",
				"odc_enabled":             "{{ epn_enabled }}",
				"odc_topology_fullname":   "{{ epn_enabled == \"true\" ? odc.GenerateEPNTopologyFullname() : \"\" }}",
				"dcs_enabled":             "false",
				"dd_enabled":              "false",
				"pdp_workflow_parameters": "",
				"user":                    "epn",
				"extra_env_vars":          "",
				"ccdb_enabled":            "true",
				"ccdb_host":               "",
			}))
		})

		It("should flatten stack correctly", func() {
			flattenedStack, err := FlattenStack(wrappedDefaults, wrappedVars, wrappedUserVars)
			Expect(err).NotTo(HaveOccurred())
			Expect(flattenedAll).To(Equal(flattenedStack))
		})

		It("should flatten parent correctly", func() {
			flattenedParent, err := wrappedAll.FlattenedParent()
			Expect(err).NotTo(HaveOccurred())

			wrappedParents := MakeMapWithMap(flattenedVars).Wrap(MakeMapWithMap(flattenedDefaults))
			flattenedWrappedParents, err := wrappedParents.Flattened()
			Expect(flattenedParent).To(Equal(flattenedWrappedParents))
		})

		It("should wrap and flatten correctly", func() {
			flattenedParent, err := wrappedAll.FlattenedParent()
			Expect(err).NotTo(HaveOccurred())

			wrappedAndFlattenedParents, err := MakeMapWithMap(flattenedVars).WrappedAndFlattened(MakeMapWithMap(flattenedDefaults))
			Expect(err).NotTo(HaveOccurred())

			Expect(wrappedAndFlattenedParents).To(Equal(flattenedParent))
		})

		It("should correctly look into its hierarchy", func() {
			ok := wrappedDefaults.HierarchyContains(defaults1)
			Expect(ok).To(BeTrue())

			ok = wrappedDefaults.HierarchyContains(defaults2)
			Expect(ok).To(BeTrue())

			ok = wrappedDefaults.HierarchyContains(vars1)
			Expect(ok).NotTo(BeTrue())

			ok = wrappedDefaults.IsHierarchyRoot()
			Expect(ok).To(BeFalse())

			ok = defaults2.IsHierarchyRoot()
			Expect(ok).To(BeFalse())

			ok = defaults1.IsHierarchyRoot()
			Expect(ok).To(BeTrue())

			ok = userVars1.IsHierarchyRoot()
			Expect(ok).To(BeTrue())
		})

		It("should correctly perform KV Has operation", func() {
			Expect(wrappedAll.Has("odc_enabled")).To(BeTrue())
			Expect(wrappedAll.Has("ccdb_host")).To(BeTrue())
			Expect(wrappedAll.Has("detector")).To(BeTrue())
			Expect(wrappedAll.Has("it")).To(BeTrue())
			Expect(wrappedAll.Has("pdp_workflow_parameters")).To(BeTrue())
			Expect(wrappedAll.Has("invalid_key")).To(BeFalse())
			Expect(wrappedAll.Has("")).To(BeFalse())
		})

		It("should correctly perform KV Len operation", func() {
			Expect(defaults1.Len()).To(Equal(5))
			Expect(len(defaults2.Raw())).To(Equal(6))
			Expect(defaults2.Len()).To(Equal(10))
			Expect(wrappedVars.Len()).To(Equal(8))
			Expect(wrappedAll.Len()).To(Equal(16))
		})

		It("should correctly perform KV Get operations", func() {
			value, ok := wrappedAll.Get("odc_enabled")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("{{ epn_enabled }}"))

			value, ok = wrappedAll.Get("detector")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("{{ctp_readout_enabled == 'true' ? inventory.DetectorForHost( ctp_readout_host ) : \"\" }}"))

			value, ok = wrappedAll.Get("dpl_workflow")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("12345"))

			value, ok = wrappedAll.Get("ccdb_enabled")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("true"))

			value, ok = wrappedAll.Get("ccdb_host")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(""))

			value, ok = defaults1.Get("it")
			Expect(ok).To(BeFalse())
			Expect(value).To(BeEmpty())

			value, ok = defaults1.Get("pdp_workflow_parameters")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("QC,CALIB,GPU,CTF,EVENT_DISPLAY"))

			value, ok = wrappedAll.Get("pdp_workflow_parameters")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(""))
		})

		It("should correctly perform KV Set operations", func() {
			ok := userVars1.Set("detector", "")
			Expect(ok).To(BeTrue())

			ok = userVars1.Set("dpl_workflow", "someValue")
			Expect(ok).To(BeTrue())

			ok = defaults2.Set("user", "someone")
			Expect(ok).To(BeTrue())

			ok = defaults2.Set("dpl_command", "someCommand")
			Expect(ok).To(BeTrue())

			// We need to reflatten after setting new values
			flattenedDefaults, err = wrappedDefaults.Flattened()
			Expect(err).NotTo(HaveOccurred())

			flattenedVars, err = wrappedVars.Flattened()
			Expect(err).NotTo(HaveOccurred())

			flattenedUserVars, err = wrappedUserVars.Flattened()
			Expect(err).NotTo(HaveOccurred())

			wrappedAll = MakeMapWithMap(flattenedUserVars).Wrap(MakeMapWithMap(flattenedVars).Wrap(MakeMapWithMap(flattenedDefaults)))

			value, ok := wrappedAll.Get("detector")
			Expect(ok).To(BeTrue())
			Expect(value).To(BeEmpty())

			value, ok = wrappedAll.Get("dpl_workflow")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("someValue"))

			value, ok = wrappedAll.Get("user")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("someone"))

			value, ok = wrappedAll.Get("dpl_command")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(""))

			flattenedStack, err := FlattenStack(wrappedDefaults, wrappedVars, wrappedUserVars)
			Expect(err).NotTo(HaveOccurred())

			// ouch!!! this is a bug in the original code
			value, ok = flattenedStack["detector"]
			Expect(ok).To(BeTrue())
			Expect(value).To(BeEmpty())

			value, ok = flattenedStack["dpl_workflow"]
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("someValue"))

			value, ok = flattenedStack["user"]
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("someone"))

			value, ok = flattenedStack["dpl_command"]
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(""))
		})

		It("should correctly perform KV Del operations", func() {
			ok := userVars1.Del("detector")
			Expect(ok).To(BeTrue())

			ok = userVars1.Del("dpl_workflow")
			Expect(ok).To(BeTrue())

			ok = defaults2.Del("user")
			Expect(ok).To(BeTrue())

			ok = defaults2.Del("dpl_command")
			Expect(ok).To(BeTrue())

			// We need to reflatten after deleting values
			flattenedDefaults, err = wrappedDefaults.Flattened()
			Expect(err).NotTo(HaveOccurred())

			flattenedVars, err = wrappedVars.Flattened()
			Expect(err).NotTo(HaveOccurred())

			flattenedUserVars, err = wrappedUserVars.Flattened()
			Expect(err).NotTo(HaveOccurred())

			wrappedAll = MakeMapWithMap(flattenedUserVars).Wrap(MakeMapWithMap(flattenedVars).Wrap(MakeMapWithMap(flattenedDefaults)))

			value, ok := wrappedAll.Get("detector")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("{{ctp_readout_enabled == 'true' ? inventory.DetectorForHost( ctp_readout_host ) : \"\" }}"))

			value, ok = wrappedAll.Get("dpl_workflow")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("12345"))

			value, ok = wrappedAll.Get("user")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("flp"))

			value, ok = wrappedAll.Get("dpl_command")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(""))

			flattenedStack, err := FlattenStack(wrappedDefaults, wrappedVars, wrappedUserVars)
			Expect(err).NotTo(HaveOccurred())

			value, ok = flattenedStack["detector"]
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("{{ctp_readout_enabled == 'true' ? inventory.DetectorForHost( ctp_readout_host ) : \"\" }}"))

			value, ok = flattenedStack["dpl_workflow"]
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("12345"))

			value, ok = flattenedStack["user"]
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("flp"))

			value, ok = flattenedStack["dpl_command"]
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(""))
		})
	})
})

func TestMap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hierarchical Map Test Suite")
}
