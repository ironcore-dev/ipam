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
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	machinev1alpha1 "github.com/onmetal/ipam/api/v1alpha1"
)

const (
	CNetworkFinalizer = "network.ipam.onmetal.de/finalizer"

	CVXLANCounterName = "k8s-vxlan-network-counter"
	CMPLSCounterName  = "k8s-mpls-network-counter"
)

// NetworkReconciler reconciles a Network object
type NetworkReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networkcounters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networkcounters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networkcounters/finalizers,verbs=update

// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Network object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("network", req.NamespacedName)

	network := &machinev1alpha1.Network{}
	err := r.Get(ctx, req.NamespacedName, network)
	if apierrors.IsNotFound(err) {
		log.Error(err, "requested network resource not found", "name", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err != nil {
		log.Error(err, "unable to get network resource", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if network.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(network, CNetworkFinalizer) {
			if err := r.finalizeNetwork(ctx, log, network); err != nil {
				log.Error(err, "unable to finalize network resource", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(network, CNetworkFinalizer)
			err := r.Update(ctx, network)
			if err != nil {
				log.Error(err, "unable to update network resource on finalizer removal", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(network, CNetworkFinalizer) {
		controllerutil.AddFinalizer(network, CNetworkFinalizer)
		err = r.Update(ctx, network)
		if err != nil {
			log.Error(err, "unable to update network resource with finalizer", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if network.Status.State == machinev1alpha1.CFinishedRequestState &&
		network.Status.Reserved == nil &&
		network.Spec.Type != "" {
		network.Status.State = machinev1alpha1.CProcessingRequestState
		if err := r.Status().Update(ctx, network); err != nil {
			log.Error(err, "unable to update network resource status", "name", req.NamespacedName, "currentStatus", network.Status.State, "targetStatus", machinev1alpha1.CProcessingRequestState)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if network.Status.State == machinev1alpha1.CFinishedRequestState ||
		network.Status.State == machinev1alpha1.CFailedRequestState {
		return ctrl.Result{}, nil
	}

	if network.Status.State == "" {
		network.Status.State = machinev1alpha1.CProcessingRequestState
		if err := r.Status().Update(ctx, network); err != nil {
			log.Error(err, "unable to update network resource status", "name", req.NamespacedName, "currentStatus", network.Status.State, "targetStatus", machinev1alpha1.CProcessingRequestState)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if network.Spec.Type == "" {
		log.Info("network does not specify type, nothing to do for now", "name", req.NamespacedName)
		network.Status.State = machinev1alpha1.CFinishedRequestState
		if err := r.Status().Update(ctx, network); err != nil {
			log.Error(err, "unable to update network status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	counterName, err := r.typeToCounterName(network.Spec.Type)
	if err != nil {
		log.Error(err, "unable to get counter name", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	counterNamespacedName := types.NamespacedName{
		Namespace: network.Namespace,
		Name:      counterName,
	}
	counter := machinev1alpha1.NetworkCounter{}
	err = r.Get(ctx, counterNamespacedName, &counter)
	if apierrors.IsNotFound(err) {
		counter.Name = counterName
		counter.Namespace = network.Namespace
		counter.Spec = *machinev1alpha1.NewNetworkCounterSpec(network.Spec.Type)
		if err := r.Create(ctx, &counter); err != nil {
			log.Error(err, "unable to create counter resource", "name", req.NamespacedName, "counter name", counterNamespacedName)
			return ctrl.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "unable to get counter resource", "name", req.NamespacedName, "counter name", counterNamespacedName)
		return ctrl.Result{}, err
	}

	var networkIdToReserve *machinev1alpha1.NetworkID
	if network.Spec.ID == nil {
		networkId, err := counter.Spec.Propose()
		if err != nil {
			network.Status.State = machinev1alpha1.CFailedRequestState
			network.Status.Message = err.Error()
			if err := r.Status().Update(ctx, network); err != nil {
				log.Error(err, "unable to update network status", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			log.Error(err, "unable to get network id", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		networkIdToReserve = networkId
	}

	if err := counter.Spec.Reserve(networkIdToReserve); err != nil {
		network.Status.State = machinev1alpha1.CFailedRequestState
		network.Status.Message = err.Error()
		if err := r.Status().Update(ctx, network); err != nil {
			log.Error(err, "unable to update network status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		log.Error(err, "unable to reserve network id", "name", req.NamespacedName, "network id", network.Spec.ID)
		return ctrl.Result{}, err
	}

	if err := r.Update(ctx, &counter); err != nil {
		log.Error(err, "unable to update counter state", "name", req.NamespacedName, "counter name", counterNamespacedName)
		return ctrl.Result{}, err
	}

	network.Status.State = machinev1alpha1.CFinishedRequestState
	network.Status.Reserved = networkIdToReserve
	if err := r.Status().Update(ctx, network); err != nil {
		log.Error(err, "unable to update network status", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) finalizeNetwork(ctx context.Context, log logr.Logger, network *machinev1alpha1.Network) error {
	if network.Spec.Type == "" {
		return nil
	}

	counterName, err := r.typeToCounterName(network.Spec.Type)
	if err != nil {
		return err
	}

	counterNamespacedName := types.NamespacedName{
		Namespace: network.Namespace,
		Name:      counterName,
	}

	counter := machinev1alpha1.NetworkCounter{}
	err = r.Get(ctx, counterNamespacedName, &counter)
	if apierrors.IsNotFound(err) {
		log.Error(err, "unable to find network counter, will let to remove finalizer and remove resource", "counter name", counterNamespacedName)
		return nil
	}
	if err != nil {
		log.Error(err, "unexpected error while retrieving a counter", "counter name", counterNamespacedName)
		return err
	}

	if network.Status.Reserved == nil {
		log.Info("id has not been booked, nothing to do")
		return nil
	}

	// For the cases of failure or external release
	if counter.Spec.CanReserve(network.Status.Reserved) {
		log.Info("id already released, will let to remove finalizer and remove resource", "counter name", counterNamespacedName)
		return nil
	}

	if err := counter.Spec.Release(network.Status.Reserved); err != nil {
		log.Error(err, "unexpected error while releasing ID", "counter name", counterNamespacedName)
		return err
	}

	if err := r.Update(ctx, &counter); err != nil {
		log.Error(err, "unexpected error while updating counter", "counter name", counterNamespacedName)
		return err
	}

	return nil
}

func (r *NetworkReconciler) typeToCounterName(networkType machinev1alpha1.NetworkType) (string, error) {
	counterName := ""
	switch networkType {
	case machinev1alpha1.CVXLANNetworkType:
		counterName = CVXLANCounterName
	case machinev1alpha1.CMPLSNetworkType:
		counterName = CMPLSCounterName
	default:
		return "", errors.Errorf("unsupported network type %s", networkType)
	}

	return counterName, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&machinev1alpha1.Network{}).
		Complete(r)
}
