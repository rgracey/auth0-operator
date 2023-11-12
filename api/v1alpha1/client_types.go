/*
Copyright 2023.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type SecretRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type ClientSecret struct {
	// +kubebuilder:validation:MinLength:=48
	Literal string `json:"literal,omitempty"`

	SecretRef SecretRef `json:"secretRef,omitempty"`

	OutputSecretRef SecretRef `json:"outputSecretRef,omitempty"`
}

// ClientSpec defines the desired state of Client
type ClientSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The name of the client
	Name string `json:"name,omitempty"`

	// The description of the client
	Description string `json:"description,omitempty"`

	// Allowed callback URLs for the client
	CallbackUrls []string `json:"callbackUrls,omitempty"`

	// The type of client this is
	// +kubebuilder:validation:Enum:={"spa","native","regular","non_interactive"}
	Type string `json:"type,omitempty"`

	// The metadata associated with this client
	// +kubebuilder:validation:MaxProperties:=10
	Metadata map[string]string `json:"metadata,omitempty"`

	ClientSecret ClientSecret `json:"clientSecret,omitempty"`
}

// ClientStatus defines the observed state of Client
type ClientStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The Auth0 ID of this client
	Auth0Id string `json:"auth0Id,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Client is the Schema for the clients API
type Client struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClientSpec   `json:"spec,omitempty"`
	Status ClientStatus `json:"status,omitempty"`
}

// IsBeingDeleted returns true if the Client is being deleted (i.e. has a deletion timestamp)
func (c *Client) IsBeingDeleted() bool {
	return c.GetDeletionTimestamp() != nil
}

// Auth0Id returns the Auth0 ID of the Client
func (c *Client) Auth0Id() string {
	return c.Status.Auth0Id
}

// ShouldOutputSecret returns true if the Client should create a k8s secret
func (c *Client) ShouldOutputSecret() bool {
	return c.Spec.ClientSecret.OutputSecretRef.Name != ""
}

//+kubebuilder:object:root=true

// ClientList contains a list of Client
type ClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Client `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Client{}, &ClientList{})
}
