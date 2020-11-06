/*
Copyright 2019 Google LLC

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
	"time"

	"github.com/go-logr/logr"
	hypercloudv1 "github.com/tmax-cloud/console-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ConsoleReconciler) jobForTls(console *hypercloudv1.Console) *batchv1.Job {
	secretConsoleName := console.Name + "-https-secret"
	bo := bool(true)
	in := int64(2000)
	delTime := int32(100)
	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{APIVersion: batchv1.SchemeGroupVersion.Version, Kind: "Job"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: console.Namespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &delTime,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: console.Name,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:            "create",
							Image:           "docker.io/jettech/kube-webhook-certgen:v1.3.0",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args: []string{
								"create",
								"--host=console,console.$(POD_NAMESPACE).svc",
								"--namespace=$(POD_NAMESPACE)",
								"--secret-name=" + secretConsoleName,
							},
							Env: []corev1.EnvVar{
								corev1.EnvVar{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
						},
					},
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					ServiceAccountName: console.Name,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &bo,
						RunAsUser:    &in,
					},
				},
			},
		},
	}
}

func (r *ConsoleReconciler) serviceAccount(console *hypercloudv1.Console) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{APIVersion: corev1.SchemeGroupVersion.String(), Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: console.Namespace,
		},
	}
}

func (r *ConsoleReconciler) clusterRole(console *hypercloudv1.Console) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "ClusterRole"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: console.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}
}

func (r *ConsoleReconciler) clusterRoleBinding(console *hypercloudv1.Console) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "RoleBinding"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: console.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      console.Name,
				Namespace: console.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     console.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func (r *ConsoleReconciler) desiredServiceAccount(console hypercloudv1.Console) (corev1.ServiceAccount, error) {
	sa := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{APIVersion: corev1.SchemeGroupVersion.String(), Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: console.Namespace,
		},
	}
	if err := ctrl.SetControllerReference(&console, &sa, r.Scheme); err != nil {
		return sa, err
	}
	return sa, nil
}

func (r *ConsoleReconciler) desiredClusterRole(console hypercloudv1.Console) (rbacv1.ClusterRole, error) {
	cr := rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "ClusterRole"},
		ObjectMeta: metav1.ObjectMeta{
			Name: console.Name,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}
	// if err := ctrl.SetControllerReference(&console, &cr, r.Scheme); err != nil {
	// 	return cr, err
	// }
	return cr, nil
}

func (r *ConsoleReconciler) desiredClusterRoleBinding(console hypercloudv1.Console) (rbacv1.ClusterRoleBinding, error) {
	crb := rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "RoleBinding"},
		ObjectMeta: metav1.ObjectMeta{
			Name: console.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      console.Name,
				Namespace: console.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     console.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	// if err := ctrl.SetControllerReference(&console, &crb, r.Scheme); err != nil {
	// 	return crb, err
	// }
	return crb, nil
}

func (r *ConsoleReconciler) desiredJob(console hypercloudv1.Console) (batchv1.Job, error) {
	// secretConsoleName := console.Name + "-https-secret"
	bo := bool(true)
	in := int64(2000)
	delTime := int32(100)
	job := batchv1.Job{
		TypeMeta: metav1.TypeMeta{APIVersion: batchv1.SchemeGroupVersion.String(), Kind: "Job"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: console.Namespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &delTime,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: console.Name,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:            "create",
							Image:           "docker.io/jettech/kube-webhook-certgen:v1.3.0",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args: []string{
								"create",
								"--host=console,console.$(POD_NAMESPACE).svc",
								"--namespace=$(POD_NAMESPACE)",
								"--secret-name=" + console.Name + "-https-secret",
							},
							Env: []corev1.EnvVar{
								corev1.EnvVar{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
						},
					},
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: console.Name,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &bo,
						RunAsUser:    &in,
					},
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(&console, &job, r.Scheme); err != nil {
		return job, err
	}
	return job, nil

}

func (r *ConsoleReconciler) desiredService(console hypercloudv1.Console) (corev1.Service, error) {
	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: corev1.SchemeGroupVersion.String(), Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: console.Namespace,
		},
		Spec: corev1.ServiceSpec{
			ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeLocal,
			Ports: []corev1.ServicePort{
				{Name: "https", Port: 433, Protocol: "TCP", TargetPort: intstr.FromInt(6433)},
			},
			Selector: map[string]string{"console": console.Name},
			Type:     console.Spec.App.ServiceType,
		},
	}

	// always set the controller reference so that we know which object owns this.
	if err := ctrl.SetControllerReference(&console, &svc, r.Scheme); err != nil {
		return svc, err
	}
	return svc, nil
}

func (r *ConsoleReconciler) desiredDeployment(console hypercloudv1.Console) (appsv1.Deployment, error) {
	depl := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: console.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &console.Spec.App.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"console": console.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"console": console.Name},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: console.Name,
					Containers: []corev1.Container{
						{
							Name:            console.Name,
							Image:           console.Spec.App.Repository + ":" + console.Spec.App.Tag,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"/opt/bridge/bin/bridge",
								"--public-dir=/opt/bridge/static",
								"--listen=https://0.0.0.0:6443",
								"--base-address=https://0.0.0.0:6443",
								"--tls-cert-file=/var/https-cert/cert",
								"--tls-key-file=/var/https-cert/key",
								"--user-auth=disabled",
							},
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									ContainerPort: 6443,
									Protocol:      "TCP",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								corev1.VolumeMount{
									MountPath: "/var/https-cert",
									Name:      "https-cert",
									ReadOnly:  true,
								},
							},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: "File",
						},
					},
					Volumes: []corev1.Volume{
						corev1.Volume{
							Name: "https-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: console.Name + "-https-secret",
								},
							},
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(&console, &depl, r.Scheme); err != nil {
		return depl, err
	}
	return depl, nil
}

func (r *ConsoleReconciler) deployment(console *hypercloudv1.Console) *appsv1.Deployment {
	cnts := make([]corev1.Container, 0)
	secretConsoleName := console.Name + "-https-secret"
	cnt := corev1.Container{
		Name:            console.Name,
		Image:           console.Spec.App.Repository + ":" + console.Spec.App.Tag,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"/opt/bridge/bin/bridge",
			"--public-dir=/opt/bridge/static",
			"--listen=https://0.0.0.0:6443",
			"--base-address=https://0.0.0.0:6443",
			"--tls-cert-file=/var/https-cert/cert",
			"--tls-key-file=/var/https-cert/key",
			"--user-auth=disabled",
		},
		Ports: []corev1.ContainerPort{
			corev1.ContainerPort{
				ContainerPort: 6443,
				Protocol:      "TCP",
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				MountPath: "/var/https-cert",
				Name:      "https-cert",
				ReadOnly:  true,
			},
		},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: "File",
	}
	cnts = append(cnts, cnt)
	podTempSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"app": console.Name},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: console.Name,
			Containers:         cnts,
			Volumes: []corev1.Volume{
				corev1.Volume{
					Name: "https-cert",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secretConsoleName,
						},
					},
				},
			},
		},
	}
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: console.Namespace,
			// Labels:    console.ObjectMeta.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				// MatchLabels: client.MatchingField("app", console.Name),
				MatchLabels: map[string]string{"app": console.Name},
			},
			Replicas: &console.Spec.App.Replicas,
			Template: podTempSpec,
		},
	}
	return dep
}

// RemoveDeployment deletes deployment from the cluster
func (r *ConsoleReconciler) removeDeployment(ctx context.Context, deplmtToRemove *appsv1.Deployment, log logr.Logger) (ctrl.Result, error) {

	if err := r.Delete(ctx, deplmtToRemove); err != nil {
		log.Error(err, "unable to delete console deployment for console", "console", deplmtToRemove.Name)
		return ctrl.Result{}, err
	}
	log.V(1).Info("Removed console deployment for console run")
	return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
}

// RemoveService deletes the service from the cluster.
func (r *ConsoleReconciler) removeService(ctx context.Context, serviceToRemove *corev1.Service, log logr.Logger) (ctrl.Result, error) {

	if err := r.Delete(ctx, serviceToRemove); err != nil {
		log.Error(err, "unable to delete console service for console", "console", serviceToRemove.Name)
		return ctrl.Result{}, err
	}
	log.V(1).Info("Removed console service for console run")
	return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
}

func (r *ConsoleReconciler) removeJob(ctx context.Context, jobToRemove *batchv1.Job, log logr.Logger) (ctrl.Result, error) {

	if err := r.Delete(ctx, jobToRemove); err != nil {
		log.Error(err, "unable to delete console job for console")
		return ctrl.Result{}, err
	}
	log.V(1).Info("Removed console job for console tls")
	return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
}

func (r *ConsoleReconciler) removeSa(ctx context.Context, saToRemove *corev1.ServiceAccount, log logr.Logger) (ctrl.Result, error) {
	if err := r.Delete(ctx, saToRemove); err != nil {
		log.Error(err, "unable to delete console sa for console")
		return ctrl.Result{}, err
	}
	log.Info("Removed console sa for console")
	return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
}

func (r *ConsoleReconciler) removeCr(ctx context.Context, crToRemove *rbacv1.ClusterRole, log logr.Logger) (ctrl.Result, error) {
	if err := r.Delete(ctx, crToRemove); err != nil {
		log.Error(err, "unable to delete console cr for console")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Removed console cr for console")
	return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
}
func (r *ConsoleReconciler) removeCrb(ctx context.Context, crbToRemove *rbacv1.ClusterRoleBinding, log logr.Logger) (ctrl.Result, error) {
	if err := r.Delete(ctx, crbToRemove); err != nil {
		log.Error(err, "unable to delete console crb for console")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Removed console crb for console")
	return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
}

// func (r *ConsoleReconciler) getUrl(ctx, checkSvc corev1.Service, console hypercloudv1.Console, log logr.Logger) error {
// 	// console.Status.LeaderService = string(checkSvc.Spec.Type)
// 	var serviceAddr string
// 	if checkSvc.Spec.Type == corev1.ServiceTypeLoadBalancer {
// 		console.Status.TYPE = string(corev1.ServiceTypeLoadBalancer)
// 		if checkSvc.Status.LoadBalancer.Ingress == nil || len(checkSvc.Status.LoadBalancer.Ingress) == 0 {
// 			serviceAddr = "Undefined"
// 		}
// 		serviceAddr = "https://" + checkSvc.Status.LoadBalancer.Ingress[0].IP
// 	} else {
// 		console.Status.TYPE = string(corev1.ServiceTypeNodePort)
// 		if checkSvc.Spec.Ports == nil || len(checkSvc.Spec.Ports) == 0 {
// 			serviceAddr = "Undefined"
// 		}
// 		serviceAddr = "https://<NodeIP>:" + strconv.Itoa(int(checkSvc.Spec.Ports[0].NodePort))
// 	}
// 	console.Status.URL = serviceAddr
// // }

// func (r *ConsoleReconciler) urlForService(svc corev1.Service, port int32) string {

// 	// notice that we unset this if it's not present -- we always want the
// 	// state to reflect what we observe.
// 	if len(svc.Status.LoadBalancer.Ingress) == 0 {
// 		return ""
// 	}

// 	host := svc.Status.LoadBalancer.Ingress[0].Hostnam=e
// 	if host == "" {
// 		host = svc.Status.LoadBalancer.Ingress[0].IP
// 	}

// 	return fmt.Sprintf("http://%s", net.JoinHostPort(host, fmt.Sprintf("%v", port)))
// }
