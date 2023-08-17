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

package v1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
)

type DeploymentSpecApplyConfiguration appsv1apply.DeploymentSpecApplyConfiguration

func (c *DeploymentSpecApplyConfiguration) DeepCopy() *DeploymentSpecApplyConfiguration {
	out := new(DeploymentSpecApplyConfiguration)
	bytes, err := json.Marshal(c)
	if err != nil {
		panic("Failed to marshal")
	}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		panic("Failed to unmarshal")
	}
	return out
}

// K8sNoviceOperatorSpec defines the desired state of K8sNoviceOperator
type K8sNoviceOperatorSpec struct {
	DeploymentName string                            `json:"deploymentName"`
	DeploymentSpec *DeploymentSpecApplyConfiguration `json:"deploymentSpec"`
}

// K8sNoviceOperatorStatus defines the observed state of K8sNoviceOperator
type K8sNoviceOperatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// K8sNoviceOperator is the Schema for the k8snoviceoperators API
type K8sNoviceOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sNoviceOperatorSpec   `json:"spec,omitempty"`
	Status K8sNoviceOperatorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sNoviceOperatorList contains a list of K8sNoviceOperator
type K8sNoviceOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sNoviceOperator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sNoviceOperator{}, &K8sNoviceOperatorList{})
}
