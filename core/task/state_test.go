package task_test

import (
	"testing"

	"github.com/AliceO2Group/Control/core/task/sm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("task state", func() {
	Describe("enumerated state to string conversion", func() {
		When("a State is one of the variant states", func() {
			It("should be converted literally", func() {
				Expect(sm.CONFIGURED.String()).To(Equal("CONFIGURED"))
			})
		})
		When("a State is invariant", func() {
			It("should be converted to UNKNOWN", func() {
				Expect(sm.INVARIANT.String()).To(Equal("UNKNOWN"))
			})
		})
	})

	Describe("string state to enum conversion", func() {
		When("a State is one of the variant states", func() {
			It("should be converted literally", func() {
				Expect(sm.StateFromString("CONFIGURED")).To(Equal(sm.CONFIGURED))
			})
		})
		When("a State is wrong", func() {
			It("should be converted to UNKNOWN", func() {
				Expect(sm.StateFromString("NEVADA")).To(Equal(sm.UNKNOWN))
			})
		})
	})

	Describe("state multiplication", func() {
		When("the states are equal", func() {
			It("returns the same state", func() {
				Expect(sm.UNKNOWN.X(sm.UNKNOWN)).To(Equal(sm.UNKNOWN))
				Expect(sm.RUNNING.X(sm.RUNNING)).To(Equal(sm.RUNNING))
				Expect(sm.INVARIANT.X(sm.INVARIANT)).To(Equal(sm.INVARIANT))
			})
		})

		When("one of the states is ERROR", func() {
			It("returns ERROR", func() {
				Expect(sm.ERROR.X(sm.UNKNOWN)).To(Equal(sm.ERROR))
				Expect(sm.ERROR.X(sm.RUNNING)).To(Equal(sm.ERROR))
				Expect(sm.ERROR.X(sm.INVARIANT)).To(Equal(sm.ERROR))

				Expect(sm.UNKNOWN.X(sm.ERROR)).To(Equal(sm.ERROR))
				Expect(sm.RUNNING.X(sm.ERROR)).To(Equal(sm.ERROR))
				Expect(sm.INVARIANT.X(sm.ERROR)).To(Equal(sm.ERROR))
			})
		})

		When("One of the states is INVARIANT", func() {
			It("returns the non-invariant state", func() {
				Expect(sm.INVARIANT.X(sm.UNKNOWN)).To(Equal(sm.UNKNOWN))
				Expect(sm.INVARIANT.X(sm.RUNNING)).To(Equal(sm.RUNNING))
				Expect(sm.UNKNOWN.X(sm.INVARIANT)).To(Equal(sm.UNKNOWN))
				Expect(sm.RUNNING.X(sm.INVARIANT)).To(Equal(sm.RUNNING))
				Expect(sm.DONE.X(sm.INVARIANT)).To(Equal(sm.DONE))
				Expect(sm.MIXED.X(sm.INVARIANT)).To(Equal(sm.MIXED))
				Expect(sm.ERROR.X(sm.INVARIANT)).To(Equal(sm.ERROR))
			})
		})

		When("If both states are INVARIANT", func() {
			It("returns the INVARIANT state", func() {
				Expect(sm.INVARIANT.X(sm.INVARIANT)).To(Equal(sm.INVARIANT))
			})
		})

		When("The states are concrete, but different", func() {
			It("returns MIXED", func() {
				Expect(sm.UNKNOWN.X(sm.STANDBY)).To(Equal(sm.MIXED))
				Expect(sm.RUNNING.X(sm.CONFIGURED)).To(Equal(sm.MIXED))
			})
		})
	})
})

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "State Test Suite")
}
