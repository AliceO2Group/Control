package configuration_test

import (
	. "github.com/AliceO2Group/Control/configuration"
	"github.com/AliceO2Group/Control/coconut/configuration"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Conf", func() {

	Describe("Test helper methods in configuration package", func() {
		Context("IsInputNameValid() - when the user provides input for commands", func() {
			It("should pass if input is one word without `/` or `@`", func() {
				isValid := configuration.IsInputNameValid("component")
				Expect(isValid).To(Equal(true))
			})
			It("should pass if input is in format `<string>/<string>", func() {
				isValid := configuration.IsInputNameValid("component/entry")
				Expect(isValid).To(Equal(true))
			})
			It("should pass if input is in format `<string>/<string>@<int>", func() {
				isValid := configuration.IsInputNameValid("component/entry@12345")
				Expect(isValid).To(Equal(true))
			})
			It("should fail if input is one word containing `/` or `@`", func() {
				isValidSlash := configuration.IsInputNameValid("component/")
				isValidAt := configuration.IsInputNameValid("component@")
				Expect(isValidSlash).To(Equal(false))
				Expect(isValidAt).To(Equal(false))
			})
		})

		Context("GetListOfComponentsAndOrWithTimestamps() - get latest timestamp for each component given", func() {
			var keys []string
			var readoutKeys []string
			readoutKeys = append(readoutKeys,"o2/components/readout/cru/1", "o2/components/readout/cru/2",
				"o2/components/readout/no-cru/10")
			keys = append(readoutKeys, "o2/components/tpc/no-cru/3", "o2/components/tpc/cru/0")

			keyPrefix := "o2/components/"

			It("should fail and return error and exit code 3 for empty array of keys being passed", func() {
				components, err, code := configuration.GetListOfComponentsAndOrWithTimestamps([]string{},  keyPrefix, false)
				Expect(err.Error()).To(Equal("No keys found"))
				Expect(code).To(Equal(3))
				Expect(components).To(Equal([]string{}))
			})

			It("should successfully pass, return error nil, code 0 and array of components when no component/timestamp is added to keyprefix", func() {
				components, err, code := configuration.GetListOfComponentsAndOrWithTimestamps(keys,  keyPrefix, false)
				Expect(err).To(BeNil())
				Expect(code).To(Equal(0))
				Expect(components).To(Equal([]string{"readout", "tpc"}))
			})

			It("should successfully pass, return error nil, code 0 and array of entries when component is provided", func() {
				components, err, code := configuration.GetListOfComponentsAndOrWithTimestamps(readoutKeys,  keyPrefix + "readout/", false)
				Expect(err).To(BeNil())
				Expect(code).To(Equal(0))
				Expect(components).To(Equal([]string{"cru", "no-cru"}))
			})

			It("should successfully pass, return error nil, code 0 and array of entries when component is provided and timestamp is used", func() {
				components, err, code := configuration.GetListOfComponentsAndOrWithTimestamps(readoutKeys,  keyPrefix + "readout/", true)
				Expect(err).To(BeNil())
				Expect(code).To(Equal(0))
				Expect(components).To(Equal([]string{"cru@2", "no-cru@10"}))
			})
		})

		Context("GetLatestTimestamp() - given a component and entry it will return the latest timestamp if exists", func() {
			var keysReadoutCRU = append([]string{}, "o2/components/readout/cru/1", "o2/components/readout/cru/2")

			It("should fail and return error and exit code 3 due to empty keys", func() {
				_, err, code := configuration.GetLatestTimestamp([]string{},  "readout", "cru")
				Expect(err.Error()).To(Equal("No keys found"))
				Expect(code).To(Equal(3))
			})

			It("should successfully return the latest timestamp of a given component & entry", func() {
				timestamp, err, code := configuration.GetLatestTimestamp(keysReadoutCRU,  "readout", "cru")
				Expect(err).To(BeNil())
				Expect(code).To(Equal(0))
				Expect(timestamp).To(Equal("2"))
			})
		})
	})

})