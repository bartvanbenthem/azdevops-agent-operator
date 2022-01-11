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

package controllers

import (
	azdevopsv1alpha1 "github.com/bartvanbenthem/azdevops-agent-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *AgentReconciler) deploymentForAgent(m *azdevopsv1alpha1.Agent) *appsv1.Deployment {
	ls := labelsForAgent(m.Name)
	replicas := m.Spec.Size

	if m.Spec.Image == "" {
		m.Spec.Image = "bartvanbenthem/agent:latest"
	}

	dep := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: m.Spec.Image,
						Name:  "kubepodcreation",
						Env: []corev1.EnvVar{
							{
								Name: "AZP_URL",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: m.Name},
										Key: "AZP_URL",
									},
								},
							},
							{
								Name: "AZP_TOKEN",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: m.Name},
										Key: "AZP_TOKEN",
									},
								},
							},
							{
								Name: "AZP_POOL",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: m.Name},
										Key: "AZP_POOL",
									},
								},
							},
						},
					}},
				},
			},
		},
	}
	// Set Agent instance as the owner and controller
	ctrl.SetControllerReference(m, &dep, r.Scheme)
	return &dep
}

func (r *AgentReconciler) secretForAgent(m *azdevopsv1alpha1.Agent) *corev1.Secret {
	ls := labelsForAgent(m.Name)

	azp := azdevopsv1alpha1.AzDevPool{
		PoolName:  m.Spec.Pool.PoolName,
		URL:       m.Spec.Pool.URL,
		Token:     m.Spec.Pool.Token,
		AgentName: m.Spec.Pool.AgentName,
		WorkDir:   m.Spec.Pool.WorkDir,
	}

	proxy := azdevopsv1alpha1.ProxyConfig{
		HTTPProxy:  m.Spec.Proxy.HTTPProxy,
		HTTPSProxy: m.Spec.Proxy.HTTPSProxy,
		FTPProxy:   m.Spec.Proxy.FTPProxy,
		NoProxy:    m.Spec.Proxy.NoProxy,
	}

	secdata := map[string]string{}
	secdata["AZP_POOL"] = string(azp.PoolName)
	secdata["AZP_URL"] = string(azp.URL)
	secdata["AZP_TOKEN"] = string(azp.Token)
	secdata["AZP_WORK"] = string(azp.WorkDir)
	secdata["AZP_AGENT_NAME"] = string(azp.AgentName)
	secdata["HTTP_PROXY"] = string(proxy.HTTPProxy)
	secdata["HTTPS_PROXY"] = string(proxy.HTTPSProxy)
	secdata["FTP_PROXY"] = string(proxy.FTPProxy)
	secdata["NO_PROXY"] = string(proxy.NoProxy)
	secdata["AGENT_MTU_VALUE"] = string(m.Spec.MTUValue)

	sec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    ls,
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		StringData: secdata,
	}

	return &sec
}

func labelsForAgent(name string) map[string]string {
	return map[string]string{"app": "azdevops-agent", "agent_cr": name}
}

func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

func getSecret(secrets []corev1.Secret, name string) *corev1.Secret {
	var secret corev1.Secret
	for _, s := range secrets {
		if s.Name == name {
			secret = s
		}
	}
	return &secret
}
