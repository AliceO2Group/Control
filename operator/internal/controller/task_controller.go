/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2024 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
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
	"fmt"
	"reflect"

	"github.com/AliceO2Group/Control/common/product"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	aliecsv1alpha1 "github.com/AliceO2Group/Control/operator/api/v1alpha1"
)

// TaskReconciler reconciles a Task object
type TaskReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=aliecs.alice.cern,resources=tasks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aliecs.alice.cern,resources=tasks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aliecs.alice.cern,resources=tasks/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Task object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	t := &aliecsv1alpha1.Task{}
	err := r.Get(ctx, req.NamespacedName, t)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// check if the pod already exists, if not create a new pod
	found := &v1.Pod{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("aliecs-task-pod-%s", t.Name),
		Namespace: t.Namespace,
	},
		found)
	if err != nil {
		if errors.IsNotFound(err) {
			// define and create a new pod
			pod := r.podForTask(t)

			log.Log.Info("create pod", "pod", pod)
			if err = r.Create(ctx, pod); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	// update status.Pod if needed
	podStatus := found.Status.DeepCopy()
	if !reflect.DeepEqual(podStatus, t.Status.Pod) {
		t.Status.Pod = *podStatus
		if err = r.Status().Update(ctx, t); err != nil {
			return ctrl.Result{}, err
		}
	}

	// update status.State if needed - this is the handler of a task FSM state change request
	taskStatus := t.Status.DeepCopy()
	if taskStatus.State != t.Spec.State {
		// gRPC request to the task to change state or similar goes here
		t.Status.State = t.Spec.State
		if err = r.Status().Update(ctx, t); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *TaskReconciler) podForTask(t *aliecsv1alpha1.Task) *v1.Pod {
	lbls := labelsForTask(t)

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("aliecs-task-pod-%s", t.Name),
			Namespace: t.Namespace,
			Labels:    lbls,
		},
		Spec: t.Spec.Pod,
	}

	_ = controllerutil.SetControllerReference(t, pod, r.Scheme)
	return pod
}

func labelsForTask(t *aliecsv1alpha1.Task) map[string]string {
	return map[string]string{
		"task_name":   t.Name,
		"application": product.NAME,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aliecsv1alpha1.Task{}).
		Owns(&v1.Pod{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
