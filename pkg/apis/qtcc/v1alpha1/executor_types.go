/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ExecutorSpec defines the desired state of Executor
type ExecutorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Configs []ConfigCite `json:"configs,omitempty"`
}
type ConfigCite struct {
	Kind       string `json:"kind,omitempty"`
	ConfigName string `json:"configname,omitempty"`
}

// ExecutorStatus defines the observed state of Executor
type ExecutorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status of the condition, one of True, False, Unknown.
	// +required
	Status corev1.ConditionStatus `json:"status,omitempty" description:"status of the condition, one of True, False, Unknown"`

	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`

	ObservedGeneration int64                `json:"observedGeneration,omitempty"`
	Condition          map[string]Condition `json:"condition,omitempty"`
}

type Condition struct {
	Hash string `json:"hash,omitempty"`
}

//+genclient
//+genclient:noStatus
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Executor is the Schema for the executors API
type Executor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExecutorSpec   `json:"spec,omitempty"`
	Status ExecutorStatus `json:"status,omitempty"`
}

func (e *Executor) IsFinish() bool {
	if e.Generation != e.Status.ObservedGeneration {
		e.Status.ObservedGeneration = e.Generation
		return false
	}
	return e.Status.Status == corev1.ConditionFalse ||
		e.Status.Status == corev1.ConditionTrue
}

//+kubebuilder:object:root=true

// ExecutorList contains a list of Executor
type ExecutorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Executor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Executor{}, &ExecutorList{})
}
