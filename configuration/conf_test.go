package configuration_test

import (
	"github.com/AliceO2Group/Control/coconut/configuration"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)
var _ = Describe("Conf", func() {
	Describe("Test helper methods", func() {
		Context("when the user provides input for commands", func() {
			It("should pass if input is one word without `/` or `@`", func() {
				isValid := configuration.IsInputNameValid("component")
				Expect(isValid).To(Equal(true))
			})
			It("should pass if input is in format `<string>/<string>", func() {
				isValid := configuration.IsInputNameValid("component/entry")
				Expect(isValid).To(Equal(true))
			})
			It("should pass if input is in format `<string>/<string>@<int>", func() {
				isValid := configuration.IsInputNameValid("component/entry/12345")
				Expect(isValid).To(Equal(true))
			})
			It("should fail if input is one word containing `/` or `@`", func() {
				isValidSlash := configuration.IsInputNameValid("component/")
				isValidAt := configuration.IsInputNameValid("component@")
				Expect(isValidSlash).To(Equal(false))
				Expect(isValidAt).To(Equal(false))
			})

			It("should not error", func() {
			})
		})

	})

	Describe("Test coconut configuration dump command", func() {
		It("should test dump command", func() {
		})
	})

	Describe("Test coconut configuration list command", func() {
		It("should test list command", func() {
		})
	})

	Describe("Test coconut configuration show command", func() {
		It("should test show command", func() {
		})
	})

})