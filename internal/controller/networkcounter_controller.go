// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
)

const (
	CVXLANCounterName  = "k8s-vxlan-network-counter"
	CGENEVECounterName = "k8s-geneve-network-counter"
	CMPLSCounterName   = "k8s-mpls-network-counter"

	CFailedNetworkOfTypeIndexKey = "failedNetworkOfType"
)

// NetworkCounterReconciler reconciles a NetworkCounter object
type NetworkCounterReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

func (r *NetworkCounterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("machine", req.NamespacedName)

	nc := &v1alpha1.NetworkCounter{}
	err := r.Get(ctx, req.NamespacedName, nc)
	if apierrors.IsNotFound(err) {
		log.Info("Resource not found, it might have been deleted.")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err != nil {
		log.Error(err, "unable to get machine resource", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if nc.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	netType, err := r.counterNameToType(nc.Name)
	if err != nil {
		log.Error(err, "unknown network counter", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	matchingFields := client.MatchingFields{
		CFailedNetworkOfTypeIndexKey: string(netType),
	}

	networks := &v1alpha1.NetworkList{}
	if err := r.List(context.Background(), networks, client.InNamespace(req.Namespace), matchingFields); err != nil {
		log.Error(err, "unable to get connected networks", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	for _, network := range networks.Items {
		network.Status.State = v1alpha1.CProcessingNetworkState
		network.Status.Message = ""
		if err := r.Status().Update(ctx, &network); err != nil {
			log.Error(err, "unable to update network", "name", req.NamespacedName, "network", network.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkCounterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	createFailedNetworkOfTypeIndexValue := func(object client.Object) []string {
		net, ok := object.(*v1alpha1.Network)
		if !ok {
			return nil
		}
		state := net.Status.State
		netType := net.Spec.Type
		if netType == "" {
			return nil
		}
		if state != v1alpha1.CFailedNetworkState {
			return nil
		}
		return []string{string(netType)}
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &v1alpha1.Network{}, CFailedNetworkOfTypeIndexKey, createFailedNetworkOfTypeIndexValue); err != nil {
		return err
	}

	r.EventRecorder = mgr.GetEventRecorderFor("networkcounter-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.NetworkCounter{}).
		Complete(r)
}

func (r *NetworkCounterReconciler) counterNameToType(name string) (v1alpha1.NetworkType, error) {
	var counterType v1alpha1.NetworkType
	switch name {
	case CVXLANCounterName:
		counterType = v1alpha1.VXLANNetworkType
	case CGENEVECounterName:
		counterType = v1alpha1.GENEVENetworkType
	case CMPLSCounterName:
		counterType = v1alpha1.MPLSNetworkType
	default:
		return "", errors.Errorf("unknown network counter %s", name)
	}

	return counterType, nil
}
