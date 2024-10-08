package environment

import (
	"context"
	"fmt"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type DummyTransition struct {
	baseTransition
	fail bool
}

func NewDummyTransition(transition string, fail bool) Transition {
	return &DummyTransition{
		baseTransition: baseTransition{
			name:    transition,
			taskman: nil,
		},
		fail: fail,
	}
}

func (t DummyTransition) do(env *Environment) (err error) {
	if t.fail {
		return fmt.Errorf("transition successfully failed")
	}
	return nil
}

var _ = Describe("calling hooks on FSM events", func() {
	var env *Environment
	BeforeEach(func() {
		envId, err := uid.FromString("2oDvieFrVTi")
		Expect(err).NotTo(HaveOccurred())

		env, err = newEnvironment(map[string]string{}, envId)
		Expect(err).NotTo(HaveOccurred())
		Expect(env).NotTo(BeNil())
	})

	It("should execute the requested plugin call without errors", func() {
		env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
			workflow.NewCallRole(
				"call",
				task.Traits{Trigger: "before_CONFIGURE", Timeout: "5s", Critical: true, Await: "before_CONFIGURE"},
				"testplugin.Test()",
				"")})
		workflow.LinkChildrenToParents(env.workflow)
		env.Sm.SetState("DEPLOYED")

		err := env.Sm.Event(context.Background(), "CONFIGURE", NewDummyTransition("CONFIGURE", false))

		Expect(err).NotTo(HaveOccurred())
		v, ok := env.workflow.GetUserVars().Get("root.call_called")
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal("true"))
	})

	It("should return an error and cancel transition if a critical hook fails at before_<event>", func() {
		env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
			workflow.NewCallRole(
				"call",
				task.Traits{Trigger: "before_CONFIGURE", Timeout: "5s", Critical: true, Await: "before_CONFIGURE"},
				"testplugin.Test()",
				"")})
		workflow.LinkChildrenToParents(env.workflow)
		env.Sm.SetState("DEPLOYED")
		env.workflow.GetUserVars().Set("testplugin_fail", "true")

		err := env.Sm.Event(context.Background(), "CONFIGURE", NewDummyTransition("CONFIGURE", false))

		Expect(err).To(HaveOccurred())
		v, ok := env.workflow.GetUserVars().Get("root.call_called")
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal("true"))
		Expect(env.Sm.Current()).To(Equal("DEPLOYED"))
	})

	It("should return an error and cancel transition if a critical hook fails at leave_<state>", func() {
		env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
			workflow.NewCallRole(
				"call",
				task.Traits{Trigger: "leave_DEPLOYED", Timeout: "5s", Critical: true, Await: "leave_DEPLOYED"},
				"testplugin.Test()",
				"")})
		workflow.LinkChildrenToParents(env.workflow)
		env.Sm.SetState("DEPLOYED")
		env.workflow.GetUserVars().Set("testplugin_fail", "true")

		err := env.Sm.Event(context.Background(), "CONFIGURE", NewDummyTransition("CONFIGURE", false))

		Expect(err).To(HaveOccurred())
		v, ok := env.workflow.GetUserVars().Get("root.call_called")
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal("true"))
		Expect(env.Sm.Current()).To(Equal("DEPLOYED"))
	})

	It("should return an error, but NOT cancel the transition if a critical hook fails at enter_<state>", func() {
		// ...because we cannot cancel transition that is already done.
		env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
			workflow.NewCallRole(
				"call",
				task.Traits{Trigger: "enter_CONFIGURED", Timeout: "5s", Critical: true, Await: "enter_CONFIGURED"},
				"testplugin.Test()",
				"")})
		workflow.LinkChildrenToParents(env.workflow)
		env.Sm.SetState("DEPLOYED")
		env.workflow.GetUserVars().Set("testplugin_fail", "true")

		err := env.Sm.Event(context.Background(), "CONFIGURE", NewDummyTransition("CONFIGURE", false))

		Expect(err).To(HaveOccurred())
		v, ok := env.workflow.GetUserVars().Get("root.call_called")
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal("true"))
		Expect(env.Sm.Current()).To(Equal("CONFIGURED"))
	})

	It("should return an error, but NOT cancel the transition if a critical hook fails at after_<event>", func() {
		// ...because we cannot cancel transition that is already done.
		env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
			workflow.NewCallRole(
				"call",
				task.Traits{Trigger: "after_CONFIGURE", Timeout: "5s", Critical: true, Await: "after_CONFIGURE"},
				"testplugin.Test()",
				"")})
		workflow.LinkChildrenToParents(env.workflow)
		env.Sm.SetState("DEPLOYED")
		env.workflow.GetUserVars().Set("testplugin_fail", "true")

		err := env.Sm.Event(context.Background(), "CONFIGURE", NewDummyTransition("CONFIGURE", false))

		Expect(err).To(HaveOccurred())
		v, ok := env.workflow.GetUserVars().Get("root.call_called")
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal("true"))
		Expect(env.Sm.Current()).To(Equal("CONFIGURED"))
	})

	It("should not return an error if an non-critical hook fails", func() {
		env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
			workflow.NewCallRole(
				"call",
				task.Traits{Trigger: "before_CONFIGURE", Timeout: "5s", Critical: false, Await: "before_CONFIGURE"},
				"testplugin.Test()",
				"")})
		workflow.LinkChildrenToParents(env.workflow)
		env.Sm.SetState("DEPLOYED")
		env.workflow.GetUserVars().Set("testplugin_fail", "true")

		err := env.Sm.Event(context.Background(), "CONFIGURE", NewDummyTransition("CONFIGURE", false))

		Expect(err).NotTo(HaveOccurred())
		v, ok := env.workflow.GetUserVars().Get("root.call_called")
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal("true"))
	})

	It("should execute a hook with await statement different than the trigger, but within the same transition", func() {
		env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
			workflow.NewCallRole(
				"call",
				task.Traits{Trigger: "before_CONFIGURE", Timeout: "5s", Critical: true, Await: "after_CONFIGURE"},
				"testplugin.Test()",
				"")})
		workflow.LinkChildrenToParents(env.workflow)
		env.Sm.SetState("DEPLOYED")

		err := env.Sm.Event(context.Background(), "CONFIGURE", NewDummyTransition("CONFIGURE", false))

		Expect(err).NotTo(HaveOccurred())
		v, ok := env.workflow.GetUserVars().Get("root.call_called")
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal("true"))
	})

	It("should execute a hook with await statement at a different transition than the trigger", func() {
		env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
			workflow.NewCallRole(
				"call",
				task.Traits{Trigger: "before_CONFIGURE", Timeout: "5s", Critical: true, Await: "before_RESET"},
				"testplugin.Test()",
				"")})
		workflow.LinkChildrenToParents(env.workflow)
		env.Sm.SetState("DEPLOYED")

		err := env.Sm.Event(context.Background(), "CONFIGURE", NewDummyTransition("CONFIGURE", false))
		Expect(err).NotTo(HaveOccurred())
		err = env.Sm.Event(context.Background(), "RESET", NewDummyTransition("RESET", false))
		Expect(err).NotTo(HaveOccurred())

		v, ok := env.workflow.GetUserVars().Get("root.call_called")
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal("true"))
	})

	It("should not execute a hook that should happen after a successful transition, but the transition fails", func() {
		env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
			workflow.NewCallRole(
				"call",
				task.Traits{Trigger: "after_CONFIGURE", Timeout: "5s", Critical: true, Await: "after_CONFIGURE"},
				"testplugin.Test()",
				"")})
		workflow.LinkChildrenToParents(env.workflow)
		env.Sm.SetState("DEPLOYED")

		err := env.Sm.Event(context.Background(), "CONFIGURE", NewDummyTransition("CONFIGURE", true))
		Expect(err).To(HaveOccurred())

		_, ok := env.workflow.GetUserVars().Get("root.call_called")
		Expect(ok).To(BeFalse())
	})

	Context("activity-related timestamps", func() {
		It("should set run_start_time_ms just after before_START_ACTIVITY<0 hooks are executed", func() {
			env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
				workflow.NewCallRole(
					"call1",
					task.Traits{Trigger: "before_START_ACTIVITY-1", Timeout: "5s", Critical: true, Await: "before_START_ACTIVITY-1"},
					"testplugin.TimestampObserver()",
					""),
				workflow.NewCallRole(
					"call2",
					task.Traits{Trigger: "before_START_ACTIVITY", Timeout: "5s", Critical: true, Await: "before_START_ACTIVITY"},
					"testplugin.TimestampObserver()",
					"")})
			workflow.LinkChildrenToParents(env.workflow)
			env.Sm.SetState("CONFIGURED")

			err := env.Sm.Event(context.Background(), "START_ACTIVITY", NewDummyTransition("START_ACTIVITY", false))
			Expect(err).NotTo(HaveOccurred())

			_, ok := env.workflow.GetUserVars().Get("root.call1_saw_run_start_time_ms")
			Expect(ok).To(BeFalse())
			_, ok = env.workflow.GetUserVars().Get("root.call2_saw_run_start_time_ms")
			Expect(ok).To(BeTrue())
			_, ok = env.workflow.GetUserVars().Get("root.call2_saw_run_start_completion_time_ms")
			Expect(ok).To(BeFalse())
		})
		It("should set run_start_completion_time_ms just after after_START_ACTIVITY<0 hooks are executed", func() {
			env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
				workflow.NewCallRole(
					"call1",
					task.Traits{Trigger: "after_START_ACTIVITY-1", Timeout: "5s", Critical: true, Await: "after_START_ACTIVITY-1"},
					"testplugin.TimestampObserver()",
					""),
				workflow.NewCallRole(
					"call2",
					task.Traits{Trigger: "after_START_ACTIVITY", Timeout: "5s", Critical: true, Await: "after_START_ACTIVITY"},
					"testplugin.TimestampObserver()",
					"")})
			workflow.LinkChildrenToParents(env.workflow)
			env.Sm.SetState("CONFIGURED")

			err := env.Sm.Event(context.Background(), "START_ACTIVITY", NewDummyTransition("START_ACTIVITY", false))
			Expect(err).NotTo(HaveOccurred())

			_, ok := env.workflow.GetUserVars().Get("root.call1_saw_run_start_completion_time_ms")
			Expect(ok).To(BeFalse())
			_, ok = env.workflow.GetUserVars().Get("root.call2_saw_run_start_completion_time_ms")
			Expect(ok).To(BeTrue())
		})
		It("should set run_end_time_ms just after before_STOP_ACTIVITY<0 hooks are executed", func() {
			env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
				workflow.NewCallRole(
					"call1",
					task.Traits{Trigger: "before_STOP_ACTIVITY-1", Timeout: "5s", Critical: true, Await: "before_STOP_ACTIVITY-1"},
					"testplugin.TimestampObserver()",
					""),
				workflow.NewCallRole(
					"call2",
					task.Traits{Trigger: "before_STOP_ACTIVITY", Timeout: "5s", Critical: true, Await: "before_STOP_ACTIVITY"},
					"testplugin.TimestampObserver()",
					"")})
			workflow.LinkChildrenToParents(env.workflow)
			env.Sm.SetState("CONFIGURED")

			err := env.Sm.Event(context.Background(), "START_ACTIVITY", NewDummyTransition("START_ACTIVITY", false))
			Expect(err).NotTo(HaveOccurred())
			err = env.Sm.Event(context.Background(), "STOP_ACTIVITY", NewDummyTransition("STOP_ACTIVITY", false))
			Expect(err).NotTo(HaveOccurred())

			_, ok := env.workflow.GetUserVars().Get("root.call1_saw_run_end_time_ms")
			Expect(ok).To(BeFalse())
			_, ok = env.workflow.GetUserVars().Get("root.call2_saw_run_end_time_ms")
			Expect(ok).To(BeTrue())
			_, ok = env.workflow.GetUserVars().Get("root.call2_saw_run_end_completion_time_ms")
			Expect(ok).To(BeFalse())
		})
		It("should set run_end_completion_time_ms just before after_STOP_ACTIVITY<0 hooks are executed", func() {
			env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
				workflow.NewCallRole(
					"call1",
					task.Traits{Trigger: "after_STOP_ACTIVITY-1", Timeout: "5s", Critical: true, Await: "after_STOP_ACTIVITY-1"},
					"testplugin.TimestampObserver()",
					""),
				workflow.NewCallRole(
					"call2",
					task.Traits{Trigger: "after_STOP_ACTIVITY", Timeout: "5s", Critical: true, Await: "after_STOP_ACTIVITY"},
					"testplugin.TimestampObserver()",
					"")})
			workflow.LinkChildrenToParents(env.workflow)
			env.Sm.SetState("CONFIGURED")

			err := env.Sm.Event(context.Background(), "START_ACTIVITY", NewDummyTransition("START_ACTIVITY", false))
			Expect(err).NotTo(HaveOccurred())
			err = env.Sm.Event(context.Background(), "STOP_ACTIVITY", NewDummyTransition("STOP_ACTIVITY", false))
			Expect(err).NotTo(HaveOccurred())

			_, ok := env.workflow.GetUserVars().Get("root.call1_saw_run_end_completion_time_ms")
			Expect(ok).To(BeFalse())
			_, ok = env.workflow.GetUserVars().Get("root.call2_saw_run_end_completion_time_ms")
			Expect(ok).To(BeTrue())
		})
		It("should clear timestamps from previous runs and set run_start_time_ms again before before_START_ACTIVITY hooks", func() {
			env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
				workflow.NewCallRole(
					"call",
					task.Traits{Trigger: "before_START_ACTIVITY", Timeout: "5s", Critical: true, Await: "before_START_ACTIVITY"},
					"testplugin.TimestampObserver()",
					"")})
			workflow.LinkChildrenToParents(env.workflow)
			env.Sm.SetState("CONFIGURED")

			err := env.Sm.Event(context.Background(), "START_ACTIVITY", NewDummyTransition("START_ACTIVITY", false))
			Expect(err).NotTo(HaveOccurred())
			err = env.Sm.Event(context.Background(), "STOP_ACTIVITY", NewDummyTransition("STOP_ACTIVITY", false))
			Expect(err).NotTo(HaveOccurred())

			env.workflow.GetUserVars().Del("root.call_saw_run_start_time_ms")
			env.workflow.GetUserVars().Del("root.call_saw_run_start_completion_time_ms")
			env.workflow.GetUserVars().Del("root.call_saw_run_end_time_ms")
			env.workflow.GetUserVars().Del("root.call_saw_run_end_completion_time_ms")
			err = env.Sm.Event(context.Background(), "START_ACTIVITY", NewDummyTransition("START_ACTIVITY", false))
			Expect(err).NotTo(HaveOccurred())

			v, ok := env.workflow.GetUserVars().Get("root.call_saw_run_start_time_ms")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("true"))
			_, ok = env.workflow.GetUserVars().Get("root.call_saw_run_start_completion_time_ms")
			Expect(ok).To(BeFalse())
			_, ok = env.workflow.GetUserVars().Get("root.call_saw_run_end_time_ms")
			Expect(ok).To(BeFalse())
			_, ok = env.workflow.GetUserVars().Get("root.call_saw_run_end_completion_time_ms")
			Expect(ok).To(BeFalse())
		})
		When("START_ACTIVITY transition fails", func() {
			It("should set SOSOR timestamp, while the subsequent GO_ERROR transition should set SOEOR and EOEOR (but NOT EOSOR)", func() {
				env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{})
				workflow.LinkChildrenToParents(env.workflow)
				env.Sm.SetState("CONFIGURED")

				err := env.Sm.Event(context.Background(), "START_ACTIVITY", NewDummyTransition("START_ACTIVITY", true))
				Expect(err).To(HaveOccurred())

				v, ok := env.workflow.GetUserVars().Get("run_start_time_ms")
				Expect(ok).To(BeTrue())
				Expect(v).NotTo(BeEmpty())
				v, ok = env.workflow.GetUserVars().Get("run_start_completion_time_ms")
				Expect(ok).To(BeTrue())
				Expect(v).To(BeEmpty())

				err = env.Sm.Event(context.Background(), "GO_ERROR", NewDummyTransition("GO_ERROR", false))
				Expect(err).NotTo(HaveOccurred())
				v, ok = env.workflow.GetUserVars().Get("run_start_completion_time_ms")
				Expect(ok).To(BeTrue())
				Expect(v).To(BeEmpty())
				v, ok = env.workflow.GetUserVars().Get("run_end_time_ms")
				Expect(ok).To(BeTrue())
				Expect(v).NotTo(BeEmpty())
				v, ok = env.workflow.GetUserVars().Get("run_end_completion_time_ms")
				Expect(ok).To(BeTrue())
				Expect(v).NotTo(BeEmpty())
			})
		})
		When("STOP_ACTIVITY transition fails", func() {
			It("should set SOEOR timestamp, while EOEOR should be set by subsequent GO_ERROR transition", func() {
				env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{})
				workflow.LinkChildrenToParents(env.workflow)
				env.Sm.SetState("CONFIGURED")

				err := env.Sm.Event(context.Background(), "START_ACTIVITY", NewDummyTransition("START_ACTIVITY", false))
				Expect(err).NotTo(HaveOccurred())
				err = env.Sm.Event(context.Background(), "STOP_ACTIVITY", NewDummyTransition("STOP_ACTIVITY", true))
				Expect(err).To(HaveOccurred())

				v, ok := env.workflow.GetUserVars().Get("run_end_time_ms")
				Expect(ok).To(BeTrue())
				Expect(v).NotTo(BeEmpty())
				v, ok = env.workflow.GetUserVars().Get("run_end_completion_time_ms")
				Expect(ok).To(BeTrue())
				Expect(v).To(BeEmpty())

				err = env.Sm.Event(context.Background(), "GO_ERROR", NewDummyTransition("GO_ERROR", false))
				Expect(err).NotTo(HaveOccurred())
				v, ok = env.workflow.GetUserVars().Get("run_end_completion_time_ms")
				Expect(ok).To(BeTrue())
				Expect(v).NotTo(BeEmpty())
			})
		})
		When("environment goes to ERROR while in RUNNING", func() {
			It("should set both run end timestamps", func() {
				env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
					workflow.NewCallRole(
						"call",
						task.Traits{Trigger: "leave_RUNNING", Timeout: "5s", Critical: true, Await: "leave_RUNNING"},
						"testplugin.TimestampObserver()",
						"")})
				workflow.LinkChildrenToParents(env.workflow)
				env.Sm.SetState("CONFIGURED")

				err := env.Sm.Event(context.Background(), "START_ACTIVITY", NewDummyTransition("START_ACTIVITY", false))
				Expect(err).NotTo(HaveOccurred())
				err = env.Sm.Event(context.Background(), "GO_ERROR", NewDummyTransition("GO_ERROR", false))
				Expect(err).NotTo(HaveOccurred())

				v, ok := env.workflow.GetUserVars().Get("root.call_saw_run_end_time_ms")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal("true"))
				_, ok = env.workflow.GetUserVars().Get("root.call_saw_run_end_completion_time_ms")
				Expect(ok).To(BeFalse())
				v, ok = env.workflow.GetUserVars().Get("run_end_completion_time_ms")
				Expect(ok).To(BeTrue())
				Expect(v).NotTo(BeEmpty())
			})
		})
	})

	It("should allow to arrange multiple calls in order", func() {
		env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
			workflow.NewCallRole(
				"call3",
				task.Traits{Trigger: "before_CONFIGURE+50", Timeout: "5s", Critical: true, Await: "before_CONFIGURE+50"},
				"testplugin.CallOrderObserver()",
				""),
			workflow.NewCallRole(
				"call2",
				task.Traits{Trigger: "before_CONFIGURE+0", Timeout: "5s", Critical: true, Await: "before_CONFIGURE+0"},
				"testplugin.CallOrderObserver()",
				""),
			workflow.NewCallRole(
				"call1",
				task.Traits{Trigger: "before_CONFIGURE-50", Timeout: "5s", Critical: true, Await: "before_CONFIGURE-50"},
				"testplugin.CallOrderObserver()",
				"")})
		workflow.LinkChildrenToParents(env.workflow)
		env.Sm.SetState("DEPLOYED")

		err := env.Sm.Event(context.Background(), "CONFIGURE", NewDummyTransition("CONFIGURE", false))

		Expect(err).NotTo(HaveOccurred())
		v, ok := env.workflow.GetUserVars().Get("call_history") // set by testplugin.CallOrderObserver
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal("root.call1,root.call2,root.call3"))
	})

	It("should allow to cancel hooks in case that await trigger never happens", func() {
		env.workflow = workflow.NewAggregatorRole("root", []workflow.Role{
			workflow.NewCallRole(
				"call1", // this call should return immediately and should not be accessible later
				task.Traits{Trigger: "before_CONFIGURE", Timeout: "5s", Critical: true, Await: "before_CONFIGURE"},
				"testplugin.Test()",
				""),
			workflow.NewCallRole(
				"call2", // this call should not return, but should be cancelled later
				task.Traits{Trigger: "before_CONFIGURE", Timeout: "5s", Critical: true, Await: "after_CONFIGURE"},
				"testplugin.Test()",
				"")})
		workflow.LinkChildrenToParents(env.workflow)
		env.Sm.SetState("DEPLOYED")

		err := env.Sm.Event(context.Background(), "CONFIGURE", NewDummyTransition("CONFIGURE", true))
		Expect(err).To(HaveOccurred())

		callMapForAwait := env.callsPendingAwait
		Expect(callMapForAwait).To(HaveKey("after_CONFIGURE"))
		callsForWeight := callMapForAwait["after_CONFIGURE"]
		Expect(callsForWeight).To(HaveKey(callable.HookWeight(0)))
		calls := callsForWeight[0]
		Expect(calls).To(HaveLen(1))
		Expect(calls[0]).NotTo(BeNil())
		// the first cancel attempt should return "true" to say it was successful
		Expect(calls[0].Cancel()).To(BeTrue())
		// the subsequent cancel attempts should return "false", because the call was already cancelled
		Expect(calls[0].Cancel()).To(BeFalse())
	})
})
