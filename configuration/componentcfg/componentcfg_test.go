package componentcfg

import (
	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

var _ = Describe("Component configuration helpers", func() {
	Describe("IsInputValidComponentName", func() {
		When("a component does not have '/' characters", func() {
			It("should be considered as a valid component name", func() {
				Expect(IsInputValidComponentName("aZ0+@")).To(BeTrue())
			})
		})
		When("a component has a '/' character", func() {
			It("should be considered as an invalid component name", func() {
				Expect(IsInputValidComponentName("foo/bar")).To(BeFalse())
			})
		})
	})

	Describe("IsInputValidEntryName", func() {
		When("an entry does not have '/resolve' string at the end", func() {
			It("should be considered as a valid component name", func() {
				Expect(IsInputValidEntryName("aZ0+@")).To(BeTrue())
			})
		})
		When("an entry has a '/resolve' string at the end", func() {
			It("should be considered as an invalid entry name", func() {
				Expect(IsInputValidComponentName("foo/resolve")).To(BeFalse())
			})
		})
		When("an entry consists of directories, i.e. it has '/' characters", func() {
			It("should be considered as a valid entry name", func() {
				Expect(IsInputValidComponentName("foo/bar")).To(BeFalse())
			})
		})
	})

	Describe("GetComponentsMapFromKeysList", func() {
		When("given a list of keys with valid component names", func() {
			It("should return a map with the component names as keys", func() {
				keys := []string{
					"o2/components/component1/key1",
					"o2/components/component1/key2",
					"o2/components/component2/key1",
				}
				expected := map[string]bool{
					"component1": true,
					"component2": true,
				}
				Expect(GetComponentsMapFromKeysList(keys)).To(Equal(expected))
			})
		})

		When("given a list of keys with empty component names", func() {
			It("should return an empty map", func() {
				keys := []string{
					"o2/components//key1",
					"o2/components/",
				}
				expected := map[string]bool{}
				Expect(GetComponentsMapFromKeysList(keys)).To(Equal(expected))
			})
		})

		When("given an empty list of keys", func() {
			It("should return an empty map", func() {
				keys := []string{}
				expected := map[string]bool{}
				Expect(GetComponentsMapFromKeysList(keys)).To(Equal(expected))
			})
		})

		When("given a list of keys which are not in o2/components", func() {
			It("should return an empty map", func() {
				keys := []string{
					"o2/hardware/key",
				}
				expected := map[string]bool{}
				Expect(GetComponentsMapFromKeysList(keys)).To(Equal(expected))
			})
		})
	})

	Describe("GetEntriesMapOfComponentFromKeysList", func() {
		var (
			component = "qc"
			runtype   = apricotpb.RunType_PHYSICS // Assume 1 is a valid RunType
			rolename  = "testRole"
			keys      = []string{
				"o2/components/qc/PHYSICS/testRole/entry1",
				"o2/components/qc/PHYSICS/testRole/entry2",
				"o2/components/qc/PHYSICS/testRole/entry3",
			}
		)
		// fixme: it seems that this function does not support the "any" role any "ANY" run type. should this be fixed?

		When("keys list contains valid entries for the given component, runtype, and rolename", func() {
			It("should return a map with the entries matching the provided component", func() {
				expectedMap := map[string]bool{
					"entry1": true,
					"entry2": true,
					"entry3": true,
				}
				result := GetEntriesMapOfComponentFromKeysList(component, runtype, rolename, keys)
				Expect(result).To(Equal(expectedMap))
			})
		})

		When("keys list contains entries for a different component", func() {
			It("should not include those entries in the map", func() {
				keysWithDifferentComponent := append(keys, "o2/components/readout/PHYSICS/testRole/entry5")
				result := GetEntriesMapOfComponentFromKeysList(component, runtype, rolename, keysWithDifferentComponent)
				Expect(result).ToNot(HaveKey("entry5"))
			})
		})

		When("keys list contains entries for a different runtype", func() {
			It("should not include those entries in the map", func() {
				keysWithDifferentRuntype := append(keys, "o2/components/qc/TECHNICAL/testRole/entry6")
				result := GetEntriesMapOfComponentFromKeysList(component, runtype, rolename, keysWithDifferentRuntype)
				Expect(result).ToNot(HaveKey("entry6"))
			})
		})

		When("keys list contains entries for a different rolename", func() {
			It("should not include those entries in the map", func() {
				keysWithDifferentRolename := append(keys, "o2/components/qc/PHYSICS/differentRole/entry7")
				result := GetEntriesMapOfComponentFromKeysList(component, runtype, rolename, keysWithDifferentRolename)
				Expect(result).ToNot(HaveKey("entry7"))
			})
		})

		When("keys list contains an entry with an unexpected format", func() {
			It("should gracefully ignore malformed entries", func() {
				malformedKeys := append(keys, "o2/components/qc/PHYSICS/testRole")
				result := GetEntriesMapOfComponentFromKeysList(component, runtype, rolename, malformedKeys)
				Expect(result).ToNot(HaveKey(""))
			})
			// fixme: undefined behaviour:
			//  "o2/components/qc/PHYSICS/testRole//" will return a key "", is this correct?
		})
	})
})

func TestComponentCfg(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Component Configuration Test Suite")
}
