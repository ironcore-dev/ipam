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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	machinerequestv1alpha1 "github.com/onmetal/k8s-machine-requests/api/v1alpha1"
	subnetmachinerequestv1alpha1 "github.com/onmetal/k8s-subnet-machine-request/api/v1alpha1"
	subnetv1alpha1 "github.com/onmetal/k8s-subnet/api/v1alpha1"
)

// SubnetMachineRequestReconciler reconciles a SubnetMachineRequest object
type SubnetMachineRequestReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=subnetmachinerequest.onmetal.de,resources=subnetmachinerequests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=subnetmachinerequest.onmetal.de,resources=subnetmachinerequests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=subnetmachinerequest.onmetal.de,resources=subnetmachinerequests/finalizers,verbs=update

func (r *SubnetMachineRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("subnetmachinerequest", req.NamespacedName)

	subnetMachineRequest := &subnetmachinerequestv1alpha1.SubnetMachineRequest{}
	if err := r.Get(ctx, req.NamespacedName, subnetMachineRequest); err != nil {
		// No logging if object is being deleted
		if subnetMachineRequest.GetDeletionTimestamp() != nil {
			log.Error(err, "unable to fetch SubnetMachineRequest")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var subnet subnetv1alpha1.Subnet
	if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: subnetMachineRequest.Spec.Subnet}, &subnet); err != nil {
		log.Error(err, "unable to get gateway of Subnet")
		subnetMachineRequest.Status.Status = "failed"
		subnetMachineRequest.Status.Message = "Subnet is not found"
		return r.updateStatus(log, ctx, subnetMachineRequest)
	}
	var machineRequest machinerequestv1alpha1.MachineRequest
	if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: subnetMachineRequest.Spec.MachineRequest}, &machineRequest); err != nil {
		log.Error(err, "unable to fetch MachineRequest")
		subnetMachineRequest.Status.Status = "failed"
		subnetMachineRequest.Status.Message = "MachineRequest is not found"
		return r.updateStatus(log, ctx, subnetMachineRequest)
	}
	subnetMachineRequest.Status.Message = ""
	subnetMachineRequest.Status.Status = "ready"
	return r.updateStatus(log, ctx, subnetMachineRequest)
}

func (r *SubnetMachineRequestReconciler) updateStatus(log logr.Logger, ctx context.Context, subnetMachineRequest *subnetmachinerequestv1alpha1.SubnetMachineRequest) (ctrl.Result, error) {
	err := r.Client.Status().Update(ctx, subnetMachineRequest)
	if err != nil {
		log.Error(err, "unable to update status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SubnetMachineRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&subnetmachinerequestv1alpha1.SubnetMachineRequest{}).
		Complete(r)
}
