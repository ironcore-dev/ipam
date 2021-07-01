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
	"fmt"

	"github.com/onmetal/ipam/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IpamFinalizer = "ipam.ipam.onmetal.de/finalizer"
)

// IpReconciler reconciles a Ip object
type IpReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=*,resources=*,verbs=get;list;watch
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=ips,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=ips/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.onmetal.de,resources=ips/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IpReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ip", req.NamespacedName)

	ip := &v1alpha1.Ip{}
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
		if controllerutil.ContainsFinalizer(ip, IpamFinalizer) {
			// Free IP on resource deletion
			if err := r.finalizeIp(ctx, ip); err != nil {
				log.Error(err, "unable to finalize ip resource", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(ip, IpamFinalizer)
			err := r.Update(ctx, ip)
			if err != nil {
				log.Error(err, "unable to update ip resource on finalizer removal", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	if !controllerutil.ContainsFinalizer(ip, IpamFinalizer) {
		controllerutil.AddFinalizer(ip, IpamFinalizer)
		err = r.Update(ctx, ip)
		if err != nil {
			log.Error(err, "unable to update ip resource with finalizer", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Resource created
	if ip.Status.LastUsedIP == nil {
		subnet, err := r.findSubnet(ctx, ip)
		if err != nil {
			log.Error(err, "unable to find ip subnet", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		newCidr, err := ip.Spec.IP.AsCidr()
		if err != nil {
			log.Error(err, "unable to get ip as cidr", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		// Occupy new IP
		err = subnet.Reserve(newCidr)
		if err != nil {
			log.Error(err, "unable to reserve IP", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		if err := r.Status().Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet state", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		ip.Status.LastUsedIP = ip.Spec.IP
		if err := r.Update(ctx, ip); err != nil {
			log.Error(err, "unable to update ip state", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		// Resource was updated - e.g. IP changed
	} else if !ip.Status.LastUsedIP.Net.Equal(ip.Spec.IP.Net) {
		// Free old IP
		subnet, err := r.findSubnet(ctx, ip)
		if err != nil {
			log.Error(err, "unable to find ip subnet", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		lastCidr, err := ip.Status.LastUsedIP.AsCidr()
		if err != nil {
			log.Error(err, "unable to get ip as cidr", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		err = subnet.Release(lastCidr)
		if err != nil {
			log.Error(err, "unable to release IP", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		// Occupy new IP
		newCidr, err := ip.Spec.IP.AsCidr()
		if err != nil {
			log.Error(err, "unable to get ip as cidr", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		err = subnet.Reserve(newCidr)
		if err != nil {
			log.Error(err, "unable to reserve IP", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		if err := r.Status().Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet state", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		ip.Status.LastUsedIP = ip.Spec.IP
		if err := r.Update(ctx, ip); err != nil {
			log.Error(err, "unable to update ip state", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *IpReconciler) finalizeIp(ctx context.Context, ipam *v1alpha1.Ip) error {
	// Free subnet IP
	subnet, err := r.findSubnet(ctx, ipam)
	if err != nil {
		return fmt.Errorf("unable to find ipam subnet: %w", err)
	}
	ipCidr, err := ipam.Spec.IP.AsCidr()
	if err != nil {
		return fmt.Errorf("unable to get ip as cidr: %w", err)
	}
	err = subnet.Release(ipCidr)
	if err != nil {
		return fmt.Errorf("unable to release IP: %w", err)
	}
	if err := r.Status().Update(ctx, subnet); err != nil {
		return fmt.Errorf("\"unable to update subnet state: %w", err)
	}
	return nil
}

func (r *IpReconciler) findSubnet(ctx context.Context, ipam *v1alpha1.Ip) (*v1alpha1.Subnet, error) {
	subnet := &v1alpha1.Subnet{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: ipam.Namespace, Name: ipam.Spec.Subnet}, subnet); err != nil {
		return nil, fmt.Errorf("unable to get gateway of Subnet: %w", err)
	}
	return subnet, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IpReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Ip{}).
		Complete(r)
}
