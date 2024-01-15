/*
Copyright 2024 Magnus Bengtsson.

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

package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hyperdhcpv1beta1 "github.com/cldmnky/hyperdhcp/api/v1beta1"
)

var (
	DHCPImage = "cldmnky/hyperdhcp:latest"
)

// ServerReconciler reconciles a Server object
type ServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=hyperdhcp.blahonga.me,resources=servers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hyperdhcp.blahonga.me,resources=servers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=hyperdhcp.blahonga.me,resources=servers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
func (r *ServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var server hyperdhcpv1beta1.Server
	if err := r.Get(ctx, req.NamespacedName, &server); err != nil {
		log.Error(err, "unable to fetch Server")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.ensureDHCPDeployment(ctx, &server); err != nil {
		log.Error(err, "unable to ensure DHCP deployment")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hyperdhcpv1beta1.Server{}).
		Complete(r)
}

// ensureDHCPDeployment ensures that a DHCP server deployment exists
func (r *ServerReconciler) ensureDHCPDeployment(ctx context.Context, server *hyperdhcpv1beta1.Server) error {
	log := log.FromContext(ctx)

	deployment := newDHCPDeployment(server)
	if err := ctrl.SetControllerReference(server, deployment, r.Scheme); err != nil {
		log.Error(err, "unable to set owner reference on DHCP deployment")
		return err
	}

	_, err := CreateOrUpdateWithRetries(ctx, r.Client, deployment, func() error {
		return ctrl.SetControllerReference(server, deployment, r.Scheme)
	})
	if err != nil {
		log.Error(err, "unable to ensure DHCP deployment")
		return err
	}

	return nil
}

func newDHCPDeployment(server *hyperdhcpv1beta1.Server) *appsv1.Deployment {
	labels := map[string]string{
		"app": server.Name,
	}

	replicas := int32(1)
	runAsUser := int64(1000)
	priviliged := true

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      server.Name,
			Namespace: server.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      server.Name,
					Namespace: server.Namespace,
					Labels:    labels,
					Annotations: map[string]string{
						"k8s.v1.cni.cncf.io/networks": fmt.Sprintf(`[
							{
							  "name": "%s",
							  "namespace": "%s",
							  "ips": %s
							}
						  ]`, server.Spec.NetworkAttachment.Name, server.Spec.NetworkAttachment.NameSpace, server.Spec.NetworkAttachment.GetIPs()),
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: server.Name,
					Containers: []corev1.Container{
						{
							Name:  server.Name,
							Image: DHCPImage,
							Ports: []corev1.ContainerPort{
								{
									Name:          "dhcp",
									ContainerPort: 67,
									Protocol:      corev1.ProtocolUDP,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &runAsUser,
								Privileged: &priviliged,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "dhcp-config",
									MountPath: "/etc/dhcp",
								},
								{
									Name:      "dhcp-leases",
									MountPath: "/var/lib/dhcp",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "dhcp-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: server.Name,
									},
								},
							},
						},
						{
							Name: "dhcp-leases",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: server.Name,
								},
							},
						},
					},
				},
			},
		},
	}
}
