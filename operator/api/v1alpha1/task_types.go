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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type TaskSpecControl struct {
	// +kubebuilder:validation:Enum=direct;fairmq;basic;hook
	Mode string `json:"mode,omitempty"`
	// +kubebuilder:validation:Minimum:1024
	// +kubebuilder:validation:Maximum:49151
	Port int `json:"port,omitempty"`
}

type TaskSpecChannelInbound struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Enum=push;pull;pub;sub
	Type        string `json:"type"`
	SndBufSize  int    `json:"sndBufSize,omitempty"`
	RcvBufSize  int    `json:"rcvBufSize,omitempty"`
	RateLogging string `json:"rateLogging,omitempty"`
	// +kubebuilder:validation:Enum=default;zeromq;nanomsg;shmem
	// +kubebuilder:default=default
	Transport string `json:"transport,omitempty"`
	Target    string `json:"target"`

	Global string `json:"global"`
	// +kubebuilder:validation:Enum=tcp;ipc
	// +kubebuilder:default=tcp
	Addressing string `json:"addressing,omitempty"`
}

type TaskSpecChannelOutbound struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Enum=push;pull;pub;sub
	Type        string `json:"type"`
	SndBufSize  int    `json:"sndBufSize,omitempty"`
	RcvBufSize  int    `json:"rcvBufSize,omitempty"`
	RateLogging string `json:"rateLogging,omitempty"`
	// +kubebuilder:validation:Enum=default;zeromq;nanomsg;shmem
	// +kubebuilder:default=default
	Transport string `json:"transport,omitempty"`
	Target    string `json:"target"`
}

// TaskSpec defines the desired state of Task
type TaskSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Task. Edit task_types.go to remove/update
	Pod        v1.PodSpec                `json:"pod,omitempty"`
	Control    TaskSpecControl           `json:"control,omitempty"`
	Bind       []TaskSpecChannelInbound  `json:"bind,omitempty"`
	Connect    []TaskSpecChannelOutbound `json:"connect,omitempty"`
	Properties map[string]string         `json:"properties,omitempty"`
	// +kubebuilder:validation:Enum=standby;deployed;configured;running
	State string `json:"state,omitempty"` // this is the *requested* state, there are other states the task may end up in but cannot be requested
}

// TaskStatus defines the observed state of Task
type TaskStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Pod   v1.PodStatus `json:"pod,omitempty"`
	State string       `json:"state,omitempty"`
	Error string       `json:"error,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Task is the Schema for the tasks API
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec,omitempty"`
	Status TaskStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TaskList contains a list of Task
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Task{}, &TaskList{})
}
