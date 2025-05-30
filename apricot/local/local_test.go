package local

import (
	"github.com/spf13/viper"
	"io"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var tmpDir *string

const serviceConfigFile = "service_test.yaml"
const serviceHTTPConfigFile = "servicehttp_test.yaml"

func TestLocal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Local Apricot Service Suite")
}

var _ = BeforeSuite(func() {
	var err error
	tmpDir = new(string)
	*tmpDir, err = os.MkdirTemp("", "o2control-local-service")
	Expect(err).NotTo(HaveOccurred())

	// copy config files
	configFiles := []string{serviceConfigFile, serviceHTTPConfigFile}
	for _, configFile := range configFiles {
		from, err := os.Open("./" + configFile)
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := from.Close()
			if err != nil {
				Expect(err).NotTo(HaveOccurred())
			}
		}()

		to, err := os.OpenFile(*tmpDir+"/"+configFile, os.O_RDWR|os.O_CREATE, 0666)
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := to.Close()
			if err != nil {
				Expect(err).NotTo(HaveOccurred())
			}
		}()

		_, err = io.Copy(to, from)
		Expect(err).NotTo(HaveOccurred())
	}

	viper.Set("coreWorkingDir", tmpDir) // used by NewRunNumber with YAML backend
})

var _ = AfterSuite(func() {
	err := os.RemoveAll(*tmpDir)
	Expect(err).NotTo(HaveOccurred())
})
