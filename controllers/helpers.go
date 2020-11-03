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
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: console.Namespace,
		},
	}
}

func (r *ConsoleReconciler) clusterRole(console *hypercloudv1.Console) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
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

func (r *ConsoleReconciler) service(console *hypercloudv1.Console) *corev1.Service {
	p := make([]corev1.ServicePort, 0)
	servicePort := corev1.ServicePort{
		Name:       "tcp-port",
		Port:       433,
		TargetPort: intstr.FromInt(6433),
	}
	p = append(p, servicePort)
	consoleSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: console.Namespace,
			Labels:    map[string]string{"app": console.Name},
		},
		Spec: corev1.ServiceSpec{
			Ports:    p,
			Type:     console.Spec.App.ServiceType,
			Selector: map[string]string{"app": console.Name},
		},
	}
	return consoleSvc
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
			APIVersion: "v1",
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
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// RemoveService deletes the service from the cluster.
func (r *ConsoleReconciler) removeService(ctx context.Context, serviceToRemove *corev1.Service, log logr.Logger) (ctrl.Result, error) {

	if err := r.Delete(ctx, serviceToRemove); err != nil {
		log.Error(err, "unable to delete console service for console", "console", serviceToRemove.Name)
		return ctrl.Result{}, err
	}
	log.V(1).Info("Removed console service for console run")
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *ConsoleReconciler) removeJob(ctx context.Context, jobToRemove *batchv1.Job, log logr.Logger) (ctrl.Result, error) {

	if err := r.Delete(ctx, jobToRemove); err != nil {
		log.Error(err, "unable to delete console job for console")
		return ctrl.Result{}, err
	}
	log.V(1).Info("Removed console job for console tls")
	return ctrl.Result{}, nil
}