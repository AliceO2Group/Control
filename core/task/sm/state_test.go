package sm

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("task state", func() {
	Describe("enumerated state to string conversion", func() {
		When("a State is one of the variant states", func() {
			It("should be converted literally", func() {
				Expect(CONFIGURED.String()).To(Equal("CONFIGURED"))
			})
		})
		When("a State is invariant", func() {
			It("should be converted to UNKNOWN", func() {
				Expect(INVARIANT.String()).To(Equal("UNKNOWN"))
			})
		})
	})

	Describe("string state to enum conversion", func() {
		When("a State is one of the variant states", func() {
			It("should be converted literally", func() {
				Expect(StateFromString("CONFIGURED")).To(Equal(CONFIGURED))
			})
		})
		When("a State is wrong", func() {
			It("should be converted to UNKNOWN", func() {
				Expect(StateFromString("NEVADA")).To(Equal(UNKNOWN))
			})
		})
	})

	Describe("state multiplication", func() {
		When("the states are equal", func() {
			It("returns the same state", func() {
				Expect(UNKNOWN.X(UNKNOWN)).To(Equal(UNKNOWN))
				Expect(RUNNING.X(RUNNING)).To(Equal(RUNNING))
				Expect(INVARIANT.X(INVARIANT)).To(Equal(INVARIANT))
			})
		})

		When("one of the states is ERROR", func() {
			It("returns ERROR", func() {
				Expect(ERROR.X(UNKNOWN)).To(Equal(ERROR))
				Expect(ERROR.X(RUNNING)).To(Equal(ERROR))
				Expect(ERROR.X(INVARIANT)).To(Equal(ERROR))

				Expect(UNKNOWN.X(ERROR)).To(Equal(ERROR))
				Expect(RUNNING.X(ERROR)).To(Equal(ERROR))
				Expect(INVARIANT.X(ERROR)).To(Equal(ERROR))
			})
		})

		When("One of the states is INVARIANT", func() {
			It("returns the non-invariant state", func() {
				Expect(INVARIANT.X(UNKNOWN)).To(Equal(UNKNOWN))
				Expect(INVARIANT.X(RUNNING)).To(Equal(RUNNING))
				Expect(UNKNOWN.X(INVARIANT)).To(Equal(UNKNOWN))
				Expect(RUNNING.X(INVARIANT)).To(Equal(RUNNING))
				Expect(DONE.X(INVARIANT)).To(Equal(DONE))
				Expect(MIXED.X(INVARIANT)).To(Equal(MIXED))
				Expect(ERROR.X(INVARIANT)).To(Equal(ERROR))
			})
		})

		When("If both states are INVARIANT", func() {
			It("returns the INVARIANT state", func() {
				Expect(INVARIANT.X(INVARIANT)).To(Equal(INVARIANT))
			})
		})

		When("The states are concrete, but different", func() {
			It("returns MIXED", func() {
				Expect(UNKNOWN.X(STANDBY)).To(Equal(MIXED))
				Expect(RUNNING.X(CONFIGURED)).To(Equal(MIXED))
			})
		})
	})
})

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "State Test Suite")
}
