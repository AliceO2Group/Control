package integration_test

import (
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/integration/testplugin"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin interface", func() {
	var (
		p        integration.Plugin
		varStack map[string]string
	)

	DoPluginTests := func() {
		It("should successfully create a test plugin instance", func() {
			Expect(p).NotTo(BeNil())
		})

		Context("when getting the plugin name", func() {
			It("should return the correct name", func() {
				Expect(p.GetName()).To(Equal("testplugin"))
			})
		})

		Context("when getting the plugin pretty name", func() {
			It("should return the correct pretty name", func() {
				Expect(p.GetPrettyName()).To(Equal("Test plugin"))
			})
		})

		Context("when getting the plugin endpoint", func() {
			It("should return an empty endpoint", func() {
				Expect(p.GetEndpoint()).To(BeEmpty())
			})
		})

		Context("when getting the plugin connection state", func() {
			It("should return the correct connection state", func() {
				Expect(p.GetConnectionState()).To(Equal("READY"))
			})
		})

		Context("when getting the plugin data", func() {
			It("should return the correct data", func() {
				Expect(p.GetData(nil)).To(BeEmpty())
			})
		})

		Context("when getting the plugin environments data", func() {
			It("should return the correct data", func() {
				Expect(p.GetEnvironmentsData(nil)).To(BeNil())
			})
		})

		Context("when getting the plugin environments short data", func() {
			It("should return the correct data", func() {
				Expect(p.GetEnvironmentsShortData(nil)).To(BeNil())
			})
		})

		Context("when initializing the plugin", func() {
			It("should not return an error", func() {
				Expect(p.Init("")).To(Succeed())
			})
		})

		Context("when getting the plugin object stack", func() {
			It("should return an empty structure", func() {
				Expect(p.ObjectStack(nil, nil)).To(BeEmpty())
			})
		})

		Context("when getting the plugin call stack", func() {
			It("should contain a Noop function", func() {
				Expect(p.CallStack(&callable.Call{
					VarStack: varStack,
				})).To(HaveKey("Noop"))
			})

			It("should contain a Test function", func() {
				Expect(p.CallStack(&callable.Call{
					VarStack: varStack,
				})).To(HaveKey("Test"))
			})
			
			It("should have a destructor", func() {
				Expect(p.Destroy()).To(Succeed())
			})

		})
	}

	Describe("when interacting with a test plugin instance", func() {
		BeforeEach(func() {
			p = testplugin.NewPlugin("")
			varStack = map[string]string{
				"environment_id": "test",
				"__call_timeout": "50ms",
			}
		})

		It("should be of type *testplugin.Plugin", func() {
			Expect(p).To(BeAssignableToTypeOf(&testplugin.Plugin{}))
		})

		DoPluginTests()
	})
})
