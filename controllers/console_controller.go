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
	"reflect"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	hypercloudv1 "github.com/tmax-cloud/console-operator/api/v1"
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

	// your logic here
	log.Info("Reconciling Console")

	// If any changes occur then reconcile function will be called.
	// Get the console object on which reconcile is called
	console := &hypercloudv1.Console{}
	if err := r.Get(ctx, req.NamespacedName, console); err != nil {
		// Created objects are automatically garbage collected.
		// For additional cleanup logic use finalizers.
		// Object not found, delete resource related console.
		log.Info("unable to fetch Consoles")
		if errors.IsNotFound(err) {
			consoleService := &corev1.Service{}
			if err := r.Get(ctx, req.NamespacedName, consoleService); err == nil {
				return r.removeService(ctx, consoleService, log)
			}

			consoleDeploy := &appsv1.Deployment{}
			if err := r.Get(ctx, req.NamespacedName, consoleDeploy); err == nil {
				return r.removeDeployment(ctx, consoleDeploy, log)
			}

			consoleJob := &batchv1.Job{}
			if err := r.Get(ctx, req.NamespacedName, consoleJob); err == nil {
				return r.removeJob(ctx, consoleJob, log)
			}
			return ctrl.Result{}, nil

			// consoleCrb := &rbacv1.ClusterRole{}
			// if err := r.Get(ctx, req.NamespacedName, consoleCrb); err == nil {
			// 	if err := r.Delete(ctx, consoleCrb); err != nil {
			// 	log.Error(err, "unable to delete console ClusterRoleBinding for console")
			// 	return ctrl.Result{}, err
			// 	}
			// log.Info("Removed console ClusterRoleBinding for console run")
			// }
		}

		return ctrl.Result{}, err
	}
	// Define the desired ServiceAccount object
	consoleSa := r.serviceAccount(console)
	//Check if the ServiceAccount already exists
	foundSa := &corev1.ServiceAccount{}
	log.Info("Checking ServiceAccount")
	if err := r.Get(ctx, types.NamespacedName{Name: consoleSa.Name, Namespace: consoleSa.Namespace}, foundSa); err != nil && errors.IsNotFound(err) {
		log.Info("ServiceAccount does not exist, Creating ServiceAccount")
		if err := r.Create(ctx, consoleSa); err != nil {
			log.Info("Unable to create console ServiceAccount")
			return ctrl.Result{}, err
		}
		log.Info("Creating ServiceAccount is successed")
	} else if err != nil {
		log.Info("Failed to Get console Service account")
		return ctrl.Result{}, err
	}
	log.Info("ServiceAccount confirmed")

	// Define the desired ClusterRole object
	consoleCr := r.clusterRole(console)
	// if err := controllerutil.SetControllerReference(console, consoleCr, r.Scheme); err != nil {
	// 	return ctrl.Result{}, err
	// }
	// Check if the ClusterRole already exists
	foundCr := &rbacv1.ClusterRole{}
	log.Info("Checking ClusterRole")
	if err := r.Get(ctx, client.ObjectKey{Name: consoleCr.Name}, foundCr); err != nil && errors.IsNotFound(err) {
		log.Info("ClusterRole does not exist, Creating ClusterRole")
		if err := r.Create(ctx, consoleCr); err != nil {
			log.Info("Unable to create console ClusterRole")
			return ctrl.Result{}, err
		}
		log.Info("Creating ClusterRole is successed")
	} else if err != nil {
		log.Info("Failed to Get console ClusterRole")
		return ctrl.Result{}, err
	}
	log.Info("clusterRole confirmed")

	//Define the desired ClusterRoleBinding object
	consoleCrb := r.clusterRoleBinding(console)
	// if err := controllerutil.SetControllerReference(console, consoleCrb, r.Scheme); err != nil {
	// 	return ctrl.Result{}, err
	// }
	// Check if the ClusterRoleBinding already exists
	foundCrb := &rbacv1.ClusterRoleBinding{}
	log.Info("Checking ClusterRoleBinding")
	if err := r.Get(ctx, client.ObjectKey{Name: consoleCrb.Name}, foundCrb); err != nil && errors.IsNotFound(err) {
		log.Info("ClusterRoleBinding does not exist, Creating ClusterRoleBinding")
		if err := r.Create(ctx, consoleCrb); err != nil {
			log.Info("Unable to create console ClusterRole")
			return ctrl.Result{}, err
		}
		log.Info("Creating ClusterRoleBinding is successed")
	} else if err != nil {
		log.Info("Failed to Get console ClusterRoleBinding")
		return ctrl.Result{}, err
	}
	log.Info("ClusterRoleBinding confirmed")

	//Define the desired Job for creating tls certification
	consoleJob := r.jobForTls(console)
	if err := controllerutil.SetControllerReference(console, consoleJob, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	//Check if the Job already exists
	foundJob := &batchv1.Job{}
	log.Info("Checking Job")
	if err := r.Get(ctx, types.NamespacedName{Name: consoleJob.Name, Namespace: consoleJob.Namespace}, foundJob); err != nil && errors.IsNotFound(err) {
		log.Info("Job for tls does not exist, Creating Job for tls")
		if err := r.Create(ctx, consoleJob); err != nil {
			log.Info("Unable to create console Job")
			return ctrl.Result{}, err
		}
		log.Info("Creating Job is successed")
	} else if err != nil {
		log.Info("Failed to get console Job")
		return ctrl.Result{}, err
	}
	log.Info("Job confirmed")

	// Define a new deployment
	consoleDeploy := r.deployment(console)
	if err := controllerutil.SetControllerReference(console, consoleDeploy, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	foundDeploy := &appsv1.Deployment{}
	log.Info("Checking Deployment")
	if err := r.Get(ctx, types.NamespacedName{Name: consoleDeploy.Name, Namespace: consoleDeploy.Namespace}, foundDeploy); err != nil && errors.IsNotFound(err) {
		log.Info("Deploy does not exist, Creating Deployment")

		// if err := r.Create(ctx, consoleDeploy); err != nil {
		if _, err := ctrl.CreateOrUpdate(ctx, r.Client, consoleDeploy, func() error { return nil }); err != nil {
			log.Info("Unable to create console Deployment")
			return ctrl.Result{}, err
		}
		log.Info("Creating Deployment is successed")
	} else if err != nil {
		log.Info("Failed to get console Deployment")
		return ctrl.Result{}, err
	}
	log.Info("Deployment confirmed")

	// Update the found object and write result back if there are any changes
	if !reflect.DeepEqual(consoleDeploy.Spec, foundDeploy.Spec) {
		foundDeploy = consoleDeploy
		log.Info("Deployment Spec is changed... Updating Deployment")
		if _, err := ctrl.CreateOrUpdate(ctx, r.Client, foundDeploy, func() error { return nil }); err != nil {
			// if err := r.Update(ctx, foundDeploy); err != nil {
			log.Info("Unable to Update console Deployment")
			return ctrl.Result{}, err
		}
	}

	//Define a new Service
	consoleSvc := r.service(console)
	if err := controllerutil.SetControllerReference(console, consoleSvc, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	foundSvc := &corev1.Service{}
	log.Info("Checking Service")
	if err := r.Get(ctx, types.NamespacedName{Name: consoleSvc.Name, Namespace: consoleSvc.Namespace}, foundSvc); err != nil && errors.IsNotFound(err) {
		log.Info("Service does not exist, Creating Service")
		if err := r.Create(ctx, consoleSvc); err != nil {
			log.Info("Unable to create console Service")
			return ctrl.Result{}, err
		}
		log.Info("Creating Service is successed")
	} else if err != nil {
		log.Info("Failed to get console Service")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// var (
// 	deployOwnerKey = ".metadata.controller"
// 	apiGVstr       = hypercloudv1.GroupVersion.String()
// )

func (r *ConsoleReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// if err := mgr.GetFieldIndexer().IndexField(&appsv1.Deployment{}, deployOwnerKey, func(rawObj runtime.Object) []string {
	// 	//grab the deploy object, extract the owner
	// 	deploy := rawObj.(*appsv1.Deployment)
	// 	owner := metav1.GetControllerOf(deploy)
	// 	if owner == nil {
	// 		return nil
	// 	}
	// 	// make sure it's a Console...
	// 	if owner.APIVersion != apiGVstr || owner.Kind != "Console" {
	// 		return nil
	// 	}
	// 	// ..and if so, return it
	// 	return []string{owner.Name}
	// }); err != nil {
	// 	return err
	// }

	return ctrl.NewControllerManagedBy(mgr).
		For(&hypercloudv1.Console{}).
		// Watches(&source.Kind{Type: &appsv1.Deployment{}},&handler.EnqueueRequestForOwner{OwnerType: })
		// Watches(&appsv1.Deployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		// Watches(
		// 	&source.Kind{Type: &hypercloudv1.Console{}},
		// 	&handler.EnqueueRequestsFromMapFunc{r.}
		// )
		// Owns(&batchv1.Job{}).
		Complete(r)
}
