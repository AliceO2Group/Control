package workflow

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestCoreWorkflow(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Core Workflow Test Suite")
}
