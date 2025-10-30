// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IPFinalizer = "ip.ipam.metal.ironcore.dev/finalizer"

	IPReservationFailure = "IPReservationFailure"
	IPProposalFailure    = "IPProposalFailure"
	IPReservationSuccess = "IPReservationSuccess"
	IPReleaseSuccess     = "IPReleaseSuccess"

	IPFamilyLabelKey = "ip.ipam.metal.ironcore.dev/ip-family"
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
// +kubebuilder:rbac:groups=ipam.metal.ironcore.dev,resources=ips,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.metal.ironcore.dev,resources=ips/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.metal.ironcore.dev,resources=ips/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ip", req.NamespacedName)

	ip := &v1alpha1.IP{}
	err := r.Get(ctx, req.NamespacedName, ip)
	if apierrors.IsNotFound(err) {
		// object not found, it may have been deleted after the reconcile request.
		log.Info("Resource not found, it might have been deleted.")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err != nil {
		log.Error(err, "unable to get ip resource", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if ip.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(ip, IPFinalizer) {
			// Free IP on resource deletion
			if err := r.finalizeIP(ctx, log, ip); err != nil {
				log.Error(err, "unable to finalize ip resource", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(ip, IPFinalizer)
			err := r.Update(ctx, ip)
			if err != nil {
				log.Error(err, "unable to update ip resource on finalizer removal", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(ip, IPFinalizer) {
		controllerutil.AddFinalizer(ip, IPFinalizer)
		err = r.Update(ctx, ip)
		if err != nil {
			log.Error(err, "unable to update ip resource with finalizer", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if _, ok := ip.Labels[IPFamilyLabelKey]; !ok {
		subnet := &v1alpha1.Subnet{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: ip.Namespace,
			Name:      ip.Spec.Subnet.Name,
		}, subnet); err != nil {
			log.Error(err, "unable to get subnet resource", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}

		if ip.Labels == nil {
			ip.Labels = map[string]string{}
		}
		ip.Labels[IPFamilyLabelKey] = string(subnet.Status.Type)
		err = r.Update(ctx, ip)
		return ctrl.Result{}, err
	}

	if ip.Status.State == v1alpha1.IPStateAllocated ||
		ip.Status.State == v1alpha1.IPStateFailed {
		return ctrl.Result{}, nil
	}

	if ip.Status.State == "" {
		ip.Status.State = v1alpha1.IPStatePending
		ip.Status.Message = ""
		if err := r.Status().Update(ctx, ip); err != nil {
			log.Error(err, "unable to update ip resource status", "name", req.NamespacedName, "currentStatus", ip.Status.State, "targetStatus", v1alpha1.NetworkStatePending)
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
			ip.Status.State = v1alpha1.IPStateFailed
			ip.Status.Message = err.Error()
			if err := r.Status().Update(ctx, ip); err != nil {
				log.Error(err, "unable to update ip status", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			r.EventRecorder.Eventf(ip, v1.EventTypeWarning, IPProposalFailure, ip.Status.Message)
		}
		ipCidrToReserve = cidr
	}

	if err := subnet.Reserve(ipCidrToReserve); err != nil {
		ip.Status.State = v1alpha1.IPStateFailed
		ip.Status.Message = err.Error()
		if err := r.Status().Update(ctx, ip); err != nil {
			log.Error(err, "unable to update ip status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		r.EventRecorder.Eventf(ip, v1.EventTypeWarning, IPReservationFailure, ip.Status.Message)
		return ctrl.Result{}, err
	}

	if err := r.Status().Update(ctx, &subnet); err != nil {
		log.Error(err, "unable to update subnet status after ip reservation", "name", req.NamespacedName, "subnet name", subnetNamespacedName)
		return ctrl.Result{}, err
	}

	ip.Status.State = v1alpha1.IPStateAllocated
	ip.Status.Message = ""
	ip.Status.Reserved = ipCidrToReserve.AsIPAddr()
	if err := r.Status().Update(ctx, ip); err != nil {
		log.Error(err, "unable to update ip status after ip reservation", "name", req.NamespacedName, "subnet name", subnetNamespacedName)
		return ctrl.Result{}, err
	}
	r.EventRecorder.Eventf(ip, v1.EventTypeNormal, IPReservationSuccess, "IP %s reserved", ipCidrToReserve.String())

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

	r.EventRecorder.Eventf(ip, v1.EventTypeNormal, IPReleaseSuccess, "IP %s released", ipCidr.String())

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.EventRecorder = mgr.GetEventRecorderFor("ip-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.IP{}).
		Complete(r)
}
