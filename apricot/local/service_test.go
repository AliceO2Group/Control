package local

import (
	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"github.com/AliceO2Group/Control/configuration/cfgbackend"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TODO:
//  - with little effort this could be generalized to test also cacheproxy.service
//  - with some more effort, we should also test the same with consul backend, since we break the abstraction layer
//    in a few functions in service.go

var _ = Describe("local service", func() {

	var (
		svc *Service
		err error
	)

	Context("with YAML file backend", func() {
		BeforeEach(func() {
			svc, err = NewService("file://" + *tmpDir + "/" + serviceConfigFile)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should be of type *YamlSource", func() {
			_, ok := svc.src.(*cfgbackend.YamlSource)
			Expect(ok).To(Equal(true))
		})

		Describe("creating a new run number", func() {
			When("the run number does not exist yet", func() {
				It("should return number 1", func() {
					Expect(svc.NewRunNumber()).To(Equal(uint32(1)))
				})
				It("should return number 2 after number 1", func() {
					Expect(svc.NewRunNumber()).To(Equal(uint32(2)))
				})
			})
		})

		Describe("getting defaults", func() {
			var defaults map[string]string
			BeforeEach(func() {
				defaults = svc.GetDefaults()
			})
			When("the defaults are retrieved", func() {
				It("should contain the elements from the test YAML file at o2/runtime/aliecs/defaults", func() {
					Expect(defaults).To(HaveKeyWithValue("key1", "value1"))
				})
				It("should contain some prefilled key-values regardless of the information in the configuration backend", func() {
					Expect(defaults).To(HaveKey("core_hostname"))
				})
			})
		})

		Describe("getting vars", func() {
			var vars map[string]string
			BeforeEach(func() {
				vars = svc.GetVars()
			})
			When("the vars are retrieved", func() {
				It("should contain the elements from the test YAML file at o2/runtime/aliecs/vars", func() {
					Expect(vars).To(HaveKeyWithValue("key2", "value2"))
				})
			})
		})

		Describe("getting the list of detectors", func() {
			var detectors []string
			When("the full detector list is retrieved (incl. detector-like subsystems)", func() {
				BeforeEach(func() {
					detectors, err = svc.ListDetectors(true)
				})
				It("should contain the detectors in the test YAML file at o2/hardware/detectors", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(detectors).To(ContainElements("ABC", "DEF", "TRG", "XYZ"))
				})
			})
			When("the pure detector list is retrieved", func() {
				BeforeEach(func() {
					detectors, err = svc.ListDetectors(false)
				})
				It("should contain the detectors in the test YAML file at o2/hardware/detectors", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(detectors).To(ContainElements("ABC", "DEF", "XYZ"))
				})
			})
		})

		Describe("getting the list of hosts", func() {
			var hosts []string
			When("the list of hosts is retrieved for all detectors", func() {
				BeforeEach(func() {
					hosts, err = svc.GetHostInventory("")
				})
				It("should contain all the hosts for all the detectors in the test YAML file at o2/hardware/detectors", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(hosts).To(ContainElements("flp001", "flp002", "flp003", "flp100"))
				})
			})
			When("the list of hosts is retrieved for a valid detector", func() {
				BeforeEach(func() {
					hosts, err = svc.GetHostInventory("ABC")
				})
				It("should contain all the hosts for the ABC detector in the test YAML file at o2/hardware/detectors", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(hosts).To(ContainElements("flp001"))
				})
			})
			When("the list of hosts is retrieved for a non-existent detector", func() {
				BeforeEach(func() {
					hosts, err = svc.GetHostInventory("NOPE")
				})
				It("should produce an error", func() {
					Expect(err).To(HaveOccurred())
				})
			})
			When("the list of hosts is retrieved for an existing detector with no FLPs assigned", func() {
				BeforeEach(func() {
					hosts, err = svc.GetHostInventory("XYZ")
				})
				It("should return an empty list", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(hosts).To(BeEmpty())
				})
			})
		})

		Describe("getting the detectors FLP inventory", func() {
			var inventory map[string][]string
			When("the inventory is retrieved", func() {
				BeforeEach(func() {
					inventory, err = svc.GetDetectorsInventory()
				})
				It("should contain all the hosts for all the detectors in the test YAML file at o2/hardware/detectors", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(inventory).To(HaveKey("ABC"))
					Expect(inventory["ABC"]).To(ContainElements("flp001"))
					Expect(inventory).To(HaveKey("DEF"))
					Expect(inventory["DEF"]).To(ContainElements("flp002", "flp003"))
					Expect(inventory).To(HaveKey("TRG"))
					Expect(inventory["TRG"]).To(ContainElements("flp100"))
					Expect(inventory).To(HaveKey("XYZ"))
					Expect(inventory["XYZ"]).To(BeEmpty())
				})
			})
		})

		Describe("getting component configuration", func() {
			var (
				payload string
				query   *componentcfg.Query
				err     error
			)
			When("requesting an entry for a concrete run type and a concrete role", func() {
				It("should return the payload for the concrete run type and role", func() {
					query, err = componentcfg.NewQuery("qc/PHYSICS/role1/entry1")
					Expect(err).NotTo(HaveOccurred())
					payload, err = svc.GetComponentConfiguration(query)
					Expect(err).NotTo(HaveOccurred())
					Expect(payload).To(Equal("entry1 config PHYSICS role1"))
				})
			})
			When("requesting an entry for ANY run type and any role", func() {
				It("should return the payload for ANY/any", func() {
					query, err = componentcfg.NewQuery("qc/ANY/any/entry1")
					Expect(err).NotTo(HaveOccurred())
					payload, err = svc.GetComponentConfiguration(query)
					Expect(err).NotTo(HaveOccurred())
					Expect(payload).To(Equal("entry1 config ANY any"))
				})
			})
			When("requesting an entry in a subfolder", func() {
				It("should return the expected payload", func() {
					query, err = componentcfg.NewQuery("qc/ANY/any/sub/entry12")
					Expect(err).NotTo(HaveOccurred())
					payload, err = svc.GetComponentConfiguration(query)
					Expect(err).NotTo(HaveOccurred())
					Expect(payload).To(Equal("world"))
				})
			})
		})

		Describe("getting component configuration with last index", func() {
			var (
				query *componentcfg.Query
				err   error
			)
			When("a payload is retrieved", func() {
				It("should return an error, because file backend does not support last index", func() {
					query, err = componentcfg.NewQuery("qc/ANY/any/entry1")
					Expect(err).NotTo(HaveOccurred())
					_, _, err = svc.GetComponentConfigurationWithLastIndex(query)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("getting and processing component configuration", func() {
			var (
				payload string
				query   *componentcfg.Query
				err     error
			)
			When("requesting an entry which requires including another entry and inserting a variable", func() {
				It("should return the processed payload", func() {
					query, err = componentcfg.NewQuery("qc/ANY/any/entry10")
					Expect(err).NotTo(HaveOccurred())
					varStack := make(map[string]string)
					varStack["var1"] = "hello"
					payload, err = svc.GetAndProcessComponentConfiguration(query, varStack)
					Expect(err).NotTo(HaveOccurred())
					Expect(payload).To(Equal("hello world"))
				})
			})
			When("requesting an entry which requires including another entry in a different node", func() {
				It("should return the processed payload", func() {
					query, err = componentcfg.NewQuery("qc/ANY/any/entry12")
					Expect(err).NotTo(HaveOccurred())
					varStack := make(map[string]string)
					payload, err = svc.GetAndProcessComponentConfiguration(query, varStack)
					Expect(err).NotTo(HaveOccurred())
					Expect(payload).To(Equal("hello world"))
				})
			})
			When("requesting an entry which is in a subfolder", func() {
				It("should return the processed payload", func() {
					query, err = componentcfg.NewQuery("qc/ANY/any/sub/entry12")
					Expect(err).NotTo(HaveOccurred())
					varStack := make(map[string]string)
					payload, err = svc.GetAndProcessComponentConfiguration(query, varStack)
					Expect(err).NotTo(HaveOccurred())
					Expect(payload).To(Equal("world"))
				})
			})
		})

		Describe("resolving a query", func() {
			var (
				query    *componentcfg.Query
				resolved *componentcfg.Query
				err      error
			)

			When("resolving an ANY/any query which may match an ANY/any and a concrete one in the configuration tree", func() {
				It("should resolve to ANY/any query", func() {
					query, err = componentcfg.NewQuery("qc/ANY/any/entry1")
					Expect(err).NotTo(HaveOccurred())
					resolved, err = svc.ResolveComponentQuery(query)
					Expect(err).NotTo(HaveOccurred())
					Expect(resolved.RunType).To(Equal(apricotpb.RunType_ANY))
					Expect(resolved.RoleName).To(Equal("any"))
				})
			})
			When("resolving an ANY/any query while there is only PHYSICS/role1 entry in the configuration tree", func() {
				It("should fail to resolve", func() {
					query, err = componentcfg.NewQuery("qc/ANY/any/entry2")
					Expect(err).NotTo(HaveOccurred())
					_, err = svc.ResolveComponentQuery(query)
					Expect(err).To(HaveOccurred())
				})
			})
			When("resolving a concrete PHYSICS/role1 query which may match an ANY/any and that concrete one in the configuration tree", func() {
				It("should resolve to PHYSICS/role1 query", func() {
					query, err = componentcfg.NewQuery("qc/PHYSICS/role1/entry1")
					Expect(err).NotTo(HaveOccurred())
					resolved, err = svc.ResolveComponentQuery(query)
					Expect(err).NotTo(HaveOccurred())
					Expect(resolved.RunType).To(Equal(apricotpb.RunType_PHYSICS))
					Expect(resolved.RoleName).To(Equal("role1"))
				})
			})
			When("resolving a concrete PHYSICS/role1 query while there is ANY/any entry in the configuration tree", func() {
				It("should resolve to ANY/any query", func() {
					query, err = componentcfg.NewQuery("qc/PHYSICS/role1/entry11")
					Expect(err).NotTo(HaveOccurred())
					resolved, err = svc.ResolveComponentQuery(query)
					Expect(err).NotTo(HaveOccurred())
					Expect(resolved.RunType).To(Equal(apricotpb.RunType_ANY))
					Expect(resolved.RoleName).To(Equal("any"))
				})
			})
			When("resolving a query to an entry in a subfolder", func() {
				It("should resolve it correctly", func() {
					query, err = componentcfg.NewQuery("qc/ANY/any/sub/entry12")
					Expect(err).NotTo(HaveOccurred())
					resolved, err = svc.ResolveComponentQuery(query)
					Expect(err).NotTo(HaveOccurred())
					Expect(resolved.RunType).To(Equal(apricotpb.RunType_ANY))
					Expect(resolved.RoleName).To(Equal("any"))
					Expect(resolved.EntryKey).To(Equal("sub/entry12"))
				})
			})
		})

		Describe("getting raw json payload of everything under a node", func() {
			var (
				payload string
				err     error
			)
			When("retrieving a raw json payload", func() {
				It("should return a json with correct content and formatting", func() {
					payload, err = svc.RawGetRecursive("o2/components/qc/TECHNICAL")
					Expect(err).NotTo(HaveOccurred())
					Expect(payload).To(Equal("{\n\t\"any\": {\n\t\t\"entry\": \"config\"\n\t}\n}"))
				})
			})
		})

		Describe("importing component configuration", func() {
			var (
				importedPayload          string
				retrievedPayload         string
				existingComponentUpdated bool
				existingEntryUpdated     bool
				query                    *componentcfg.Query
				err                      error
			)
			When("we import a new configuration payload for a new component", func() {
				It("should be correctly stored", func() {
					query, err = componentcfg.NewQuery("reco/ANY/any/entry2")
					Expect(err).NotTo(HaveOccurred())
					importedPayload = "hello"
					existingComponentUpdated, existingEntryUpdated, err = svc.ImportComponentConfiguration(query, importedPayload, true)
					Expect(err).NotTo(HaveOccurred())
					Expect(existingComponentUpdated).To(BeFalse())
					Expect(existingEntryUpdated).To(BeFalse())
					retrievedPayload, err = svc.GetComponentConfiguration(query)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrievedPayload).To(Equal(importedPayload))
				})
			})
			When("we import a new configuration payload for a new component, but we do not set the newComponent flag", func() {
				It("should produce an error", func() {
					query, err = componentcfg.NewQuery("gpu/ANY/any/entry2")
					Expect(err).NotTo(HaveOccurred())
					importedPayload = "hello"
					existingComponentUpdated, existingEntryUpdated, err = svc.ImportComponentConfiguration(query, importedPayload, false)
					Expect(err).To(HaveOccurred())
				})
			})
			When("we import a new configuration entry for an existing component", func() {
				It("should be correctly stored", func() {
					query, err = componentcfg.NewQuery("qc/ANY/any/entry3")
					Expect(err).NotTo(HaveOccurred())
					importedPayload = "hello"
					existingComponentUpdated, existingEntryUpdated, err = svc.ImportComponentConfiguration(query, importedPayload, false)
					Expect(err).NotTo(HaveOccurred())
					Expect(existingComponentUpdated).To(BeTrue())
					Expect(existingEntryUpdated).To(BeFalse())
					retrievedPayload, err = svc.GetComponentConfiguration(query)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrievedPayload).To(Equal(importedPayload))
				})
			})
			When("we import a new payload to an existing entry for an existing component", func() {
				It("should be correctly stored", func() {
					query, err = componentcfg.NewQuery("qc/ANY/any/entry4")
					Expect(err).NotTo(HaveOccurred())
					importedPayload = "hello"
					existingComponentUpdated, existingEntryUpdated, err = svc.ImportComponentConfiguration(query, importedPayload, false)
					Expect(err).NotTo(HaveOccurred())

					importedPayload = "hello2"
					existingComponentUpdated, existingEntryUpdated, err = svc.ImportComponentConfiguration(query, importedPayload, false)
					Expect(err).NotTo(HaveOccurred())
					Expect(existingComponentUpdated).To(BeTrue())
					Expect(existingEntryUpdated).To(BeTrue())
					retrievedPayload, err = svc.GetComponentConfiguration(query)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrievedPayload).To(Equal(importedPayload))
				})
			})
			When("we import a new payload to a new entry which should go to a non-existing subfolder", func() {
				It("should be correctly stored", func() {
					query, err = componentcfg.NewQuery("qc/PHYSICS/role1/sub/entry2")
					Expect(err).NotTo(HaveOccurred())
					importedPayload = "sub hello"
					existingComponentUpdated, existingEntryUpdated, err = svc.ImportComponentConfiguration(query, importedPayload, false)
					Expect(err).NotTo(HaveOccurred())
					Expect(svc.src.Exists("o2/components/qc/PHYSICS/role1/sub/entry2")).To(BeTrue())

					retrievedPayload, err = svc.GetComponentConfiguration(query)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrievedPayload).To(Equal(importedPayload))
				})
			})

		})

		// TODO:
		//  GetDetectorForHost (currently not supporting yaml backend)
		//  GetDetectorsForHosts (currently not supporting yaml backend)
		//  GetCRUCardsForHost (currently not supporting yaml backend)
		//  GetEndpointsForCRUCard (currently not supporting yaml backend)
		//  GetRuntimeEntry (currently not supporting yaml backend)
		//  SetRuntimeEntry (currently not supporting yaml backend)
		//  GetRuntimeEntries (currently not supporting yaml backend)
		//  ListRuntimeEntries (currently not supporting yaml backend)
		// as far as i can tell, all of those could rather easily support yaml backend and share the code paths with
		// the consul backend, thus allow us to test them well without requiring a consul instance

		Describe("listing components", func() {
			var (
				components []string
				err        error
			)
			When("the list of components is retrieved", func() {
				It("should have the expected components", func() {
					components, err = svc.ListComponents()
					Expect(err).NotTo(HaveOccurred())
					Expect(components).To(ContainElements("qc"))
					// there might be also 'readout' in components, but it is put by another Describe node,
					// so i would not treat it as guaranteed to be there.
				})
			})
		})

		Describe("listing component entries", func() {
			var (
				query   *componentcfg.EntriesQuery
				entries []string
				err     error
			)
			When("the list of entries for a component is retrieved", func() {
				// fixme: the tested function does not perform run type and role name matching, it just takes
				//  everything under the provided component. i'm not sure if this is correct.
				It("should have the expected entries listed in the test yaml file", func() {
					query, err = componentcfg.NewEntriesQuery("qc/ANY/any")
					entries, err = svc.ListComponentEntries(query)
					Expect(err).NotTo(HaveOccurred())
					Expect(entries).To(ContainElements(
						"ANY/any/entry1",
						"ANY/any/entry10",
						"ANY/any/entry11",
						"ANY/any/entry12",
						"ANY/any/sub/entry12",
						"ANY/role1/entry1",
						"PHYSICS/role1/entry1",
						"PHYSICS/role1/entry2",
						"TECHNICAL/any/entry",
					))
					// there might be also a few other elements which are added in different Describe nodes,
					// which we do not look for, as they are not guaranteed to be there.
				})
			})
		})

		Describe("invalidating template cache", func() {
			var (
				payload string
				query   *componentcfg.Query
				err     error
			)
			When("requesting an entry after having invalidated cache", func() {
				It("should provide a valid entry", func() {
					query, err = componentcfg.NewQuery("qc/ANY/any/entry10")
					Expect(err).NotTo(HaveOccurred())
					varStack := make(map[string]string)
					varStack["var1"] = "hello"
					payload, err = svc.GetAndProcessComponentConfiguration(query, varStack)
					Expect(err).NotTo(HaveOccurred())
					Expect(payload).To(Equal("hello world"))

					svc.InvalidateComponentTemplateCache()
					payload, err = svc.GetAndProcessComponentConfiguration(query, varStack)
					Expect(err).NotTo(HaveOccurred())
					Expect(payload).To(Equal("hello world"))
				})
			})
		})
	})
})
