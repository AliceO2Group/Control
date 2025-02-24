package template_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io"
	"os"
	"testing"
)

var tmpDir *string

const serviceConfigFile = "stack_test.yaml"

func TestTemplate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Component Configuration Test Suite")
}

var _ = BeforeSuite(func() {
	var err error
	tmpDir = new(string)
	*tmpDir, err = os.MkdirTemp("", "o2control-local-service")
	Expect(err).NotTo(HaveOccurred())

	// copy config files
	configFiles := []string{serviceConfigFile}
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
})

var _ = AfterSuite(func() {
	os.RemoveAll(*tmpDir)
})
