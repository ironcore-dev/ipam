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
	if subnetMachineRequest.Spec.IP != "" {
		if !r.isIPFree() {
			subnetMachineRequest.Status.Status = "failed"
			subnetMachineRequest.Status.Message = "IP is already allocated"
			return r.updateStatus(log, ctx, subnetMachineRequest)
		}
	} else {
		ip, err := r.getFreeIP(ctx, subnetMachineRequest.Namespace, subnetMachineRequest.Spec.Subnet)
		if err != nil {
			log.Error(err, "unable to get free IP for SubnetMachineRequest")
			return ctrl.Result{}, err
		}
		subnetMachineRequest.Spec.IP = ip
		err = r.Update(ctx, subnetMachineRequest)
		if err != nil {
			log.Error(err, "unable to update SubnetMachineRequest")
			return ctrl.Result{}, err
		}
	}
	subnetMachineRequest.Status.Message = ""
	subnetMachineRequest.Status.Status = "ready"
	return r.updateStatus(log, ctx, subnetMachineRequest)
}

func (r *SubnetMachineRequestReconciler) isIPFree() bool {
	// TODO do actual check
	return true
}

// TODO both ipv4 and ipv6
func (r *SubnetMachineRequestReconciler) getFreeIP(ctx context.Context, namespace string, subnetName string) (string, error) {
	_, err := r.findChildrenSubnetRanges(ctx, namespace, subnetName)
	if err != nil {
		return "", err
	}
	_, err = r.findReservedIPs(ctx, namespace, subnetName)
	if err != nil {
		return "", err
	}
	// TODO calculate all ranges - all ips
	return "10.10.10.10", nil
}

func (r *SubnetMachineRequestReconciler) findChildrenSubnetRanges(ctx context.Context, namespace string, subnetName string) ([]string, error) {
	subnets := []string{}
	subnetList := &subnetv1alpha1.SubnetList{}
	err := r.List(ctx, subnetList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	for index, subnet := range subnetList.Items {
		if subnet.Spec.SubnetParentID == subnetName {
			subnets = append(subnets, subnetList.Items[index].Spec.CIDR)
		}
	}
	return subnets, nil
}

func (r *SubnetMachineRequestReconciler) findReservedIPs(ctx context.Context, namespace string, subnetName string) ([]string, error) {
	reservedIPs := []string{}
	subnetMachineRequestList := &subnetmachinerequestv1alpha1.SubnetMachineRequestList{}
	err := r.List(ctx, subnetMachineRequestList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	for index, subnetMachineRequest := range subnetMachineRequestList.Items {
		if subnetMachineRequest.Spec.Subnet == subnetName && subnetMachineRequest.Spec.IP != "" {
			reservedIPs = append(reservedIPs, subnetMachineRequestList.Items[index].Spec.IP)
		}
	}
	return reservedIPs, nil
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
