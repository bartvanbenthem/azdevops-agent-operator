/*
Copyright 2021.

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

// AgentSpec defines the desired state of Agent
type AgentSpec struct {
	//+kubebuilder:validation:Minimum=0
	// Size is the size of the Agent deployment
	Size int32 `json:"size"`
	// Image when provided overrides the default Agent image
	Image string `json:"image,omitempty"`
	// AzureDevPortal is configuring the Azure DevOps pool settings of the Agent
	// by using additional environment variables.
	Pool AzDevPool `json:"pool,omitempty"`
	// configures the proxy settings on the agent
	Proxy ProxyConfig `json:"proxy,omitempty"`
	// Allow specifying MTU value for networks used by container jobs
	// useful for docker-in-docker scenarios in k8s cluster
	MTUValue string `json:"mtuValue,omitempty"`
	// ConfigMap is for additional configurations of the Agent
	ConfigMap corev1.ConfigMap `json:"configMap,omitempty"`
}

// control the pool and agent work directory
type AzDevPool struct {
	URL       string `json:"url"`
	Token     string `json:"token"`
	PoolName  string `json:"poolName"`
	AgentName string `json:"agentName,omitempty"`
	WorkDir   string `json:"workDir,omitempty"`
}

// control the proxy configuration of the agent
type ProxyConfig struct {
	HTTPProxy  string `json:"httpProxy,omitempty"`
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	FTPProxy   string `json:"ftpProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
}

// AgentStatus defines the observed state of Agent
type AgentStatus struct {
	// Agents contains the names of the Agent pods
	// this verrifies the deployment
	Agents []string `json:"agents,omitempty"`
	// Secret contains the name of the Secret
	// this verrifies the Secret availability
	SecretAvailable string `json:"secretAvailable,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Agent is the Schema for the agents API
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentSpec   `json:"spec,omitempty"`
	Status AgentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AgentList contains a list of Agent
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Agent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Agent{}, &AgentList{})
}
