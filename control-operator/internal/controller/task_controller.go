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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	aliecsv1alpha1 "github.com/AliceO2Group/Control/operator/api/v1alpha1"
	"github.com/go-logr/logr"
)

// TaskReconciler reconciles a Task object
type TaskReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	NodeName string
}

var clientsForContainers map[string]*OccClient = make(map[string]*OccClient)

const taskFinalizer string = "aliecs.alice.cern/finalizer"

//+kubebuilder:rbac:groups=aliecs.alice.cern,resources=tasks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aliecs.alice.cern,resources=tasks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aliecs.alice.cern,resources=tasks/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Task object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
// TODO: right now if POD fails and stops sooner, reconciliation creates a new one...
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	t := &aliecsv1alpha1.Task{}
	if err := r.Get(ctx, req.NamespacedName, t); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.V(1).Info("new reconcile request on existing Task Kind", "request", req)

	// Handle finalizers for clean deletion
	if res, stop, err := r.handleFinalizer(ctx, t, log); err != nil || stop {
		return res, err
	}

	// TODO: add this check to direct_transition.go
	if t.Status.State == "error" && t.Spec.State != "standby" {
		return ctrl.Result{}, nil
	}

	// Get existing pod or create new one, Reconciler owns the pod, so all changes to the pod and task will trigger
	// reconciliation
	// TODO: make this function
	existingPod := &v1.Pod{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      podNameFromTask(t.Name),
		Namespace: t.Namespace,
	}, existingPod)
	if err != nil {
		if errors.IsNotFound(err) {
			pod := r.podForTask(t)
			log.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
			if err = r.Create(ctx, pod); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if failed, reason := isPodFailed(existingPod); failed {
		if t.Status.State != "error" {
			log.Info("Pod failure detected, setting Task state to error", "reason", reason)
			t.Status.State = "error"
			t.Status.Error = reason
			t.Status.Pod = *existingPod.Status.DeepCopy()
			if err := r.Status().Update(ctx, t); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Always stop reconciliation if the Pod is in a failed state
		return ctrl.Result{}, nil
	}

	if _, exists := clientsForContainers[t.Name]; !exists {
		if existingPod.Status.PodIP == "" {
			log.Info("pod doesn't have IP yet, we wait for different event")
			return ctrl.Result{}, nil
		}
		res, err := r.createGRPCConsumer(ctx, t, existingPod, req.NamespacedName, log)
		if err != nil || !res.IsZero() {
			return res, err
		}
	}

	// when this succesfully returns we should be able to communicate via gRPC
	if res := r.consumeGRPCConsumerIfReady(ctx, t, log); !res.IsZero() {
		return res, nil
	}

	// As far as I understand this is necessary because even though we have gRPC streams, we cannot rely
	// on them being implemented
	if t.Status.State == "" {
		log.V(1).Info("Status.State is empty, querying container")
		client, exists := clientsForContainers[t.Name]
		if !exists {
			return ctrl.Result{Requeue: true}, nil
		}

		stateReply, err := client.GetState(ctx)
		if err != nil {
			log.Error(err, "Failed to GetState")
			return ctrl.Result{}, nil
		}

		log.V(1).Info("Current State inside POD is: ", "state", stateReply.State)
		t.Status.State = stateReply.State
		log.Info("Updating empty Task status ", "state", t.Status.State)
		if err := r.Status().Update(ctx, t); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	oldStatus := t.Status.DeepCopy()

	t.Status.Pod = *existingPod.Status.DeepCopy()

	// Handle Spec -> gRPC State Sync
	if t.Status.State != t.Spec.State {
		client, exists := clientsForContainers[t.Name]
		if !exists {
			return ctrl.Result{Requeue: true}, nil
		}

		stateReply, err := client.GetState(ctx)
		if err != nil {
			log.Info("Failed to get state for sync, retrying in 5s", "error", err.Error())
			client.Close()
			delete(clientsForContainers, t.Name)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}

		if stateReply.GetState() != t.Spec.State {
			var (
				newState string
				transErr error
			)

			if t.Spec.Control.Mode == "fairmq" {
				newState, transErr = client.FairMQTransitionRequest(ctx, stateReply.GetState(), t.Spec.State, t.Spec.Arguments)
			} else {
				reply, err := client.TransitionRequest(ctx, stateReply.GetState(), t.Spec.State, t.Spec.Arguments)
				transErr = err
				if err == nil && reply.GetOk() {
					newState = strings.ToLower(reply.GetState())
				}
			}

			if transErr != nil {
				log.Error(transErr, "failed to Transition")
				t.Status.Error = transErr.Error()
				log.Info("Updating Task status")
				if err := r.Status().Update(ctx, t); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			}
			if newState != "" {
				log.V(1).Info("succeeded in transition", "new state", newState)
				t.Status.State = newState
			}
		}

	}

	// Status Update: Only call the API if the status actually changed
	// TODO: should I also add some query for GetState in case some Update or whatever fails?
	// TODO: is this the best way? shouldn't we do the Update the moment something changes?
	if !reflect.DeepEqual(oldStatus, &t.Status) {
		log.Info("Updating Task status")
		if err := r.Status().Update(ctx, t); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *TaskReconciler) createGRPCConsumer(ctx context.Context, t *aliecsv1alpha1.Task, found *v1.Pod, taskName types.NamespacedName, log logr.Logger) (ctrl.Result, error) {
	addr := fmt.Sprintf("%s:%d", found.Status.PodIP, t.Spec.Control.Port)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	log.Info("creating gRPC client for task to task: ", "address", addr)
	client, err := NewOccClient(ctx, addr, t.Spec.Control.Mode, r, taskName, log)
	if err != nil {
		log.Error(err, "failed to create gRPC client, retrying in 5 seconds")
		// TODO: what would be proper error handling here?
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	clientsForContainers[t.Name] = client

	return ctrl.Result{}, nil
}

func (r *TaskReconciler) consumeGRPCConsumerIfReady(ctx context.Context, t *aliecsv1alpha1.Task, log logr.Logger) ctrl.Result {
	client, exists := clientsForContainers[t.Name]

	if !exists {
		log.Info("didn't found existing client, retrying ", "task", t.Name)
		return ctrl.Result{RequeueAfter: time.Second}
	}

	if !client.ConsumeIfReady(ctx) {
		log.Info("gRPC client is not ready, retrying in 5 seconds", "name", t.Name)
		return ctrl.Result{RequeueAfter: 5 * time.Second}
	}
	return ctrl.Result{}
}

// Note: if you add finalizer to a task you cannot delete it unless you remove the finalizer, run:
// kubectl patch task --type=json -p='[{"op": "remove", "path": "/metadata/finalizers"}]'
func (r *TaskReconciler) handleFinalizer(ctx context.Context, t *aliecsv1alpha1.Task, log logr.Logger) (ctrl.Result, bool, error) {
	if t.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(t, taskFinalizer) {
			controllerutil.AddFinalizer(t, taskFinalizer)
			if err := r.Update(ctx, t); err != nil {
				return ctrl.Result{}, true, err
			}
			return ctrl.Result{}, true, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(t, taskFinalizer) {
			log.Info("Cleaning up gRPC connection before deletion")
			if client, exists := clientsForContainers[t.Name]; exists {
				if err := client.Close(); err != nil {
					log.Error(err, "Failed to close gRPC client during deletion")
				}
				delete(clientsForContainers, t.Name)
			}

			controllerutil.RemoveFinalizer(t, taskFinalizer)
			if err := r.Update(ctx, t); err != nil {
				return ctrl.Result{}, true, err
			}
		}
		return ctrl.Result{}, true, nil
	}
	return ctrl.Result{}, false, nil
}

func podNameFromTask(name string) string {
	return fmt.Sprintf("aliecs-task-pod-%s", name)
}

func (r *TaskReconciler) podForTask(t *aliecsv1alpha1.Task) *v1.Pod {
	lbls := labelsForTask(t)

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podNameFromTask(t.Name),
			Namespace: t.Namespace,
			Labels:    lbls,
		},
		Spec: t.Spec.Pod,
	}
	pod.Spec.RestartPolicy = v1.RestartPolicyNever

	_ = controllerutil.SetControllerReference(t, pod, r.Scheme)
	return pod
}

func labelsForTask(t *aliecsv1alpha1.Task) map[string]string {
	return map[string]string{
		"task_name":   t.Name,
		"application": "ControlOperator",
	}
}

func isPodFailed(pod *v1.Pod) (bool, string) {
	if pod.Status.Phase == v1.PodFailed {
		return true, fmt.Sprintf("Pod failed: %s", pod.Status.Reason)
	}

	// Helper to check container statuses
	checkContainerStatus := func(statuses []v1.ContainerStatus) (bool, string) {
		for _, cs := range statuses {
			if cs.State.Waiting != nil {
				reason := cs.State.Waiting.Reason
				switch reason {
				case "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "CreateContainerConfigError", "InvalidImageName", "CreateContainerError":
					return true, fmt.Sprintf("Container %s is in %s: %s", cs.Name, reason, cs.State.Waiting.Message)
				}
			}
			if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
				return true, fmt.Sprintf("Container %s failed with ExitCode %d: %s", cs.Name, cs.State.Terminated.ExitCode, cs.State.Terminated.Message)
			}
			// If it has restarted multiple times and is currently not ready, it's likely failing
			if cs.RestartCount > 3 && !cs.Ready {
				return true, fmt.Sprintf("Container %s has crashed %d times", cs.Name, cs.RestartCount)
			}
		}
		return false, ""
	}

	// Check main containers
	if failed, reason := checkContainerStatus(pod.Status.ContainerStatuses); failed {
		return failed, reason
	}

	// Check init containers
	if failed, reason := checkContainerStatus(pod.Status.InitContainerStatuses); failed {
		return failed, reason
	}

	return false, ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aliecsv1alpha1.Task{}, builder.WithPredicates(
			predicate.NewPredicateFuncs(func(obj client.Object) bool {
				task, ok := obj.(*aliecsv1alpha1.Task)
				if !ok {
					return false
				}
				return task.Spec.NodeName == r.NodeName
			}),
		)).
		Owns(&v1.Pod{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

func prettyPrint(i any) string {
	s, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		// If marshalling fails, return a simple error string
		return "failed to pretty-print object"
	}
	return string(s)
}
