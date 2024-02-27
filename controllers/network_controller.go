// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	machinev1alpha1 "github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
)

const (
	CNetworkFinalizer = "network.ipam.onmetal.de/finalizer"

	CNetworkIDProposalFailureReason    = "NetworkIDProposalFailure"
	CNetworkIDReservationFailureReason = "NetworkIDReservationFailure"
	CNetworkIDReservationSuccessReason = "NetworkIDReservationSuccess"
	CNetworkIDReleaseSuccessReason     = "NetworkIDReleaseSuccess"

	CFailedTopLevelSubnetIndexKey = "failedTopLevelSubnet"
)

// NetworkReconciler reconciles a Network object
type NetworkReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networkcounters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networkcounters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networkcounters/finalizers,verbs=update

// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=networks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
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

	if network.Status.State == machinev1alpha1.CFinishedNetworkState &&
		network.Status.Reserved == nil &&
		network.Spec.Type != "" {
		network.Status.State = machinev1alpha1.CProcessingNetworkState
		network.Status.Message = ""
		if err := r.Status().Update(ctx, network); err != nil {
			log.Error(err, "unable to update network resource status", "name", req.NamespacedName, "currentStatus", network.Status.State, "targetStatus", machinev1alpha1.CProcessingNetworkState)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if network.Status.State == machinev1alpha1.CFinishedNetworkState ||
		network.Status.State == machinev1alpha1.CFailedNetworkState {
		if err := r.requeueFailedSubnets(ctx, log, network); err != nil {
			log.Error(err, "unable to requeue top level subnets", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if network.Status.State == "" {
		network.Status.State = machinev1alpha1.CProcessingNetworkState
		network.Status.Message = ""
		if err := r.Status().Update(ctx, network); err != nil {
			log.Error(err, "unable to update network resource status", "name", req.NamespacedName, "currentStatus", network.Status.State, "targetStatus", machinev1alpha1.CProcessingNetworkState)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if network.Spec.Type == "" {
		log.Info("network does not specify type, nothing to do for now", "name", req.NamespacedName)
		network.Status.State = machinev1alpha1.CFinishedNetworkState
		network.Status.Message = ""
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

	networkIdToReserve := network.Spec.ID
	if networkIdToReserve == nil {
		networkId, err := counter.Spec.Propose()
		if err != nil {
			network.Status.State = machinev1alpha1.CFailedNetworkState
			network.Status.Message = err.Error()
			if err := r.Status().Update(ctx, network); err != nil {
				log.Error(err, "unable to update network status", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			r.EventRecorder.Event(network, v1.EventTypeWarning, CNetworkIDProposalFailureReason, network.Status.Message)
			log.Error(err, "unable to get network id", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		networkIdToReserve = networkId
	}

	if err := counter.Spec.Reserve(networkIdToReserve); err != nil {
		network.Status.State = machinev1alpha1.CFailedNetworkState
		network.Status.Message = err.Error()
		if err := r.Status().Update(ctx, network); err != nil {
			log.Error(err, "unable to update network status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		r.EventRecorder.Event(network, v1.EventTypeWarning, CNetworkIDReservationFailureReason, network.Status.Message)
		log.Error(err, "unable to reserve network id", "name", req.NamespacedName, "network id", network.Spec.ID)
		return ctrl.Result{}, err
	}

	if err := r.Update(ctx, &counter); err != nil {
		log.Error(err, "unable to update counter state", "name", req.NamespacedName, "counter name", counterNamespacedName)
		return ctrl.Result{}, err
	}
	r.EventRecorder.Eventf(network, v1.EventTypeNormal, CNetworkIDReservationSuccessReason, "ID %s for type %s reserved successfully", networkIdToReserve, network.Spec.Type)

	network.Status.State = machinev1alpha1.CFinishedNetworkState
	network.Status.Message = ""
	network.Status.Reserved = networkIdToReserve
	if err := r.Status().Update(ctx, network); err != nil {
		log.Error(err, "unable to update network status", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) requeueFailedSubnets(ctx context.Context, log logr.Logger, network *machinev1alpha1.Network) error {
	matchingFields := client.MatchingFields{
		CFailedTopLevelSubnetIndexKey: network.Name,
	}

	subnets := &machinev1alpha1.SubnetList{}
	if err := r.List(context.Background(), subnets, client.InNamespace(network.Namespace), matchingFields); err != nil {
		log.Error(err, "unable to get connected top level subnets", "name", types.NamespacedName{Namespace: network.Namespace, Name: network.Name})
		return err
	}

	for _, subnet := range subnets.Items {
		subnet.Status.State = machinev1alpha1.CProcessingSubnetState
		subnet.Status.Message = ""
		if err := r.Status().Update(ctx, &subnet); err != nil {
			log.Error(err, "unable to update top level subnet", "name", types.NamespacedName{Namespace: network.Namespace, Name: network.Name}, "subnet", subnet.Name)
			return err
		}
	}

	return nil
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
	r.EventRecorder.Eventf(network, v1.EventTypeNormal, CNetworkIDReleaseSuccessReason, "ID %s for type %s released successfully", network.Status.Reserved, network.Spec.Type)

	return nil
}

func (r *NetworkReconciler) typeToCounterName(networkType machinev1alpha1.NetworkType) (string, error) {
	counterName := ""
	switch networkType {
	case machinev1alpha1.CVXLANNetworkType:
		counterName = CVXLANCounterName
	case machinev1alpha1.CGENEVENetworkType:
		counterName = CGENEVECounterName
	case machinev1alpha1.CMPLSNetworkType:
		counterName = CMPLSCounterName
	default:
		return "", errors.Errorf("unsupported network type %s", networkType)
	}

	return counterName, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	createFailedSubnetIndexValue := func(object client.Object) []string {
		subnet, ok := object.(*machinev1alpha1.Subnet)
		if !ok {
			return nil
		}
		state := subnet.Status.State
		parentNet := subnet.Spec.Network.Name
		parentSubnet := subnet.Spec.ParentSubnet.Name
		if parentSubnet != "" {
			return nil
		}
		if state != machinev1alpha1.CFailedSubnetState {
			return nil
		}
		return []string{parentNet}
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &machinev1alpha1.Subnet{}, CFailedTopLevelSubnetIndexKey, createFailedSubnetIndexValue); err != nil {
		return err
	}

	r.EventRecorder = mgr.GetEventRecorderFor("network-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&machinev1alpha1.Network{}).
		Complete(r)
}
