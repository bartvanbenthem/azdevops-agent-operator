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
	"context"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	azdevopsv1alpha1 "github.com/bartvanbenthem/azdevops-agent-operator/api/v1alpha1"
)

// AgentReconciler reconciles a Agent object
type AgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=azdevops.gofound.nl,resources=agents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=azdevops.gofound.nl,resources=agents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=azdevops.gofound.nl,resources=agents/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Agent object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	/////////////////////////////////////////////////////////////////////////
	// Fetch Agent object if it exists
	agent := &azdevopsv1alpha1.Agent{}
	err := r.Get(ctx, req.NamespacedName, agent)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Agent resource not found. Ignoring since object must be deleted")
			// Exit reconciliation as the object has been deleted
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Agent")
		// Requeue reconciliation as we were unable to fetch the object
		return ctrl.Result{}, err
	}

	/////////////////////////////////////////////////////////////////////////
	// Fetch Deployment object if it exists
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		dep := r.deploymentForAgent(agent)
		logger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			logger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	/////////////////////////////////////////////////////////////////////////
	// Fetch Secret object if it exists
	foundSec := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, foundSec)
	if err != nil && errors.IsNotFound(err) {
		sec := r.secretForAgent(agent)
		logger.Info("Creating a new Secret", "Secret.Namespace", sec.Namespace, "Secret.Name", sec.Name)
		err = r.Create(ctx, sec)
		if err != nil {
			logger.Error(err, "Failed to create new Secret", "Secret.Namespace", sec.Namespace, "Secret.Name", sec.Name)
			return ctrl.Result{}, err
		}
		// Secret created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Secret")
		return ctrl.Result{}, err
	}

	/////////////////////////////////////////////////////////////////////////
	// Fetch configmap object if it exists
	foundConfig := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, foundConfig)
	if err != nil && errors.IsNotFound(err) {
		config := r.configmapForAgent(agent)
		logger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", config.Namespace, "ConfigMap.Name", config.Name)
		err = r.Create(ctx, config)
		if err != nil {
			logger.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", config.Namespace, "ConfigMap.Name", config.Name)
			return ctrl.Result{}, err
		}
		// ConfigMap created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	}

	/////////////////////////////////////////////////////////////////////////
	// Ensure deployment replicas is the same as the Agent size
	size := agent.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			logger.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Ask to requeue after 1 minute in order to give enough time for the
		// pods be created on the cluster side and the operand be able
		// to do the next update step accurately.
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	/////////////////////////////////////////////////////////////////////////
	// Fetch pods to get their names
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(agent.Namespace),
		client.MatchingLabels(labelsForAgent(agent.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		logger.Error(err, "Failed to list pods", "Agent.Namespace", agent.Namespace, "Agent.Name", agent.Name)
		return ctrl.Result{}, err
	}

	/////////////////////////////////////////////////////////////////////////
	// Update Agent status with pod names
	podNames := getPodNames(podList.Items)
	if !reflect.DeepEqual(podNames, agent.Status.Agents) {
		agent.Status.Agents = podNames
		err := r.Status().Update(ctx, agent)
		if err != nil {
			logger.Error(err, "Failed to update Agent status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *AgentReconciler) deploymentForAgent(m *azdevopsv1alpha1.Agent) *appsv1.Deployment {
	ls := labelsForAgent(m.Name)
	replicas := m.Spec.Size

	if m.Spec.Image == "" {
		m.Spec.Image = "bartvanbenthem/agent:latest"
	}

	dep := &appsv1.Deployment{
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
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
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

	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    ls,
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		StringData: secdata,
	}

	return sec
}

func (r *AgentReconciler) configmapForAgent(m *azdevopsv1alpha1.Agent) *corev1.ConfigMap {
	ls := labelsForAgent(m.Name)

	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    ls,
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Data:       m.Spec.ConfigMap.Data,
		BinaryData: m.Spec.ConfigMap.BinaryData,
		Immutable:  m.Spec.ConfigMap.Immutable,
	}

	return configmap
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

// SetupWithManager sets up the controller with the Manager.
func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azdevopsv1alpha1.Agent{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
