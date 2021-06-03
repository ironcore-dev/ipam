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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/onmetal/ipam/api/v1alpha1"
)

const (
	CSubnetFinalizer = "subnet.ipam.onmetal.de/finalizer"
)

// SubnetReconciler reconciles a Subnet object
type SubnetReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=subnets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=subnets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=subnets/finalizers,verbs=update

// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networkglobals,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networkglobals/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *SubnetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("subnet", req.NamespacedName)

	// Get subnet
	// If resource is not found, then most likely it was deleted
	subnet := &v1alpha1.Subnet{}
	err := r.Get(ctx, req.NamespacedName, subnet)
	if apierrors.IsNotFound(err) {
		log.Error(err, "requested subnet resource not found", "name", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err != nil {
		log.Error(err, "unable to get subnet resource", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	// If deletion timestamp is present,
	// then resource is scheduled for deletion.
	if subnet.GetDeletionTimestamp() != nil {
		// If finalizer is set, then finalizer should be called to release
		// resources.
		if controllerutil.ContainsFinalizer(subnet, CSubnetFinalizer) {
			if err := r.finalizeSubnet(ctx, log, req.NamespacedName, subnet); err != nil {
				log.Error(err, "unable to finalize subnet resource", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(subnet, CSubnetFinalizer)
			err := r.Update(ctx, subnet)
			if err != nil {
				log.Error(err, "unable to update subnet resource on finalizer removal", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// If finalizer is not set, then resource should be updated with finalizer.
	if !controllerutil.ContainsFinalizer(subnet, CSubnetFinalizer) {
		controllerutil.AddFinalizer(subnet, CSubnetFinalizer)
		err = r.Update(ctx, subnet)
		if err != nil {
			log.Error(err, "unable to update subnet resource with finalizer", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// If state is empty, then subnet just have been created
	// and subnet status should be populated.
	if subnet.Status.State == "" {
		subnet.PopulateStatus()
		if err := r.Status().Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// If state is "Finished" or "Finalized", then
	// resource processing has been completed.
	if subnet.Status.State == v1alpha1.CFailedSubnetState ||
		subnet.Status.State == v1alpha1.CFinishedSubnetState {
		return ctrl.Result{}, nil
	}

	// If parent subnet is not set, then CIDR should be reserved in
	// global network resource.
	if subnet.Spec.ParentSubnetName == "" {
		networkGlobalNamespacedName := types.NamespacedName{
			Namespace: subnet.Namespace,
			Name:      subnet.Spec.NetworkGlobalName,
		}

		networkGlobal := &v1alpha1.NetworkGlobal{}

		if err := r.Get(ctx, networkGlobalNamespacedName, networkGlobal); err != nil {
			log.Error(err, "unable to get network global", "name", req.NamespacedName, "network global name", networkGlobalNamespacedName)
			return ctrl.Result{}, err
		}

		// If it is not possible to reserve subnet's CIDR in global network,
		// then CIDR (or its part) is already reserved,
		// and CIDR allocation has failed.
		if err := networkGlobal.Reserve(&subnet.Spec.CIDR); err != nil {
			log.Error(err, "unable to reserve subnet in network global", "name", req.NamespacedName, "network global name", networkGlobalNamespacedName)
			subnet.Status.State = v1alpha1.CFailedSubnetState
			subnet.Status.Message = err.Error()
			if err := r.Status().Update(ctx, subnet); err != nil {
				log.Error(err, "unable to update subnet status", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}

		if err := r.Status().Update(ctx, networkGlobal); err != nil {
			log.Error(err, "unable to update network global", "name", req.NamespacedName, "network global name", networkGlobalNamespacedName)
			return ctrl.Result{}, err
		}

		subnet.Status.State = v1alpha1.CFinishedSubnetState
		if err := r.Status().Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// If parent subnet is set, then current subnet's CIDR
	// should be registered in parent subnet.
	parentSubnetNamespacedName := types.NamespacedName{
		Namespace: subnet.Namespace,
		Name:      subnet.Spec.ParentSubnetName,
	}

	parentSubnet := &v1alpha1.Subnet{}

	if err := r.Get(ctx, parentSubnetNamespacedName, parentSubnet); err != nil {
		log.Error(err, "unable to get parent subnet", "name", req.NamespacedName, "parent name", parentSubnetNamespacedName)
		return ctrl.Result{}, err
	}

	// If it is not possible to reserve subnet's CIDR in parent subnet,
	// then CIDR (or its part) is already reserved, and CIDR allocation has failed.
	if err := parentSubnet.Reserve(&subnet.Spec.CIDR); err != nil {
		log.Error(err, "unable to reserve cidr in parent subnet", "name", req.NamespacedName, "parent name", parentSubnetNamespacedName)
		subnet.Status.State = v1alpha1.CFailedSubnetState
		subnet.Status.Message = err.Error()
		if err := r.Status().Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if err := r.Status().Update(ctx, parentSubnet); err != nil {
		log.Error(err, "unable to update parent subnet status after cidr reservation", "name", req.NamespacedName, "parent name", parentSubnetNamespacedName)
		return ctrl.Result{}, err
	}

	subnet.Status.State = v1alpha1.CFinishedSubnetState
	if err := r.Status().Update(ctx, subnet); err != nil {
		log.Error(err, "unable to update parent subnet status after cidr reservation", "name", req.NamespacedName, "parent name", parentSubnetNamespacedName)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SubnetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Subnet{}).
		Complete(r)
}

// finalizeSubnet releases subnet CIDR from parent subnet of network global.
func (r *SubnetReconciler) finalizeSubnet(ctx context.Context, log logr.Logger, namespacedName types.NamespacedName, subnet *v1alpha1.Subnet) error {
	// If subnet has no parent subnet, then it should be released from global network.
	// Otherwise, release from parent subnet
	if subnet.Spec.ParentSubnetName == "" {
		networkGlobalNamespacedName := types.NamespacedName{
			Namespace: subnet.Namespace,
			Name:      subnet.Spec.NetworkGlobalName,
		}

		networkGlobal := &v1alpha1.NetworkGlobal{}

		// If parent entity is not found, then it is possible to release CIDR.
		if err := r.Get(ctx, networkGlobalNamespacedName, networkGlobal); err != nil {
			log.Error(err, "unable to get network global", "name", namespacedName, "network global name", networkGlobalNamespacedName)
			if apierrors.IsNotFound(err) {
				log.Error(err, "network global not found, going to complete finalizer", "name", namespacedName, "network global name", networkGlobalNamespacedName)
				return nil
			}
			return err
		}

		// If release fails and it is possible to reserve the same CIDR,
		// then it can be considered as already released by 3rd party.
		if err := networkGlobal.Release(&subnet.Spec.CIDR); err != nil {
			log.Error(err, "unable to release subnet in network global", "name", namespacedName, "network global name", networkGlobalNamespacedName)
			if networkGlobal.CanReserve(&subnet.Spec.CIDR) {
				log.Error(err, "seems that CIDR was released beforehand", "name", namespacedName, "network global name", networkGlobalNamespacedName)
				return nil
			}
			return err
		}

		if err := r.Status().Update(ctx, networkGlobal); err != nil {
			log.Error(err, "unable to update network global", "name", namespacedName, "network global name", networkGlobalNamespacedName)
			return err
		}
	} else {
		parentSubnetNamespacedName := types.NamespacedName{
			Namespace: subnet.Namespace,
			Name:      subnet.Spec.ParentSubnetName,
		}

		parentSubnet := &v1alpha1.Subnet{}

		if err := r.Get(ctx, parentSubnetNamespacedName, parentSubnet); err != nil {
			log.Error(err, "unable to get parent subnet", "name", namespacedName, "parent name", parentSubnetNamespacedName)
			if apierrors.IsNotFound(err) {
				log.Error(err, "parent subnet not found, going to complete finalize", "name", namespacedName, "parent name", parentSubnetNamespacedName)
				return nil
			}
			return err
		}

		if err := parentSubnet.Release(&subnet.Spec.CIDR); err != nil {
			log.Error(err, "unable to release cidr in parent subnet", "name", namespacedName, "parent name", parentSubnetNamespacedName)
			if parentSubnet.CanReserve(&subnet.Spec.CIDR) {
				log.Error(err, "seems that CIDR was released beforehand", "name", namespacedName, "parent name", parentSubnetNamespacedName)
				return nil
			}
			return err
		}

		if err := r.Status().Update(ctx, parentSubnet); err != nil {
			log.Error(err, "unable to update parent subnet status after cidr reservation", "name", namespacedName, "parent name", parentSubnetNamespacedName)
			return err
		}
	}

	return nil
}
