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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TaskTemplateSpec defines the static part of a task that is reused across environments.
// Dynamic parts (node, arguments) are provided by the Environment at instantiation time.
type TaskTemplateSpec struct {
	// Pod defines the pod spec for the task container, excluding node assignment.
	Pod v1.PodSpec `json:"pod,omitempty"`

	// Control defines the OCC gRPC control mode and port.
	Control TaskSpecControl `json:"control,omitempty"`

	// Bind defines the inbound FairMQ channels exposed by this task.
	// +optional
	Bind []TaskSpecChannelInbound `json:"bind,omitempty"`

	// Connect defines the outbound FairMQ channels this task connects to.
	// +optional
	Connect []TaskSpecChannelOutbound `json:"connect,omitempty"`

	// Properties defines static key-value properties passed to the task.
	// +optional
	Properties map[string]string `json:"properties,omitempty"`

	// Arguments defines transition arguments passed to the task via OCC gRPC.
	// +optional
	Arguments map[string]string `json:"arguments,omitempty"`

	// Names of the expected Environment variables to be passed to Pod
	// +optional
	EnvVars []string `json:"envVars,omitempty"`
}

// +kubebuilder:object:root=true

// TaskTemplate is the Schema for the tasktemplates API
type TaskTemplate struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the static task definition
	// +required
	Spec TaskTemplateSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// TaskTemplateList contains a list of TaskTemplate
type TaskTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []TaskTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TaskTemplate{}, &TaskTemplateList{})
}
