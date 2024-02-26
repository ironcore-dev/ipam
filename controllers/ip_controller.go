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

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/ironcore-dev/ipam/api/v1alpha1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CIPFinalizer = "ip.ipam.onmetal.de/finalizer"

	CIPReservationFailureReason = "IPReservationFailure"
	CIPProposalFailureReason    = "IPProposalFailure"
	CIPReservationSuccessReason = "IPReservationSuccess"
	CIPReleaseSuccessReason     = "IPReleaseSuccess"
)

// IPReconciler reconciles a Ip object
type IPReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

// +kubebuilder:rbac:groups=*,resources=*,verbs=get;list;watch
// +kubebuilder:rbac:groups=*,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=ips,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=ips/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=ips/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ip", req.NamespacedName)

	ip := &v1alpha1.IP{}
	err := r.Get(ctx, req.NamespacedName, ip)
	if apierrors.IsNotFound(err) {
		log.Error(err, "requested ip resource not found", "name", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err != nil {
		log.Error(err, "unable to get ip resource", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if ip.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(ip, CIPFinalizer) {
			// Free IP on resource deletion
			if err := r.finalizeIP(ctx, log, ip); err != nil {
				log.Error(err, "unable to finalize ip resource", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(ip, CIPFinalizer)
			err := r.Update(ctx, ip)
			if err != nil {
				log.Error(err, "unable to update ip resource on finalizer removal", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(ip, CIPFinalizer) {
		controllerutil.AddFinalizer(ip, CIPFinalizer)
		err = r.Update(ctx, ip)
		if err != nil {
			log.Error(err, "unable to update ip resource with finalizer", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if ip.Status.State == v1alpha1.CFinishedIPState ||
		ip.Status.State == v1alpha1.CFailedIPState {
		return ctrl.Result{}, nil
	}

	if ip.Status.State == "" {
		ip.Status.State = v1alpha1.CProcessingIPState
		ip.Status.Message = ""
		if err := r.Status().Update(ctx, ip); err != nil {
			log.Error(err, "unable to update ip resource status", "name", req.NamespacedName, "currentStatus", ip.Status.State, "targetStatus", v1alpha1.CProcessingNetworkState)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	subnetNamespacedName := types.NamespacedName{
		Namespace: ip.Namespace,
		Name:      ip.Spec.Subnet.Name,
	}
	subnet := v1alpha1.Subnet{}
	if err = r.Get(ctx, subnetNamespacedName, &subnet); err != nil {
		log.Error(err, "unable to get subnet resource", "name", req.NamespacedName, "subnet name", subnetNamespacedName)
		return ctrl.Result{}, err
	}

	var ipCidrToReserve *v1alpha1.CIDR
	if ip.Spec.IP != nil {
		ipCidrToReserve = ip.Spec.IP.AsCidr()
	} else {
		cidr, err := subnet.ProposeForCapacity(resource.NewScaledQuantity(1, 0))
		if err != nil {
			ip.Status.State = v1alpha1.CFailedIPState
			ip.Status.Message = err.Error()
			if err := r.Status().Update(ctx, ip); err != nil {
				log.Error(err, "unable to update ip status", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			r.EventRecorder.Eventf(ip, v1.EventTypeWarning, CIPProposalFailureReason, ip.Status.Message)
		}
		ipCidrToReserve = cidr
	}

	if err := subnet.Reserve(ipCidrToReserve); err != nil {
		ip.Status.State = v1alpha1.CFailedIPState
		ip.Status.Message = err.Error()
		if err := r.Status().Update(ctx, ip); err != nil {
			log.Error(err, "unable to update ip status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		r.EventRecorder.Eventf(ip, v1.EventTypeWarning, CIPReservationFailureReason, ip.Status.Message)
		return ctrl.Result{}, err
	}

	if err := r.Status().Update(ctx, &subnet); err != nil {
		log.Error(err, "unable to update subnet status after ip reservation", "name", req.NamespacedName, "subnet name", subnetNamespacedName)
		return ctrl.Result{}, err
	}

	ip.Status.State = v1alpha1.CFinishedIPState
	ip.Status.Message = ""
	ip.Status.Reserved = ipCidrToReserve.AsIPAddr()
	if err := r.Status().Update(ctx, ip); err != nil {
		log.Error(err, "unable to update ip status after ip reservation", "name", req.NamespacedName, "subnet name", subnetNamespacedName)
		return ctrl.Result{}, err
	}
	r.EventRecorder.Eventf(ip, v1.EventTypeNormal, CIPReservationSuccessReason, "IP %s reserved", ipCidrToReserve.String())

	return ctrl.Result{}, nil
}

func (r *IPReconciler) finalizeIP(ctx context.Context, log logr.Logger, ip *v1alpha1.IP) error {
	if ip.Status.Reserved == nil {
		log.Info("IP has not been reserved, will release")
		return nil
	}

	subnetNamespacedName := types.NamespacedName{
		Namespace: ip.Namespace,
		Name:      ip.Spec.Subnet.Name,
	}
	subnet := v1alpha1.Subnet{}
	err := r.Get(ctx, subnetNamespacedName, &subnet)
	if apierrors.IsNotFound(err) {
		log.Error(err, "unable to find subnet, will release the IP address", "subnet name", subnetNamespacedName)
		return nil
	}
	if err != nil {
		log.Error(err, "unexpected error while retrieving subnet", "subnet name", subnetNamespacedName)
		return err
	}

	ipCidr := ip.Status.Reserved.AsCidr()

	if subnet.CanReserve(ipCidr) {
		log.Info("IP already released, will let to remove finalizer and remove resource", "subnet name", subnetNamespacedName)
		return nil
	}

	if err := subnet.Release(ipCidr); err != nil {
		log.Error(err, "unexpected error while releasing IP", "subnet name", subnetNamespacedName)
		return err
	}

	if err := r.Status().Update(ctx, &subnet); err != nil {
		log.Error(err, "unexpected error while updating subnet", "subnet name", subnetNamespacedName)
		return err
	}

	r.EventRecorder.Eventf(ip, v1.EventTypeNormal, CIPReleaseSuccessReason, "IP %s released", ipCidr.String())

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.EventRecorder = mgr.GetEventRecorderFor("ip-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.IP{}).
		Complete(r)
}
