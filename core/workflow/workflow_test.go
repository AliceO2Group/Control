package workflow

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"testing"
)

var _ = BeforeSuite(func() {
	viper.Set("config_endpoint", "mock://")
})

func TestCoreWorkflow(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Core Workflow Test Suite")
}
