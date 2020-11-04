/*


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
	"strconv"

	"github.com/go-logr/logr"
	hypercloudv1 "github.com/tmax-cloud/console-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConsoleReconciler reconciles a Console object
type ConsoleReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var (
	scheduledTimeAnnotation = "time.console.hypercloud.tmaxcloud.com/scheduled-at"
)

// +kubebuilder:rbac:groups=hypercloud.tmaxcloud.com,resources=consoles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hypercloud.tmaxcloud.com,resources=consoles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=*,verbs=*
// +kubebuilder:rbac:groups=rbac,resources=*,verbs=*

func (r *ConsoleReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	ctx := context.Background()
	log := r.Log.WithValues("console", req.NamespacedName)

	log.Info("Reconciling Console")

	var console hypercloudv1.Console
	if err := r.Get(ctx, req.NamespacedName, &console); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info(console.Namespace + "/" + console.Name)

	sa, err := r.desiredServiceAccount(console)
	if err != nil {
		return ctrl.Result{}, err
	}

	job, err := r.desiredJob(console)
	if err != nil {
		return ctrl.Result{}, err
	}

	depl, err := r.desiredDeployment(console)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info(depl.Spec.Selector.DeepCopy().String())

	svc, err := r.desiredService(console)
	if err != nil {
		return ctrl.Result{}, err
	}

	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("console-controller")}
	r.Create(ctx, &sa)
	r.Create(ctx, &job)

	if err := r.Patch(ctx, &depl, client.Apply, applyOpts...); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Patch(ctx, &svc, client.Apply, applyOpts...); err != nil {
		return ctrl.Result{}, err
	}

	// console.Status.LeaderService = svc.Spec.Ports[0].NodePort
	var serviceAddr string
	checkSvc := &corev1.Service{}
	r.Get(ctx, req.NamespacedName, checkSvc)
	console.Status.LeaderService = string(checkSvc.Spec.Type)
	if checkSvc.Spec.Type == corev1.ServiceTypeLoadBalancer {
		serviceAddr = "https://" + checkSvc.Status.LoadBalancer.Ingress[0].IP
	} else {
		serviceAddr = "https://<NodeIP>:" + strconv.Itoa(int(checkSvc.Spec.Ports[0].NodePort))

	}
	console.Status.Address = serviceAddr

	checkDepl := &appsv1.Deployment{}
	r.Get(ctx, req.NamespacedName, checkDepl)
	if checkDepl.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
		console.Status.ConsoleStatus = "Ready"
	} else {
		console.Status.ConsoleStatus = "Failed"
	}
	err = r.Status().Update(ctx, &console)
	tempJob := &batchv1.Job{}
	r.Get(ctx, req.NamespacedName, tempJob)
	log.Info(tempJob.Status.String())
	temp := &appsv1.Deployment{}
	r.Get(ctx, req.NamespacedName, temp)
	log.Info(temp.Spec.String())
	tempSvc := &corev1.Service{}
	r.Get(ctx, req.NamespacedName, tempSvc)
	log.Info(tempSvc.Spec.ClusterIP)
	// applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("console-controller")}

	// // Define the desired ServiceAccount object
	// consoleSa := r.serviceAccount(&console)
	// //Check if the ServiceAccount already exists
	// foundSa := &corev1.ServiceAccount{}
	// log.Info("Checking ServiceAccount")
	// if err := r.Get(ctx, types.NamespacedName{Name: console.Name, Namespace: console.Namespace}, foundSa); err != nil && errors.IsNotFound(err) {
	// 	log.Info("ServiceAccount does not exist, Creating ServiceAccount")
	// 	if err := r.Patch(ctx, consoleSa, client.Apply, applyOpts...); err != nil {
	// 		log.Info("Unable to create console ServiceAccount")
	// 		return ctrl.Result{}, err
	// 	} else {
	// 		log.Info("Success to create console sa")
	// 		return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
	// 	}
	// 	// log.Info("Creating ServiceAccount is successed")
	// } else if err != nil {
	// 	log.Info("Failed to Get console Service account")
	// 	return ctrl.Result{}, err
	// }
	// log.Info("ServiceAccount confirmed")

	// // Define the desired ClusterRole object
	// consoleCr := r.clusterRole(&console)
	// // if err := controllerutil.SetControllerReference(console, consoleCr, r.Scheme); err != nil {
	// // 	return ctrl.Result{}, err
	// // }
	// // Check if the ClusterRole already exists
	// foundCr := &rbacv1.ClusterRole{}
	// log.Info("Checking ClusterRole")
	// if err := r.Get(ctx, client.ObjectKey{Name: consoleCr.Name}, foundCr); err != nil && errors.IsNotFound(err) {
	// 	log.Info("ClusterRole does not exist, Creating ClusterRole")
	// 	if err := r.Patch(ctx, consoleCr, client.Apply, applyOpts...); err != nil {
	// 		log.Info("Unable to create console ClusterRole")
	// 		return ctrl.Result{}, err
	// 	} else {
	// 		log.Info("Creating ClusterRole is successed")
	// 		return ctrl.Result{Requeue: true, RequeueAfter: 3 * time.Second}, nil
	// 	}
	// } else if err != nil {
	// 	log.Info("Failed to Get console ClusterRole")
	// 	return ctrl.Result{Requeue: false}, err
	// }
	// log.Info("clusterRole confirmed")

	// //Define the desired ClusterRoleBinding object
	// consoleCrb := r.clusterRoleBinding(&console)
	// // if err := controllerutil.SetControllerReference(console, consoleCrb, r.Scheme); err != nil {
	// // 	return ctrl.Result{}, err
	// // }
	// // Check if the ClusterRoleBinding already exists
	// foundCrb := &rbacv1.ClusterRoleBinding{}
	// log.Info("Checking ClusterRoleBinding")
	// // if err := r.Get(ctx, client.ObjectKey{Name: consoleCrb.Name}, foundCrb); err != nil && errors.IsNotFound(err) {
	// if err := r.Get(ctx, client.ObjectKey{Name: consoleCr.Name}, foundCrb); err != nil && errors.IsNotFound(err) {
	// 	log.Info("ClusterRoleBinding does not exist, Creating ClusterRoleBinding")
	// 	if err := r.Patch(ctx, consoleCrb, client.Apply, applyOpts...); err != nil && errors.IsAlreadyExists(err) {

	// 		log.Info("Unable to create console ClusterRoleBinding")
	// 	} else {
	// 		log.Info("Creating ClusterRoleBinding is successed")
	// 		return ctrl.Result{Requeue: true, RequeueAfter: 3 * time.Second}, nil
	// 	}
	// } else if err != nil {
	// 	log.Info("Failed to Get console ClusterRoleBinding")
	// 	return ctrl.Result{Requeue: false}, err
	// }
	// log.Info("ClusterRoleBinding confirmed")

	// //Define the desired Job for creating tls certification
	// consoleJob := r.jobForTls(&console)
	// if err := controllerutil.SetControllerReference(&console, consoleJob, r.Scheme); err != nil {
	// 	return ctrl.Result{}, err
	// }
	//Check if the Job already exists
	// foundJob := &batchv1.Job{}
	// log.Info("Checking Job")
	// if err := r.Get(ctx, types.NamespacedName{Name: consoleJob.Name, Namespace: consoleJob.Namespace}, foundJob); err != nil && errors.IsNotFound(err) {
	// 	log.Info("Job for tls does not exist, Creating Job for tls")
	// 	if err := r.Patch(ctx, consoleJob, client.Apply, applyOpts...); err != nil {

	// 		log.Info("Unable to create console Job")
	// 		return ctrl.Result{}, err
	// 	} else {
	// 		log.Info("Success to create console Job")
	// 		return ctrl.Result{Requeue: true, RequeueAfter: 3 * time.Second}, nil
	// 	}
	// } else if err != nil {
	// 	log.Info("Failed to get console Job")
	// 	return ctrl.Result{}, err
	// }
	// log.Info("Job confirmed")

	// Manage Job
	// job, err := r.desiredJob(console)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	// // Manage Deployment
	// deployment, err := r.desiredDeployment(console)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	// svc, err := r.desiredService(console)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	// applyOpts = []client.PatchOption{client.ForceOwnership, client.FieldOwner("console-controller")}
	// // if err := r.Patch(ctx, &job, client.Apply, applyOpts...); err != nil {
	// // 	return ctrl.Result{}, err
	// // }
	// // if _, err := ctrl.CreateOrUpdate(ctx, r.Client, &deployment, func() error {
	// // 	return nil
	// // }); err != nil {
	// // 	return ctrl.Result{}, err
	// // }

	// if err := r.Patch(ctx, &deployment, client.Apply, applyOpts...); err != nil {
	// 	return ctrl.Result{}, err
	// }

	// if err := r.Patch(ctx, &svc, client.Apply, applyOpts...); err != nil {
	// 	return ctrl.Result{}, err
	// }
	// console.Status.LeaderService = svc.Spec.Ports[0].NodePort
	// console.Spec.App.Replicas = *deployment.Spec.Replicas
	// err = r.Status().Update(ctx, &console)
	log.Info("reconciled console")
	return ctrl.Result{}, nil
}

// var (
// 	deployOwnerKey = ".metadata.controller"
// 	apiGVstr       = hypercloudv1.GroupVersion.String()
// )

func (r *ConsoleReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&hypercloudv1.Console{}).
		// Owns(&corev1.ServiceAccount{}).
		Owns(&batchv1.Job{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
