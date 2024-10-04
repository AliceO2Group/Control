package environment

import (
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/integration/testplugin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"testing"
)

var _ = BeforeSuite(func() {
	integration.Reset()
	integration.RegisterPlugin("testplugin", "testPluginEndpoint", testplugin.NewPlugin)
	viper.Reset()
	viper.Set("integrationPlugins", []string{"testplugin"})
	viper.Set("testPluginEndpoint", "http://example.com")
	viper.Set("config_endpoint", "mock://")
})

func TestCoreEnvironment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Core Environment Test Suite")
}
