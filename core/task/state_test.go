package task_test

import (
	"github.com/AliceO2Group/Control/core/task"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

var _ = Describe("task state", func() {
	Describe("enumerated state to string conversion", func() {
		When("a State is one of the variant states", func() {
			It("should be converted literally", func() {
				Expect(task.CONFIGURED.String()).To(Equal("CONFIGURED"))
			})
		})
		When("a State is invariant", func() {
			It("should be converted to UNKNOWN", func() {
				Expect(task.INVARIANT.String()).To(Equal("UNKNOWN"))
			})
		})
	})

	Describe("string state to enum conversion", func() {
		When("a State is one of the variant states", func() {
			It("should be converted literally", func() {
				Expect(task.StateFromString("CONFIGURED")).To(Equal(task.CONFIGURED))
			})
		})
		When("a State is wrong", func() {
			It("should be converted to UNKNOWN", func() {
				Expect(task.StateFromString("NEVADA")).To(Equal(task.UNKNOWN))
			})
		})
	})

	Describe("state multiplication", func() {
		When("the states are equal", func() {
			It("returns the same state", func() {
				Expect(task.UNKNOWN.X(task.UNKNOWN)).To(Equal(task.UNKNOWN))
				Expect(task.RUNNING.X(task.RUNNING)).To(Equal(task.RUNNING))
				Expect(task.INVARIANT.X(task.INVARIANT)).To(Equal(task.INVARIANT))
			})
		})

		When("one of the states is ERROR", func() {
			It("returns ERROR", func() {
				Expect(task.ERROR.X(task.UNKNOWN)).To(Equal(task.ERROR))
				Expect(task.ERROR.X(task.RUNNING)).To(Equal(task.ERROR))
				Expect(task.ERROR.X(task.INVARIANT)).To(Equal(task.ERROR))

				Expect(task.UNKNOWN.X(task.ERROR)).To(Equal(task.ERROR))
				Expect(task.RUNNING.X(task.ERROR)).To(Equal(task.ERROR))
				Expect(task.INVARIANT.X(task.ERROR)).To(Equal(task.ERROR))
			})
		})

		When("One of the states is INVARIANT", func() {
			It("returns the non-invariant state", func() {
				Expect(task.INVARIANT.X(task.UNKNOWN)).To(Equal(task.UNKNOWN))
				Expect(task.INVARIANT.X(task.RUNNING)).To(Equal(task.RUNNING))
				Expect(task.UNKNOWN.X(task.INVARIANT)).To(Equal(task.UNKNOWN))
				Expect(task.RUNNING.X(task.INVARIANT)).To(Equal(task.RUNNING))
				Expect(task.DONE.X(task.INVARIANT)).To(Equal(task.DONE))
				Expect(task.MIXED.X(task.INVARIANT)).To(Equal(task.MIXED))
				Expect(task.ERROR.X(task.INVARIANT)).To(Equal(task.ERROR))
			})
		})

		When("If both states are INVARIANT", func() {
			It("returns the INVARIANT state", func() {
				Expect(task.INVARIANT.X(task.INVARIANT)).To(Equal(task.INVARIANT))
			})
		})

		When("The states are concrete, but different", func() {
			It("returns MIXED", func() {
				Expect(task.UNKNOWN.X(task.STANDBY)).To(Equal(task.MIXED))
				Expect(task.RUNNING.X(task.CONFIGURED)).To(Equal(task.MIXED))
			})
		})
	})
})

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "State Test Suite")
}
