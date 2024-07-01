package environment

import (
	"github.com/AliceO2Group/Control/common/utils/uid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("allowed states and transitions in the environment FSM", func() {
	var env *Environment
	BeforeEach(func() {
		envId, err := uid.FromString("2oDvieFrVTi")
		Expect(err).NotTo(HaveOccurred())

		env, err = newEnvironment(nil, envId)
		Expect(err).NotTo(HaveOccurred())
		Expect(env).NotTo(BeNil())
	})
	When("FSM is created", func() {
		It("should be in STANDBY", func() {
			Expect(env.Sm.Current()).To(Equal("STANDBY"))
		})
	})
	When("FSM is in STANDBY", func() {
		It("should allow for DEPLOY, GO_ERROR and EXIT transitions", func() {
			env.Sm.SetState("STANDBY")
			Expect(env.Sm.Can("DEPLOY")).To(BeTrue())
			Expect(env.Sm.Can("GO_ERROR")).To(BeTrue())
			Expect(env.Sm.Can("EXIT")).To(BeTrue())
		})
		It("should not allow for other transitions", func() {
			env.Sm.SetState("STANDBY")
			Expect(env.Sm.Cannot("CONFIGURE")).To(BeTrue())
			Expect(env.Sm.Cannot("RESET")).To(BeTrue())
			Expect(env.Sm.Cannot("START_ACTIVITY")).To(BeTrue())
			Expect(env.Sm.Cannot("STOP_ACTIVITY")).To(BeTrue())
			Expect(env.Sm.Cannot("RECOVER")).To(BeTrue())
		})
	})
	When("FSM is in DEPLOYED", func() {
		It("should allow for CONFIGURED, GO_ERROR and EXIT transitions", func() {
			env.Sm.SetState("DEPLOYED")
			Expect(env.Sm.Can("CONFIGURE")).To(BeTrue())
			Expect(env.Sm.Can("GO_ERROR")).To(BeTrue())
			Expect(env.Sm.Can("EXIT")).To(BeTrue())
		})
		It("should not allow for other transitions", func() {
			env.Sm.SetState("DEPLOYED")
			Expect(env.Sm.Cannot("DEPLOY")).To(BeTrue())
			Expect(env.Sm.Cannot("RESET")).To(BeTrue())
			Expect(env.Sm.Cannot("START_ACTIVITY")).To(BeTrue())
			Expect(env.Sm.Cannot("STOP_ACTIVITY")).To(BeTrue())
			Expect(env.Sm.Cannot("RECOVER")).To(BeTrue())
		})
	})
	When("FSM is in CONFIGURED", func() {
		It("should allow for START_ACTIVITY, RESET, GO_ERROR and EXIT transitions", func() {
			env.Sm.SetState("CONFIGURED")
			Expect(env.Sm.Can("START_ACTIVITY")).To(BeTrue())
			Expect(env.Sm.Can("RESET")).To(BeTrue())
			Expect(env.Sm.Can("GO_ERROR")).To(BeTrue())
			Expect(env.Sm.Can("EXIT")).To(BeTrue())
		})
		It("should not allow for other transitions", func() {
			env.Sm.SetState("CONFIGURED")
			Expect(env.Sm.Cannot("DEPLOY")).To(BeTrue())
			Expect(env.Sm.Cannot("CONFIGURE")).To(BeTrue())
			Expect(env.Sm.Cannot("STOP_ACTIVITY")).To(BeTrue())
			Expect(env.Sm.Cannot("RECOVER")).To(BeTrue())
		})
	})
	When("FSM is in RUNNING", func() {
		It("should allow for STOP_ACTIVITY and GO_ERROR transitions", func() {
			env.Sm.SetState("RUNNING")
			Expect(env.Sm.Can("STOP_ACTIVITY")).To(BeTrue())
			Expect(env.Sm.Can("GO_ERROR")).To(BeTrue())
		})
		It("should not allow for other transitions", func() {
			env.Sm.SetState("RUNNING")
			Expect(env.Sm.Cannot("DEPLOY")).To(BeTrue())
			Expect(env.Sm.Cannot("RESET")).To(BeTrue())
			Expect(env.Sm.Cannot("CONFIGURE")).To(BeTrue())
			Expect(env.Sm.Cannot("START_ACTIVITY")).To(BeTrue())
			Expect(env.Sm.Cannot("RECOVER")).To(BeTrue())
			Expect(env.Sm.Cannot("EXIT")).To(BeTrue())
		})
	})
	When("FSM is in ERROR", func() {
		It("should allow for RECOVER transition", func() {
			env.Sm.SetState("ERROR")
			Expect(env.Sm.Can("RECOVER")).To(BeTrue())
			// We do not include EXIT as possible transition, since anyway we kill tasks not caring about the FSM.
			// There is no known issue which could forbid us from that.
			// TEARDOWN and DESTROY are the artificial transitions which correspond to that.
		})
		It("should not allow for other transitions", func() {
			env.Sm.SetState("ERROR")
			Expect(env.Sm.Cannot("GO_ERROR")).To(BeTrue())
			Expect(env.Sm.Cannot("DEPLOY")).To(BeTrue())
			Expect(env.Sm.Cannot("RESET")).To(BeTrue())
			Expect(env.Sm.Cannot("CONFIGURE")).To(BeTrue())
			Expect(env.Sm.Cannot("START_ACTIVITY")).To(BeTrue())
			Expect(env.Sm.Cannot("STOP_ACTIVITY")).To(BeTrue())
			Expect(env.Sm.Cannot("EXIT")).To(BeTrue())
		})
	})
	When("FSM is in DONE", func() {
		It("should not allow for any transitions", func() {
			env.Sm.SetState("DONE")
			Expect(env.Sm.Cannot("GO_ERROR")).To(BeTrue())
			Expect(env.Sm.Cannot("DEPLOY")).To(BeTrue())
			Expect(env.Sm.Cannot("RESET")).To(BeTrue())
			Expect(env.Sm.Cannot("CONFIGURE")).To(BeTrue())
			Expect(env.Sm.Cannot("START_ACTIVITY")).To(BeTrue())
			Expect(env.Sm.Cannot("STOP_ACTIVITY")).To(BeTrue())
			Expect(env.Sm.Cannot("RECOVER")).To(BeTrue())
			Expect(env.Sm.Cannot("EXIT")).To(BeTrue())
		})
	})
})
