package safeacks

import (
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SafeAcks", func() {
	var sa *SafeAcks

	BeforeEach(func() {
		sa = NewAcks()
	})

	Describe("RegisterAck", func() {
		It("should register a new ack", func(ctx SpecContext) {
			err := sa.RegisterAck("test")
			Expect(err).NotTo(HaveOccurred())
			Expect(sa.ExpectsAck("test")).To(BeTrue())
		}, SpecTimeout(5*time.Second))

		It("should return error when an ack is already registered", func(ctx SpecContext) {
			err := sa.RegisterAck("test")
			Expect(err).NotTo(HaveOccurred())
			Expect(sa.ExpectsAck("test")).To(BeTrue())

			err = sa.RegisterAck("test")
			Expect(err).To(HaveOccurred())

			Expect(sa.ExpectsAck("test")).To(BeTrue())
		}, SpecTimeout(5*time.Second))
	})
	// TODO add timeout for this test
	Describe("TrySendAck and TryReceiveAck", func() {
		It("should return nil for non-existent key", func(ctx SpecContext) {
			err := sa.TrySendAck("nonexistent")
			Expect(err).To(BeNil())
		}, SpecTimeout(5*time.Second))

		It("should send ack successfully", func(ctx SpecContext) {
			err := sa.RegisterAck("test")
			Expect(err).NotTo(HaveOccurred())

			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				err := sa.TrySendAck("test")
				Expect(err).To(BeNil())
			}()
			Expect(sa.TryReceiveAck("test")).To(BeTrue())

			wg.Wait()
		}, SpecTimeout(5*time.Second))

		It("should return error when ack was already sent once", func(ctx SpecContext) {
			err := sa.RegisterAck("test")
			Expect(err).NotTo(HaveOccurred())

			result1 := make(chan error)
			result2 := make(chan error)
			go func() {
				result1 <- sa.TrySendAck("test")
			}()

			go func() {
				result2 <- sa.TrySendAck("test")
			}()

			// I really don't like relying on a sleep call to test this, but I see no other way...
			// The goal is to have both `TrySendAck` blocked at channel send before invoking TryReceiveAck.
			// Hopefully 1 second is enough to avoid having a shaky test.
			time.Sleep(1000 * time.Millisecond)

			ok := sa.TryReceiveAck("test")
			Expect(ok).To(BeTrue())

			oneErrorHaveOccured := (<-result1 != nil) != (<-result2 != nil)
			Expect(oneErrorHaveOccured).To(BeTrue())
		}, SpecTimeout(5*time.Second))
	})

	Describe("ExpectsAck", func() {
		It("should return false for non-existent key", func(ctx SpecContext) {
			Expect(sa.ExpectsAck("nonexistent")).To(BeFalse())
		}, SpecTimeout(5*time.Second))

		It("should return true for registered key", func(ctx SpecContext) {
			err := sa.RegisterAck("test")
			Expect(err).NotTo(HaveOccurred())
			Expect(sa.ExpectsAck("test")).To(BeTrue())
		}, SpecTimeout(5*time.Second))

		It("should not be permanently blocked by another call", func(ctx SpecContext) {
			err := sa.RegisterAck("test")
			Expect(err).NotTo(HaveOccurred())
			go func() {
				sa.TryReceiveAck("test")
			}()

			// I really don't like relying on a sleep call to test this, but I see no other way...
			// The goal is to have `TryReceiveAck` blocked at channel receive before invoking ExpectsAck.
			// Hopefully 1 second is enough to avoid having a shaky test.
			time.Sleep(1000 * time.Millisecond)

			Expect(sa.ExpectsAck("test")).To(BeTrue())
		}, SpecTimeout(5*time.Second))
	})

	Describe("Goroutine stuck demonstration", func() {
		It("stuck", func(ctx SpecContext) {
			sa.RegisterAck("0")

			receivedAcks := 0

			go func() {
				sa.TryReceiveAck("0")
				receivedAcks += 1
			}()
			go func() {
				sa.TryReceiveAck("0")
				receivedAcks += 1
			}()

			time.Sleep(time.Second)
			sa.TrySendAck("0")
			time.Sleep(time.Second)
			sa.TrySendAck("0")
			//	Expect(MyAmazingThing()).Should(Equal(3))

			Expect(receivedAcks).Should(Equal(2))
		}, SpecTimeout(5*time.Second))
	})
})

func TestSafeAcks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Component SafeAcks Test Suite")
}
