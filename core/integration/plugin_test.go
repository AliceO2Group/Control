package integration_test

import (
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/integration/testplugin"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"testing"
)

var _ = Describe("plugin framework", Ordered, func() {

	BeforeEach(func() {
		// Reset the integration package state before each test
		integration.Reset()
		viper.Reset()
	})

	var _ = Describe("Plugin Registration", Ordered, func() {
		It("should register a plugin correctly", func() {
			// ... even if we register it multiple times
			integration.RegisterPlugin("testplugin", "testPluginEndpoint", testplugin.NewPlugin)
			integration.RegisterPlugin("testplugin", "testPluginEndpoint", testplugin.NewPlugin)

			registeredPlugins := integration.RegisteredPlugins()
			Expect(registeredPlugins).To(HaveLen(1))
			Expect(registeredPlugins).To(HaveKey("testplugin"))

			pluginLoader := registeredPlugins["testplugin"]
			Expect(pluginLoader).ToNot(BeNil())
		})
	})

	var _ = Describe("Loading Plugins", Ordered, func() {
		BeforeEach(func() {
			// Reset the integration package state before each test
			integration.Reset()
			viper.Reset()
		})
		It("should load a plugin correctly", func() {
			integration.RegisterPlugin("testplugin", "testPluginEndpoint", testplugin.NewPlugin)
			viper.Set("testPluginEndpoint", "http://example.com")
			viper.Set("integrationPlugins", []string{"testplugin"})
			plugins := integration.PluginsInstance()
			Expect(plugins).To(HaveLen(1))
			Expect(plugins[0].GetName()).To(Equal("testplugin"))
		})
		It("should fail to load a plugin if endpoint is not set", func() {
			integration.RegisterPlugin("testplugin", "testPluginEndpoint", testplugin.NewPlugin)
			viper.Set("integrationPlugins", []string{"testplugin"})
			plugins := integration.PluginsInstance()
			Expect(plugins).To(BeEmpty())
		})
		It("should not load a plugin which is not registered", func() {
			viper.Set("testPluginEndpoint", "http://example.com")
			viper.Set("integrationPlugins", []string{"testplugin"})
			plugins := integration.PluginsInstance()
			Expect(plugins).To(BeEmpty())
		})
		It("should not load a plugin which is not in the list of integration plugins", func() {
			integration.RegisterPlugin("testplugin", "testPluginEndpoint", testplugin.NewPlugin)
			viper.Set("testPluginEndpoint", "http://example.com")
			viper.Set("integrationPlugins", []string{})
			plugins := integration.PluginsInstance()
			Expect(plugins).To(BeEmpty())
		})
	})

	Describe("Plugin Getters", Ordered, func() {
		var pluginInstance integration.Plugin

		BeforeEach(func() {
			integration.RegisterPlugin("testplugin", "testPluginEndpoint", testplugin.NewPlugin)

			registeredPlugins := integration.RegisteredPlugins()
			pluginLoader := registeredPlugins["testplugin"]

			// Simulate setting the endpoint in Viper
			viper.Set("testPluginEndpoint", "http://example.com")
			pluginInstance = pluginLoader()
		})

		It("should return the correct name", func() {
			Expect(pluginInstance.GetName()).To(Equal("testplugin"))
		})

		It("should return the correct pretty name", func() {
			Expect(pluginInstance.GetPrettyName()).To(Equal("Test plugin"))
		})

		It("should return the correct connection state", func() {
			Expect(pluginInstance.GetConnectionState()).To(Equal("READY"))
		})
		It("should set and get the endpoint correctly", func() {
			Expect(pluginInstance.GetEndpoint()).To(Equal("http://example.com"))
		})

		It("should return the correct data", func() {
			Expect(pluginInstance.GetData(nil)).To(Equal("test_data"))
		})

		It("should return the correct environments data", func() {
			envId, _ := uid.FromString("2oDvieFrVTi")
			envData := pluginInstance.GetEnvironmentsData([]uid.ID{envId})
			Expect(envData).To(HaveLen(1))
			Expect(envData).To(HaveKeyWithValue(envId, "test_data_2oDvieFrVTi"))
		})

		It("should return the correct environments short data", func() {
			envId, _ := uid.FromString("2oDvieFrVTi")
			envData := pluginInstance.GetEnvironmentsShortData([]uid.ID{envId})
			Expect(envData).To(HaveLen(1))
			Expect(envData).To(HaveKeyWithValue(envId, "test_short_data_2oDvieFrVTi"))
		})
	})

	Describe("Global Plugin Getters", Ordered, func() {
		var plugins integration.Plugins

		BeforeEach(func() {
			integration.Reset()
			integration.RegisterPlugin("testplugin", "testPluginEndpoint", testplugin.NewPlugin)
			viper.Reset()
			viper.Set("integrationPlugins", []string{"testplugin"})
			viper.Set("testPluginEndpoint", "http://example.com")
			plugins = integration.PluginsInstance()
		})
		It("should return the data returned by the plugin", func() {
			data := plugins.GetData(nil)
			Expect(data).To(HaveLen(1))
			Expect(data).To(HaveKeyWithValue("testplugin", "test_data"))
		})

		It("should return the env data returned by the plugin", func() {
			envId, _ := uid.FromString("2oDvieFrVTi")
			envData := plugins.GetEnvironmentsData([]uid.ID{envId})
			Expect(envData).To(HaveLen(1))
			Expect(envData).To(HaveKey(envId))
			envDataTestPlugin := envData[envId]
			Expect(envDataTestPlugin).To(HaveLen(1))
			Expect(envDataTestPlugin).To(HaveKeyWithValue("testplugin", "test_data_2oDvieFrVTi"))
		})

		It("should return the short env data returned by the plugin", func() {
			envId, _ := uid.FromString("2oDvieFrVTi")
			envData := plugins.GetEnvironmentsShortData([]uid.ID{envId})
			Expect(envData).To(HaveLen(1))
			Expect(envData).To(HaveKey(envId))
			envDataTestPlugin := envData[envId]
			Expect(envDataTestPlugin).To(HaveLen(1))
			Expect(envDataTestPlugin).To(HaveKeyWithValue("testplugin", "test_short_data_2oDvieFrVTi"))
		})
	})

	Describe("InitAll and DestroyAll", Ordered, func() {
		var plugins integration.Plugins

		BeforeEach(func() {
			viper.Set("integrationPlugins", []string{"testplugin"})
			viper.Set("testPluginEndpoint", "http://example.com")
			integration.RegisterPlugin("testplugin", "testPluginEndpoint", testplugin.NewPlugin)
			plugins = integration.PluginsInstance()
		})

		It("should initialize and destroy all plugins correctly", func() {
			plugins.InitAll("instance_id")
			plugins.DestroyAll()
		})
	})

	Describe("Call Stack", Ordered, func() {
		var plugins integration.Plugins

		BeforeEach(func() {
			viper.Reset()
			viper.Set("integrationPlugins", []string{"testplugin"})
			viper.Set("testPluginEndpoint", "http://example.com")
			integration.RegisterPlugin("testplugin", "testPluginEndpoint", testplugin.NewPlugin)
			plugins = integration.PluginsInstance()
		})

		It("should create a call stack for all registered plugins", func() {
			call := &callable.Call{
				VarStack: map[string]string{
					"environment_id": "2oDvieFrVTi",
				},
			}

			callStack := plugins.CallStack(call)

			Expect(callStack).To(HaveLen(1))
			Expect(callStack).To(HaveKey("testplugin"))

			testPluginCallStack := callStack["testplugin"]
			Expect(testPluginCallStack).To(HaveKey("Noop"))
			Expect(testPluginCallStack).To(HaveKey("Test"))
		})
	})

	Describe("Object Stack", Ordered, func() {
		var plugins integration.Plugins

		BeforeEach(func() {
			viper.Reset()
			viper.Set("integrationPlugins", []string{"testplugin"})
			viper.Set("testPluginEndpoint", "http://example.com")
			integration.RegisterPlugin("testplugin", "testPluginEndpoint", testplugin.NewPlugin)
			plugins = integration.PluginsInstance()
		})

		It("should create an object stack for all registered plugins", func() {
			objectStack := plugins.ObjectStack(nil, nil)

			Expect(objectStack).To(HaveLen(2))
			Expect(objectStack).To(HaveKey("odc"))
			Expect(objectStack).To(HaveKey("testplugin"))
			testPluginObjectStack := objectStack["testplugin"]
			Expect(testPluginObjectStack).To(HaveKeyWithValue("test", "test_data"))
		})
	})
})

func TestCoreIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Core Integration Test Suite")
}
