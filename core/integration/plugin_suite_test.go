package integration_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin system test suite")
}

var _ = BeforeSuite(func() {

})

var _ = AfterSuite(func() {

})
