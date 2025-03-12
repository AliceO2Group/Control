package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

var _ = Describe("Utils", func() {
	Describe("ExtractTaskClassNameFromTaskName", func() {
		It("should extract the correct task class name", func() {
			result, err := ExtractTaskClassName("alio2-cr1-hv-gw01.cern.ch:/opt/git/ControlWorkflows/tasks/readout@12b11ac4bb652e1835e3e94806a688c951691d5f#2sP21PjpfCQ")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("readout"))
		})

		It("should extract the correct JIT task class name", func() {
			result, err := ExtractTaskClassName("alio2-cr1-hv-gw01.cern.ch:/opt/git/ControlWorkflows/tasks/jit-ad6f2b64b7502198430d7d7f93f15bf94c088cab-qc-pp-TPC-CalibQC_long@12b11ac4bb652e1835e3e94806a688c951691d5f#2sP21PjpfCQ")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("jit-ad6f2b64b7502198430d7d7f93f15bf94c088cab-qc-pp-TPC-CalibQC_long"))
		})

		It("should return an error for invalid input", func() {
			result, err := ExtractTaskClassName("invalid-string")
			Expect(err).To(HaveOccurred())
			Expect(result).To(Equal(""))
		})
	})

	Describe("TrimJitPrefix", func() {
		It("should remove the JIT prefix correctly", func() {
			result := TrimJitPrefix("jit-ad6f2b64b7502198430d7d7f93f15bf94c088cab-qc-pp-TPC-CalibQC_long")
			Expect(result).To(Equal("qc-pp-TPC-CalibQC_long"))
		})

		It("should return the original string if no JIT prefix is present", func() {
			result := TrimJitPrefix("qc-pp-TPC-CalibQC_long")
			Expect(result).To(Equal("qc-pp-TPC-CalibQC_long"))
		})
	})
})

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Component Utils Test Suite")
}
