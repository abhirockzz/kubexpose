package v1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Kubexpose describes a Kubexpose resource
type Kubexpose struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	// things like...
	//  - name
	//  - namespace
	//  - self link
	//  - labels
	//  - ... etc ...
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the custom resource spec
	Spec KubexposeSpec `json:"spec"`
}

// KubexposeSpec is the spec for a Kubexpose resource
type KubexposeSpec struct {
	// ServiceName and Port are custom spec fields
	//
	// this is where you would put your custom resource data
	ServiceName string `json:"serviceName"`
	Port        *int32 `json:"port"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubexposeList is a list of Kubexpose resources
type KubexposeList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []Kubexpose `json:"items"`
}
