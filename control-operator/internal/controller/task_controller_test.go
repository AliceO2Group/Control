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
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	aliecsv1alpha1 "github.com/AliceO2Group/Control/control-operator/api/v1alpha1"
)

// newTestScheme is used to register owner of pod properly in TestPodForTask
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := aliecsv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add aliecsv1alpha1 to scheme: %v", err)
	}
	return scheme
}

func TestPodForTask(t *testing.T) {
	r := &TaskReconciler{Scheme: newTestScheme(t)}

	task := &aliecsv1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-task",
			Namespace: "my-namespace",
			UID:       types.UID("test-uid"),
		},
		Spec: aliecsv1alpha1.TaskSpec{
			Pod: v1.PodSpec{
				RestartPolicy: v1.RestartPolicyAlways,
				Containers: []v1.Container{
					{Name: "main", Image: "example.io/image:latest"},
				},
			},
		},
	}

	pod := r.podForTask(task)

	if pod.Name != "aliecs-task-pod-my-task" {
		t.Errorf("pod.Name = %q, want %q", pod.Name, "aliecs-task-pod-my-task")
	}
	if pod.Namespace != task.Namespace {
		t.Errorf("pod.Namespace = %q, want %q", pod.Namespace, task.Namespace)
	}

	wantLabels := map[string]string{
		"task_name":   "my-task",
		"application": "ControlOperator",
	}
	for k, v := range wantLabels {
		if got := pod.Labels[k]; got != v {
			t.Errorf("pod.Labels[%q] = %q, want %q", k, got, v)
		}
	}

	if len(pod.Spec.Containers) != 1 || pod.Spec.Containers[0].Name != "main" {
		t.Errorf("pod.Spec.Containers not copied from task.Spec.Pod, got %+v", pod.Spec.Containers)
	}

	// podForTask always forces RestartPolicyNever, regardless of what's set on the Task spec.
	if pod.Spec.RestartPolicy != v1.RestartPolicyNever {
		t.Errorf("pod.Spec.RestartPolicy = %q, want %q", pod.Spec.RestartPolicy, v1.RestartPolicyNever)
	}

	ownerRefs := pod.GetOwnerReferences()
	if len(ownerRefs) != 1 {
		t.Fatalf("expected exactly one owner reference, got %d: %+v", len(ownerRefs), ownerRefs)
	}
	owner := ownerRefs[0]
	if owner.Kind != "Task" {
		t.Errorf("owner.Kind = %q, want %q", owner.Kind, "Task")
	}
	if owner.Name != task.Name {
		t.Errorf("owner.Name = %q, want %q", owner.Name, task.Name)
	}
	if owner.UID != task.UID {
		t.Errorf("owner.UID = %q, want %q", owner.UID, task.UID)
	}
	if owner.Controller == nil || !*owner.Controller {
		t.Errorf("owner.Controller = %v, want true", owner.Controller)
	}
}
