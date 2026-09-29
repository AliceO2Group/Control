/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2026 CERN and copyright holders of ALICE O².
 * Author: Michal Tichak <michal.tichak@cern.ch>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * In applying this license CERN does not waive the privileges and
 * immunities granted to it by virtue of its status as an
 * Intergovernmental Organization or submit itself to any jurisdiction.
 */

package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aliecsv1alpha1 "github.com/AliceO2Group/Control/control-operator/api/v1alpha1"
)

var _ = Describe("Task Controller", func() {
	// Reconcile takes the Task one step further on every pass and then returns to wait
	// for the next event: first the finalizer, then the Pod, then the gRPC connection,
	// then the state of the container, and finally the transition. These specs run in
	// order and each one drives the pass responsible for the next step.
	Describe("When reconciling a Task step by step", Ordered, func() {
		const (
			resourceName  = "test-resource"
			namespace     = "default"
			containerName = "placeholder-name"
			containerImg  = "placeholder-image:latest"
			// The OCC stub listens on loopback, so the Pod IP the controller dials has
			// to point back at the test process.
			podIP = "127.0.0.1"
		)

		var (
			ctx        context.Context
			taskName   types.NamespacedName
			podName    types.NamespacedName
			occServer  *occServerStub
			reconciler *TaskReconciler
		)

		reconcileOnce := func() reconcile.Result {
			GinkgoHelper()
			res, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: taskName})
			Expect(err).NotTo(HaveOccurred())
			return res
		}

		getTask := func() *aliecsv1alpha1.Task {
			GinkgoHelper()
			task := &aliecsv1alpha1.Task{}
			Expect(k8sClient.Get(ctx, taskName, task)).To(Succeed())
			return task
		}

		BeforeAll(func() {
			ctx = context.Background()
			taskName = types.NamespacedName{Name: resourceName, Namespace: namespace}
			podName = types.NamespacedName{Name: podNameFromTask(resourceName), Namespace: namespace}

			By("starting the OCC server stub that stands in for the task container")
			stub, port, stopOccServer, err := startOccServerStub()
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(stopOccServer)

			occServer = stub
			occServer.setState("standby")

			reconciler = &TaskReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				// A zero-value FakeRecorder discards events; NewFakeRecorder would block
				// once its buffer filled up.
				Recorder: &record.FakeRecorder{},
			}

			By("creating the custom resource for the Kind Task")
			task := &aliecsv1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: aliecsv1alpha1.TaskSpec{
					Control: aliecsv1alpha1.TaskSpecControl{Mode: "direct", Port: port},
					State:   "standby",
					Pod: v1.PodSpec{
						Containers: []v1.Container{{Name: containerName, Image: containerImg}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			// The last spec deletes the Task and its Pod through the controller, so this
			// only clears what an earlier failure left behind.
			DeferCleanup(func() {
				if c, loaded := clientsForContainers.LoadAndDelete(resourceName); loaded {
					Expect(c.(*OccClient).Close()).To(Succeed())
				}

				pod := &v1.Pod{}
				if err := k8sClient.Get(ctx, podName, pod); err == nil {
					Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, pod, client.GracePeriodSeconds(0)))).To(Succeed())
				}

				leftover := &aliecsv1alpha1.Task{}
				if err := k8sClient.Get(ctx, taskName, leftover); err == nil {
					controllerutil.RemoveFinalizer(leftover, taskFinalizer)
					Expect(k8sClient.Update(ctx, leftover)).To(Succeed())
					Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, leftover))).To(Succeed())
				}
			})
		})

		It("sets the finalizer before touching anything else", func() {
			reconcileOnce()

			Expect(controllerutil.ContainsFinalizer(getTask(), taskFinalizer)).To(BeTrue())
			Expect(k8sClient.Get(ctx, podName, &v1.Pod{})).To(Satisfy(errors.IsNotFound),
				"the Pod should wait until the finalizer is persisted")
		})

		It("creates the Pod for the Task", func() {
			reconcileOnce()

			pod := &v1.Pod{}
			Expect(k8sClient.Get(ctx, podName, pod)).To(Succeed())

			Expect(pod.Spec.Containers).To(HaveLen(1))
			Expect(pod.Spec.Containers[0].Image).To(Equal(containerImg))
			Expect(pod.Spec.RestartPolicy).To(Equal(v1.RestartPolicyNever))
			Expect(pod.Labels).To(HaveKeyWithValue("task_name", resourceName))

			Expect(pod.OwnerReferences).To(HaveLen(1))
			Expect(pod.OwnerReferences[0].Name).To(Equal(resourceName))
			Expect(pod.OwnerReferences[0].Controller).To(HaveValue(BeTrue()))
		})

		It("waits for the Pod to report an IP before connecting", func() {
			reconcileOnce()

			_, connected := clientsForContainers.Load(resourceName)
			Expect(connected).To(BeFalse())
			Expect(meta.FindStatusCondition(getTask().Status.Conditions, aliecsv1alpha1.ConditionPodReady)).To(BeNil())
		})

		It("connects to the container once the Pod is running", func() {
			By("reporting the Pod status that a kubelet would report")
			// envtest runs no kubelet, so nothing else moves the Pod to Running.
			pod := &v1.Pod{}
			Expect(k8sClient.Get(ctx, podName, pod)).To(Succeed())
			pod.Status.Phase = v1.PodRunning
			pod.Status.PodIP = podIP
			Expect(k8sClient.Status().Update(ctx, pod)).To(Succeed())

			reconcileOnce()

			occClient, connected := clientsForContainers.Load(resourceName)
			Expect(connected).To(BeTrue())

			task := getTask()
			Expect(meta.IsStatusConditionTrue(task.Status.Conditions, aliecsv1alpha1.ConditionPodReady)).To(BeTrue())
			Expect(meta.IsStatusConditionTrue(task.Status.Conditions, aliecsv1alpha1.ConditionGRPCConnected)).To(BeTrue())

			// This pass asked to be requeued until the connection is Ready, which is what
			// the next step needs.
			connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			Expect(occClient.(*OccClient).WaitUntilConnected(connectCtx)).To(Succeed())
		})

		It("reads the state from the container", func() {
			reconcileOnce()

			task := getTask()
			Expect(task.Status.State).To(Equal(occServer.getState()))
			Expect(task.Status.Error).To(BeEmpty())
			Expect(meta.IsStatusConditionTrue(task.Status.Conditions, aliecsv1alpha1.ConditionStateAccessible)).To(BeTrue())
		})

		It("transitions the container to a newly requested state", func() {
			By("asking the Task for a state the container is not in")
			task := getTask()
			task.Spec.State = "configured"
			Expect(k8sClient.Update(ctx, task)).To(Succeed())

			reconcileOnce()

			task = getTask()
			Expect(task.Status.State).To(Equal("configured"))
			Expect(task.Status.Error).To(BeEmpty())
			Expect(meta.IsStatusConditionTrue(task.Status.Conditions, aliecsv1alpha1.ConditionStateTransitionResult)).To(BeTrue())

			By("leaving the container in the requested state")
			Expect(occServer.getState()).To(Equal("configured"))
		})

		It("clears the connection and the Pod when the Task is deleted", func() {
			Expect(k8sClient.Delete(ctx, getTask())).To(Succeed())

			reconcileOnce()

			_, connected := clientsForContainers.Load(resourceName)
			Expect(connected).To(BeFalse())

			// Nothing schedules the Pod in envtest, so the API server drops it right away
			// instead of waiting for a kubelet to confirm termination.
			Expect(k8sClient.Get(ctx, podName, &v1.Pod{})).To(Satisfy(errors.IsNotFound))
			Expect(controllerutil.ContainsFinalizer(getTask(), taskFinalizer)).To(BeTrue(),
				"this pass only deletes the Pod, the finalizer goes on the next one")

			reconcileOnce()

			Expect(k8sClient.Get(ctx, taskName, &aliecsv1alpha1.Task{})).To(Satisfy(errors.IsNotFound))
		})
	})
})
