package environment

import (
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/integration/testplugin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"io"
	"os"
	"testing"
)

const envTestConfig = "environment_test.yaml"

var tmpDir *string

var _ = BeforeSuite(func() {
	var err error
	tmpDir = new(string)
	*tmpDir, err = os.MkdirTemp("", "o2control-core-environment")
	Expect(err).NotTo(HaveOccurred())

	// copy config files
	configFiles := []string{envTestConfig}
	for _, configFile := range configFiles {
		from, err := os.Open("./" + configFile)
		Expect(err).NotTo(HaveOccurred())
		defer from.Close()

		to, err := os.OpenFile(*tmpDir+"/"+configFile, os.O_RDWR|os.O_CREATE, 0666)
		Expect(err).NotTo(HaveOccurred())
		defer to.Close()

		_, err = io.Copy(to, from)
		Expect(err).NotTo(HaveOccurred())
	}

	viper.Set("coreWorkingDir", tmpDir) // used by NewRunNumber with YAML backend

	integration.Reset()
	integration.RegisterPlugin("testplugin", "testPluginEndpoint", testplugin.NewPlugin)
	viper.Reset()
	viper.Set("integrationPlugins", []string{"testplugin"})
	viper.Set("testPluginEndpoint", "http://example.com")
	viper.Set("config_endpoint", "file://"+*tmpDir+"/"+envTestConfig)
})

func TestCoreEnvironment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Core Environment Test Suite")
}

var _ = AfterSuite(func() {
	os.RemoveAll(*tmpDir)
})
