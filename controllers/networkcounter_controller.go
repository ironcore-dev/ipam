// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	machinev1alpha1 "github.com/onmetal/ipam/api/ipam/v1alpha1"
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

	nc := &machinev1alpha1.NetworkCounter{}
	err := r.Get(ctx, req.NamespacedName, nc)
	if apierrors.IsNotFound(err) {
		log.Error(err, "requested machine resource not found", "name", req.NamespacedName)
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

	networks := &machinev1alpha1.NetworkList{}
	if err := r.List(context.Background(), networks, client.InNamespace(req.Namespace), matchingFields); err != nil {
		log.Error(err, "unable to get connected networks", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	for _, network := range networks.Items {
		network.Status.State = machinev1alpha1.CProcessingNetworkState
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
		net, ok := object.(*machinev1alpha1.Network)
		if !ok {
			return nil
		}
		state := net.Status.State
		netType := net.Spec.Type
		if netType == "" {
			return nil
		}
		if state != machinev1alpha1.CFailedNetworkState {
			return nil
		}
		return []string{string(netType)}
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &machinev1alpha1.Network{}, CFailedNetworkOfTypeIndexKey, createFailedNetworkOfTypeIndexValue); err != nil {
		return err
	}

	r.EventRecorder = mgr.GetEventRecorderFor("networkcounter-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&machinev1alpha1.NetworkCounter{}).
		Complete(r)
}

func (r *NetworkCounterReconciler) counterNameToType(name string) (machinev1alpha1.NetworkType, error) {
	var counterType machinev1alpha1.NetworkType
	switch name {
	case CVXLANCounterName:
		counterType = machinev1alpha1.CVXLANNetworkType
	case CGENEVECounterName:
		counterType = machinev1alpha1.CGENEVENetworkType
	case CMPLSCounterName:
		counterType = machinev1alpha1.CMPLSNetworkType
	default:
		return "", errors.Errorf("unknown network counter %s", name)
	}

	return counterType, nil
}
