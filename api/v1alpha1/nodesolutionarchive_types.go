/*
Copyright 2025 Scality.

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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

type SolutionArchiveChecksum struct {
	// Type of digest
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:Enum:=sha256
	Type string `json:"type"`
	// Value of the digest
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Value string `json:"value"`
}

type SolutionArchiveValidation struct {
	// Checksum of the SolutionArchive
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Checksum SolutionArchiveChecksum `json:"checksum"`
}

type SolutionArchiveSpec struct {
	// Name of the SolutionArchive
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:Pattern:=`^[a-zA-Z0-9][a-zA-Z0-9_\-\.]{1,98}[a-zA-Z0-9]$`
	Name string `json:"name"`
	// Version of the SolutionArchive
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Version string `json:"version"`
	// Validation details for the SolutionArchive (optional, but immutable)
	// +kubebuilder:validation:Optional
	Validation *SolutionArchiveValidation `json:"validation,omitempty"`
}

// NodeSolutionArchiveSpec defines the desired state of NodeSolutionArchive.
// +kubebuilder:validation:XValidation:rule="(!has(self.validation) && !has(oldSelf.validation)) || ((has(self.validation) && has(oldSelf.validation)) && (self.validation == oldSelf.validation))",message="Validation cannot be updated"
type NodeSolutionArchiveSpec struct {
	SolutionArchiveSpec `json:",inline"`
	// Name of the Node
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	NodeName string `json:"nodeName"`
}

// NodeSolutionArchiveStatus defines the observed state of NodeSolutionArchive.
type NodeSolutionArchiveStatus struct {
	// SolutionArchive on the Node is initialized and thus archive is ready to be uploaded
	Initialized *bool `json:"initialized,omitempty"`
	// Availability of the SolutionArchive on the Node
	Available *bool `json:"available,omitempty"`
	// The SolutionArchive is mounted on the Node
	Served     *bool              `json:"served,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// URL of the SolutionArchive (used internally to replicate archive between nodes)
	URL string `json:"url,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// +kubebuilder:printcolumn:name="Name",type="string",JSONPath=".spec.name"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version"
// +kubebuilder:printcolumn:name="NodeName",type="string",JSONPath=".spec.nodeName"
// +kubebuilder:printcolumn:name="Initialized",type="boolean",JSONPath=".status.initialized"
// +kubebuilder:printcolumn:name="Available",type="boolean",JSONPath=".status.available"
// +kubebuilder:printcolumn:name="Served",type="boolean",JSONPath=".status.served"
// NodeSolutionArchive is the Schema for the nodesolutionarchives API.
type NodeSolutionArchive struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeSolutionArchiveSpec   `json:"spec,omitempty"`
	Status NodeSolutionArchiveStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NodeSolutionArchiveList contains a list of NodeSolutionArchive.
type NodeSolutionArchiveList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeSolutionArchive `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &NodeSolutionArchive{}, &NodeSolutionArchiveList{})
		metav1.AddToGroupVersion(s, GroupVersion)
		return nil
	})
}

func (na *NodeSolutionArchive) SetInitialized() {
	condition := metav1.Condition{
		Type:               "Initialized",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "UploadReady",
		Message:            "The solution archive is ready to be uploaded.",
		ObservedGeneration: na.Generation,
	}
	meta.SetStatusCondition(&na.Status.Conditions, condition)
	na.Status.Initialized = ptr.To(true)
}

func (na *NodeSolutionArchive) SetNotInitialized() {
	condition := metav1.Condition{
		Type:               "Initialized",
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             "UploadNotReady",
		Message:            "The solution archive is not ready to be uploaded.",
		ObservedGeneration: na.Generation,
	}
	meta.SetStatusCondition(&na.Status.Conditions, condition)
	na.Status.Initialized = ptr.To(false)
}

func (na *NodeSolutionArchive) SetAvailable() {
	condition := metav1.Condition{
		Type:               "Available",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "ImagesAvailable",
		Message:            "The images are available in the registry.",
		ObservedGeneration: na.Generation,
	}
	meta.SetStatusCondition(&na.Status.Conditions, condition)
	na.Status.Available = ptr.To(true)
}

func (na *NodeSolutionArchive) SetUnavailable() {
	condition := metav1.Condition{
		Type:               "Available",
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             "ImagesUnavailable",
		Message:            "The images are not available in the registry.",
		ObservedGeneration: na.Generation,
	}
	meta.SetStatusCondition(&na.Status.Conditions, condition)
	na.Status.Available = ptr.To(false)
}

func (na *NodeSolutionArchive) SetServed() {
	condition := metav1.Condition{
		Type:               "Served",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "ImagesServed",
		Message:            "The images are served in the registry.",
		ObservedGeneration: na.Generation,
	}
	meta.SetStatusCondition(&na.Status.Conditions, condition)
	na.Status.Served = ptr.To(true)
}

func (na *NodeSolutionArchive) SetNotServed() {
	condition := metav1.Condition{
		Type:               "Served",
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             "ImagesNotServed",
		Message:            "The images are not served in the registry.",
		ObservedGeneration: na.Generation,
	}
	meta.SetStatusCondition(&na.Status.Conditions, condition)
	na.Status.Served = ptr.To(false)
}
