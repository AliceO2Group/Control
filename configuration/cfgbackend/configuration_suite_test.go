package cfgbackend_test

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var tmpDir *string

const configFile = "configuration_test.yaml"

func TestConfiguration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration Suite")
}

var _ = BeforeSuite(func() {
	var err error
	tmpDir = new(string)
	*tmpDir, err = ioutil.TempDir("", "o2control-configuration")
	Expect(err).NotTo(HaveOccurred())

	// copy config file
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
})

var _ = AfterSuite(func() {
	err := os.RemoveAll(*tmpDir)
	Expect(err).NotTo(HaveOccurred())
})
